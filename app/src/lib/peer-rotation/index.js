'use strict';

/**
 * Manual slot fill: one pass every ten minutes (every 15 seconds while a
 * candidate is on trial), and the toggle that decides
 * whether any of it runs at all. (The directory keeps its old name,
 * peer-rotation, so every existing require keeps resolving; in the app it is
 * no longer a rotation. It fills free manual slots and spreads them, and
 * never moves a working peer out.)
 *
 * What lives where:
 *
 *   rules.js     who may lose a slot when a person adds a peer to a full
 *                set, and how long an offline peer keeps one. No database,
 *                no RPC.
 *   spread.js    where the manual peers are, and which region a free slot
 *                should go to.
 *   addrbook.js  candidates from Core's address book, the networks they may
 *                come from, and Core's bans.
 *   fill.js      the trial: a candidate goes into a free slot, and Core
 *                connects to it or it goes again.
 *   retire.js    frees the slot of a manual peer that went dark.
 *   log.js       the audit trail: rotation_log.
 */

const config = require('../config');
const db = require('../db');
const rpc = require('../rpc');
const queries = require('../queries');
const logger = require('../logger').make('peer-rotation');
const { logAction, recentLog, recentLogWithTries, triesSinceFree } = require('./log');
const { retireOfflineManualPeers } = require('./retire');
const fill = require('./fill');
const { offlineGraceMs, offlineForMs } = require('./rules');
const clock = require('./clock');
const peerSync = require('../peer-sync');
const peersService = require('../peers-service');
const { addressKey } = require('../address');

const keyOfAddress = (address) => addressKey(address, config.bitcoin.network);

const META_KEY = 'peer_rotation_enabled';
// A pass every ten minutes (peer-profiler.js), every 15 seconds while a
// candidate is on trial (watchTrial): a peer Core reached is in for good
// within seconds, and the next candidate goes in as soon as one is out
// (15.11.2026, it was a minute).
const PASS_EVERY_MS = 10 * 60 * 1000;
const TRIAL_CHECK_MS = 15 * 1000;
// How long the page mentions a candidate that went again.
const LAST_TRY_SHOWN_MS = 30 * 60 * 1000;

// What the last pass saw and decided, for the line on the page that says what
// Slot Fill is doing. Written by the pass in the profiler, read by the
// dashboard, so it lives in the database. Nothing here asks Core anything
// the pass did not ask anyway.
const STATUS_KEY = 'slot_fill_status';
function writeStatus(status) {
  db.instance
    .prepare(`INSERT INTO meta (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`)
    .run(STATUS_KEY, JSON.stringify(status));
}
function readStatus() {
  try {
    const row = db.instance.prepare('SELECT value FROM meta WHERE key = ?').get(STATUS_KEY);
    return row ? JSON.parse(row.value) : null;
  } catch {
    return null;
  }
}

// The manual set as the page tells it: how full, and of the peers that are
// offline, which keeps its slot (locked, and Core has held it before) and when
// each other one gives its slot back.
// The ranking as a pass must count it: a manual peer removed a moment ago
// still has its live session in the profiler's last snapshot (up to 15 s
// old), and counted, the pass a removal by hand asks for found every slot in
// use and filled none until ten minutes later (15.11.2026). Its row is gone;
// only the session made it manual.
function freshRanking() {
  const gone = peerSync.recentRemovals();
  if (!Object.keys(gone).length) return queries.peerRanking();
  return queries.peerRanking().map((p) => (p.trusted && p.trustedSince == null && gone[keyOfAddress(p.address)]
    ? { ...p, trusted: false } : p));
}

function slotsOf(ranking, now) {
  const trusted = ranking.filter((p) => p.trusted);
  const offline = trusted.filter((p) => !p.live).map((p) => {
    const where = p.where ? p.where.regionName : null;
    if (p.kept && p.everManual) return { address: p.address, where, keeps: true };
    const off = offlineForMs(p, now, clock.lostBetween);
    return { address: p.address, where, freesInMs: off == null ? null : Math.max(0, offlineGraceMs(p.firstPct) - off) };
  });
  return { used: trusted.length, max: config.maxManualPeers, offline };
}

function isEnabled() {
  const row = db.instance.prepare(`SELECT value FROM meta WHERE key = ?`).get(META_KEY);
  return row ? row.value === '1' : false;
}

// When it was last switched on: the first pass after that runs at once, and
// only once (waitsForFirstPass).
const ON_AT_KEY = 'slot_fill_on_at';

function setEnabled(enabled) {
  const put = db.instance
    .prepare(`INSERT INTO meta (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`);
  put.run(META_KEY, enabled ? '1' : '0');
  // Unique per switching on, even two in the same millisecond.
  if (enabled) put.run(ON_AT_KEY, `${Date.now()}-${Math.random().toString(36).slice(2)}`);
  // Switched off, its log goes with it (15.11.2026): switched on again it
  // starts empty, as the list of tries does. What it held against candidates
  // (slot_fill_tried) stays, or the same bad ones would come straight back.
  if (!enabled) db.instance.prepare('DELETE FROM rotation_log').run();
  // What a pass saw before the switch is not what Slot Fill is doing now.
  db.instance.prepare('DELETE FROM meta WHERE key = ?').run(STATUS_KEY);
  db.instance.prepare("DELETE FROM meta WHERE key = 'slot_fill_last_try'").run();
  logger.info('manual slot fill toggled', { enabled: Boolean(enabled) });
}

/**
 * Switched off, nothing is left half done: a candidate on trial is kept if
 * Core already holds a working connection to it, and otherwise taken out
 * again.
 */
async function settleOnSwitchOff() {
  if (!fill.trials().length) return;
  try {
    const [peers, chain] = await Promise.all([rpc.getPeerInfo(), rpc.getBlockchainInfo()]);
    await fill.settle(peers, chain, Date.now(), { force: true });
  } catch (err) {
    // Core not answering: the next time the slot fill runs, it settles then.
    logger.warn('slot fill: could not settle the trial on switching off', { error: err.message });
  }
}

// One pass at a time: the ten-minute timer and the 15-second trial check
// (watchTrial) both come here, and two passes at once could each put a
// candidate on trial.
let running = false;

async function tick({ quick = false } = {}) {
  if (running) return { enabled: isEnabled(), retired: 0, started: 0, confirmed: 0, reverted: 0 };
  running = true;
  try {
    return await pass({ quick });
  } finally {
    running = false;
  }
}

// A slot freed by hand (the Remove button) is filled now, not at the next
// ten-minute pass: until then the card said "all slots taken" over a free
// one (13.11.2026). The dashboard only asks; the pass runs in the profiler
// at its next poll, seconds away, behind the one `running` above - a pass
// in the dashboard's own process could run beside the profiler's and put
// two candidates on trial. The peer removed is held against for a week, as
// one Core could not reach: removed by hand, it should not come straight
// back as the next candidate.
const WANTED_KEY = 'slot_fill_wanted';
function freedByHand(address) {
  fill.markTried(address, false, Date.now(), { reason: 'removed' });
  if (!isEnabled()) return;
  db.instance.prepare('INSERT INTO meta (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value')
    .run(WANTED_KEY, String(Date.now()));
}
// Asked by the profiler at each poll. A pass under way keeps the request for
// the next poll: it may have counted the slot before it was freed.
function wantsPass() {
  if (running) return false;
  const row = db.instance.prepare('SELECT value FROM meta WHERE key = ?').get(WANTED_KEY);
  if (!row) return false;
  db.instance.prepare('DELETE FROM meta WHERE key = ?').run(WANTED_KEY);
  return isEnabled();
}

/**
 * Every 15 seconds, from peer-profiler.js: only while a candidate is on trial, so
 * a candidate Core cannot use is out at its own deadline, not at the next
 * ten-minute pass, and the next one goes in at once. Without a trial, and
 * switched off, it does nothing at all.
 */
// A peer Slot Fill added in the last half hour gets the peer list's verdict
// once a minute (fill.dropUnmasked): its behaviour rules need ten minutes,
// and at the ten-minute pass a Disguised peer could sit in its slot twenty.
const ADDED_CHECK_MS = 60 * 1000;
let addedCheckAt = 0;
async function watchTrial(now = Date.now()) {
  if (!isEnabled()) return null;
  if (fill.trials().length) return tick({ quick: true });
  if (!fill.recentlyAdded(now) || now - addedCheckAt < ADDED_CHECK_MS) return null;
  addedCheckAt = now;
  return tick({ quick: true });
}

/**
 * One pass - a complete no-op unless the toggle is on.
 *
 *   settle  keeps the candidate on trial if Core holds a working connection
 *           to it, or takes it out once the trial is over.
 *   retire  frees the slot of a manual peer that has not been there for
 *           longer than its record buys it - never the candidate still on
 *           trial, which is the trial's to judge.
 *   start   puts candidates into the free slots, up to fill.MAX_TRIALS on
 *           trial at once.
 *
 * All read the same getpeerinfo, taken first. If Core does not answer,
 * nothing happens: every peer would look offline.
 */
// `quick`: the 15-second check while trials run. It judges them; a new
// candidate goes in only when one of them ended, not every 15 seconds.
async function pass({ quick = false } = {}) {
  const idle = { enabled: false, retired: 0, started: 0, confirmed: 0, reverted: 0 };
  if (!isEnabled()) {
    clock.idle();
    // Switched off with a trial still open - Core did not answer at that
    // moment: it is settled here, the next time Core does.
    if (fill.trials().length) await settleOnSwitchOff();
    return idle;
  }
  let peers;
  let chain;
  try {
    [peers, chain] = await Promise.all([rpc.getPeerInfo(), rpc.getBlockchainInfo()]);
  } catch (err) {
    logger.debug('slot fill: Core not answering, skipping this pass', { error: err.message });
    clock.coreDown();
    writeStatus({ at: Date.now(), nextAt: nextCheckAt(fill.trials().length > 0, Date.now()), state: 'core' });
    return { ...idle, enabled: true };
  }
  // Core was down since the last pass, or the clock stepped: the trials
  // start over and nobody is retired this pass (clock.js).
  const gap = clock.passed();
  if (gap) fill.restart(chain);
  // The peer list's word on each peer, as last fetched (at most a minute old).
  const kinds = peersService.peerKinds();
  const settled = await fill.settle(peers, chain, Date.now(), { kinds });
  const counted = { confirmed: settled.confirmed, reverted: settled.reverted };
  // A candidate taken out: the next one goes in this pass, on a fresh
  // getpeerinfo without the ones just dropped. If Core does not answer now,
  // the next pass does it.
  if (settled.gone.length) {
    try {
      peers = settled.gone.reduce((list, address) => fill.withoutSession(list, address), await rpc.getPeerInfo());
    } catch (err) {
      logger.debug('slot fill: Core not answering after a candidate went, next pass', { error: err.message });
      clock.coreDown();
      writeStatus({ at: Date.now(), nextAt: nextCheckAt(false, Date.now()), state: 'core' });
      return { ...idle, enabled: true, ...counted };
    }
  }
  // Switched off while this pass waited for Core: the switch has been
  // answered already, so nothing more is removed or added from here on.
  if (!isEnabled()) return { ...idle, enabled: false, ...counted };
  // Added earlier and found out since: the slot is free again.
  const unmasked = await fill.dropUnmasked(kinds, peers, isEnabled);
  if (unmasked.length) {
    try {
      peers = unmasked.reduce((list, address) => fill.withoutSession(list, address), await rpc.getPeerInfo());
    } catch (err) {
      logger.debug('slot fill: Core not answering after a peer went, next pass', { error: err.message });
      clock.coreDown();
      return { ...idle, enabled: true, ...counted };
    }
  }
  const onTrial = new Set(fill.trials().map((t) => t.address));
  const ranking = freshRanking();
  const retired = gap ? 0 : await retireOfflineManualPeers(ranking, onTrial, peers, isEnabled);
  // Switched off while the retiring waited for Core: no candidate goes in.
  if (!isEnabled()) {
    return { ...idle, enabled: false, retired, ...counted };
  }
  // Into the free slots, beside the candidates still on trial.
  const ended = settled.confirmed + settled.reverted + settled.left + unmasked.length;
  let started = quick && !ended && !retired ? [] : await fill.start(retired > 0 ? freshRanking() : ranking, peers, chain);
  // Switched off while this pass was adding: the switch-off could not see the
  // rows yet, so the candidates are taken out here, rather than left in slots
  // no later pass would look at.
  if (started.length && !isEnabled()) {
    for (const address of started) await fill.abandon(address);
    started = [];
  }
  if (isEnabled()) reportPass(peers, chain);
  return { enabled: true, retired, started: started.length, ...counted };
}

// Switched on and not yet checked (switching on clears the status): the
// profiler runs the first pass at its next poll, seconds away. Once per
// switching on: a pass that fails before it writes a status is not tried
// again every 15 seconds, it waits for the ten-minute pass.
let firstPassFor;
function waitsForFirstPass() {
  // A pass still under way (the timer's, from before the switch) would make
  // tick() return without one; the switching on is then asked for next poll.
  if (running || !isEnabled() || readStatus() != null) return false;
  const row = db.instance.prepare('SELECT value FROM meta WHERE key = ?').get(ON_AT_KEY);
  const on = row ? row.value : null;
  if (firstPassFor === on) return false;
  firstPassFor = on;
  return true;
}

// When the profiler's timers fire next (peer-profiler.js), so the page's
// "trying again in N min" says when that really is, not ten minutes from
// a pass the timer did not start.
const next = { passAt: 0, trialAt: 0 };
function scheduled({ passAt, trialAt }) {
  if (passAt) next.passAt = passAt;
  if (trialAt) next.trialAt = trialAt;
}
function nextCheckAt(trial, now) {
  const at = trial ? next.trialAt : next.passAt;
  return at > now ? at : now + (trial ? TRIAL_CHECK_MS : PASS_EVERY_MS);
}

// The status after a pass: a candidate on trial, the set full, or why no
// candidate went into a free slot.
function reportPass(peers, chain) {
  const now = Date.now();
  const trials = fill.trials();
  const trial = trials[0] || null;
  const slots = slotsOf(freshRanking(), now);
  const base = { at: now, nextAt: nextCheckAt(Boolean(trial), now), slots };
  // The candidate before, when it went again lately: the page's top row says
  // why, under what is going on now.
  const last = fill.lastTry();
  if (last && now - last.at < LAST_TRY_SHOWN_MS) base.lastTry = last;
  // What there is to pick from, as of the last look at Core's address book.
  const book = fill.lastBook();
  if (book) base.book = book;
  // Regions the reader picked, and whether they still have a candidate.
  const picks = fill.lastPicks();
  if (picks) base.picks = picks;
  // Every candidate tried since the slot came free, under the top row.
  const tries = slots.used < slots.max || trial ? triesSinceFree() : null;
  if (tries) base.tries = tries;
  // All candidates on trial, oldest first; `trial` the oldest, as before.
  if (trial) {
    const views = trials.map((t) => fill.trialView(t, peers, now));
    return writeStatus({ ...base, state: 'trial', trial: views[0], trials: views });
  }
  if (chain.initialblockdownload) return writeStatus({ ...base, state: 'catching_up' });
  if (slots.used >= slots.max) return writeStatus({ ...base, state: 'full' });
  const look = fill.lastLook() || { why: 'none_fit' };
  return writeStatus({ ...base, state: 'looking', look });
}

module.exports = {
  PASS_EVERY_MS,
  TRIAL_CHECK_MS,
  readStatus,
  waitsForFirstPass,
  scheduled,
  reportPassForTest: reportPass,
  isEnabled,
  setEnabled,
  settleOnSwitchOff,
  watchTrial,
  logAction,
  recentLog,
  recentLogWithTries,
  retireOfflineManualPeers,
  offlineGraceMs,
  tick,
  freedByHand,
  wantsPass,
};
