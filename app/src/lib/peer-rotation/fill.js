'use strict';

/**
 * Manual slot fill: which address gets a free manual slot, and the trial that
 * decides whether it keeps it.
 *
 * Only free slots. A working manual peer is never given up for a candidate
 * nobody has seen work yet; a slot comes free when a peer stays offline past
 * its grace (retire.js) or a person removes one, and then a peer from a region
 * the set is missing gets it first (spread.js).
 *
 * Up to MAX_TRIALS candidates at a time, each from another region and
 * provider (15.11.2026: one at a time, eight free slots took over an hour).
 * Each goes in with addnode, and the slot fill waits for Core: within its time (NO_CONNECT_MS, TRIAL_MS, HEADERS_OPEN_MS) Core must hold a manual connection to it, to a
 * node that says it serves blocks and segwit and has announced our chain's
 * tip. Then it stays. If not, it is taken out again and the next one tried.
 * That is a check that Core reached a node on this chain, not that the node
 * will relay blocks well - a free slot loses nothing by the try.
 *
 * Why Core decides, and not a probe of this app's own: a probe from this
 * container is not a Bitcoin node's handshake, so the peer could tell this
 * app was knocking, and it went out past Core's proxy and onlynet settings.
 *
 * Core's own connections are never touched. Candidates are addresses Core is
 * not connected to, peer-sync is told not to disconnect anything for them
 * (neverDisconnect), and taking a candidate out again drops only its manual
 * session, by Core's peer id.
 */

const config = require('../config');
const db = require('../db');
const rpc = require('../rpc');
const peerSync = require('../peer-sync');
const logger = require('../logger').make('peer-rotation');
const { logAction } = require('./log');
const { addressKey, hostFromAddress, unreachableNetwork } = require('../address');
const spread = require('./spread');
const { pickedRegions } = require('./picks');
const { bookEntries, usableNetworks, offeredNetworks, bannedMatcher, servesBlocks } = require('./addrbook');
const { otherChainOf } = require('../peer-rules');

const TRIAL_KEY = 'slot_fill_trial';
// Candidates on trial at once. Each holds a slot while it runs, so never more
// than there are free slots; three keeps a burst of addnodes small.
const MAX_TRIALS = 3;
// Passes run every ten minutes; while a trial is open a check runs every
// 15 seconds (peer-profiler.js), so these are the real waits.
//
// No connection at all by then: the candidate is out. Core's added-node
// thread takes a new addnode at its next round, at most a minute away, and
// shows a clearnet peer in getpeerinfo as soon as the TCP connection stands
// (5 s to connect); a peer that answers is there well inside two minutes. It was five, and on
// 15.11.2026 four South American candidates in a row took twenty minutes to
// fail. Tor adds a circuit (Winnow's census, 7-9 November 2026: 11.7 s
// median, 51 s the longest success); I2P goes through SAM, whose budget is
// minutes, but the slot fill only starts an I2P candidate once Core holds an
// I2P connection, so the session is up.
const NO_CONNECT_MS = { clearnet: 2 * 60 * 1000, Tor: 5 * 60 * 1000, I2P: 8 * 60 * 1000 };
// Connected, but not on our tip (or serving no blocks), counted from the
// connection: long enough for the peer to answer the headers Core asks every
// new peer for, which a peer on our chain does within seconds. It was a
// quarter of an hour; on 14.11.2026 a candidate connected one minute in,
// stayed below our tip, and held the slot fourteen more minutes for nothing.
const TRIAL_MS = 3 * 60 * 1000;
// Connected, serves blocks, but Core has no header height for it yet
// (synced_headers -1). A peer one or two blocks behind our tip stays there
// until it announces a block, and the next one may be half an hour away:
// that is no verdict, so it gets this long.
const HEADERS_OPEN_MS = 30 * 60 * 1000;
// An address-book address Core could not connect to is left alone this long.
// One that could not be reached at all only a day (15.11.2026): a home node
// may just have been off for the night. One that answered and was wrong - not
// the Core it names, another chain, no blocks - does not change overnight.
const FAILED_RETRY_MS = 7 * 24 * 60 * 60 * 1000;
const UNREACHED_RETRY_MS = 24 * 60 * 60 * 1000;
const TRIED_KEEP_MS = 30 * 24 * 60 * 60 * 1000;
// Adds refused before Core was even asked (already connected, say) per pass.
const ATTEMPTS_PER_PASS = 3;
// A trial this young may still be in the middle of its add, in the other
// process: no row yet is not the same as a row someone took out.
const ADDING_MS = 60 * 1000;

const keyOf = (address) => addressKey(address, config.bitcoin.network);
let regionNames = null;
function countryName(code) {
  try {
    regionNames = regionNames || new Intl.DisplayNames(['en'], { type: 'region' });
    return regionNames.of(code) || code;
  } catch {
    return code;
  }
}

// The candidates on trial, oldest first, as one JSON list in meta. A single
// object is what versions with one trial at a time left behind.
function trials() {
  try {
    const row = db.instance.prepare('SELECT value FROM meta WHERE key = ?').get(TRIAL_KEY);
    if (!row) return [];
    const v = JSON.parse(row.value);
    return (Array.isArray(v) ? v : [v]).filter((t) => t && t.address);
  } catch {
    return [];
  }
}

function writeTrials(list) {
  if (!list.length) {
    db.instance.prepare('DELETE FROM meta WHERE key = ?').run(TRIAL_KEY);
    return;
  }
  db.instance
    .prepare('INSERT INTO meta (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value')
    .run(TRIAL_KEY, JSON.stringify(list));
}

// Read and written in one transaction: the dashboard (switching off) and the
// profiler (a pass) change the list from two processes.
const changeTrials = (fn) => db.instance.transaction(() => writeTrials(fn(trials())))();
const addTrial = (trial) => changeTrials((list) => [...list.filter((t) => t.address !== trial.address), trial]);
const dropTrial = (address) => changeTrials((list) => list.filter((t) => t.address !== address));
const replaceTrial = (trial) => changeTrials((list) => list.map((t) => (t.address === trial.address ? trial : t)));

// The oldest candidate on trial, or null.
function currentTrial() {
  return trials()[0] || null;
}

// The last trial of an address: how it ended and how long it ran, kept so a
// restart does not take the reason with it (the app log is all that held it).
function markTried(address, ok, now, { reason = null, startedAt = null } = {}) {
  const net = unreachableNetwork(address);
  db.instance
    .prepare(`INSERT INTO slot_fill_tried (address, tried_at, ok, reason, started_at, network) VALUES (?, ?, ?, ?, ?, ?)
              ON CONFLICT(address) DO UPDATE SET tried_at = excluded.tried_at, ok = excluded.ok,
                reason = excluded.reason, started_at = excluded.started_at, network = excluded.network`)
    .run(address, now, ok ? 1 : 0, reason, startedAt, net === 'Tor' ? 'tor' : net === 'I2P' ? 'i2p' : 'clearnet');
}

function recentlyFailed(now) {
  db.instance.prepare('DELETE FROM slot_fill_tried WHERE tried_at < ?').run(now - TRIED_KEEP_MS);
  return new Set(db.instance
    .prepare(`SELECT address FROM slot_fill_tried WHERE ok = 0
              AND tried_at >= CASE WHEN reason = 'no_connection' THEN ? ELSE ? END`)
    .all(now - UNREACHED_RETRY_MS, now - FAILED_RETRY_MS)
    // By key: a peer removed by hand is held against under the spelling it
    // was stored with (no port, another IPv6 form), not the address book's.
    .map((r) => keyOf(r.address)));
}

// Fresh addresses first (15.11.2026): Core's address book keeps when it last
// heard of or connected to each one, and an address not heard of for a week
// is often a node that is gone - the candidates that took minutes to fail.
// Three bands, a day and a week; within a band the order is random, so the
// same few addresses are not tried by every node.
const DAY_S = 24 * 60 * 60;
function freshFirst(list, now) {
  const nowS = Math.floor(now / 1000);
  const band = (c) => {
    const age = nowS - Number(c.time || 0);
    return age <= DAY_S ? 0 : age <= 7 * DAY_S ? 1 : 2;
  };
  return shuffle(list).map((c, i) => ({ c, b: band(c), i })).sort((x, y) => x.b - y.b || x.i - y.i).map((x) => x.c);
}

function shuffle(list) {
  const a = list.slice();
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [a[i], a[j]] = [a[j], a[i]];
  }
  return a;
}

// getpeerinfo without the session of a candidate just taken out: Core drops
// it a moment later, and until then it would count as a working connection.
function withoutSession(peers, address) {
  return peers.filter((p) => !(p.connection_type === 'manual' && keyOf(p.addr) === keyOf(address)));
}

// Core's session to a manual peer, if it has one.
function manualSession(peers, address) {
  return peers.find((p) => p.connection_type === 'manual' && keyOf(p.addr) === keyOf(address)) || null;
}

// When the verdict on the candidate is due: from the trial's start while Core
// has no connection to it, from the connection once there is one - a
// connection that came late, over Tor, say, still gets its minutes to answer.
function decideAt(trial, session) {
  if (!session) return trial.startedAt + (NO_CONNECT_MS[unreachableNetwork(trial.address)] || NO_CONNECT_MS.clearnet);
  const headersOpen = servesBlocks(session.services) && !(Number(session.synced_headers) >= 0);
  if (headersOpen) return trial.startedAt + HEADERS_OPEN_MS;
  const connectWindow = NO_CONNECT_MS[unreachableNetwork(trial.address)] || NO_CONNECT_MS.clearnet;
  const connectedAt = Number(session.conntime) > 0 ? Number(session.conntime) * 1000 : trial.startedAt;
  // Never before the trial (a session from before it), never after the time a
  // connection had (a clock that runs ahead would push the verdict out).
  return Math.min(Math.max(trial.startedAt, connectedAt), trial.startedAt + connectWindow) + TRIAL_MS;
}

// What the slot needs from the peer: blocks and segwit, and our chain. Core
// asks every new peer for headers from just below its own tip, so a peer on
// this chain answers with the tip within seconds - on a real node, fresh
// connections showed the tip in synced_headers after 12 s. Later blocks do not
// move it unless that peer announces them first, so it is held against the
// tip as it was when the trial began, not as it is now: three blocks in a
// quarter of an hour would otherwise fail a good peer.
function works(session, chain, since = null) {
  if (!session || !servesBlocks(session.services)) return false;
  const tip = Number(since ?? chain.headers ?? chain.blocks);
  return Number.isFinite(tip) && Number(session.synced_headers) >= tip - 2;
}

// How the last candidate that did not keep its slot ended, for the top row of
// the page's log: that row says what Slot Fill is doing now, and with this
// also why the peer before went again (14.11.2026: a candidate showed in the
// manual list and left it again, and the log said nothing). Not a log entry:
// on a bad day ten of them would push the real changes out of the log.
const LAST_TRY_KEY = 'slot_fill_last_try';
function writeLastTry(value) {
  if (!value) {
    db.instance.prepare('DELETE FROM meta WHERE key = ?').run(LAST_TRY_KEY);
    return;
  }
  db.instance
    .prepare('INSERT INTO meta (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value')
    .run(LAST_TRY_KEY, JSON.stringify(value));
}
function lastTry() {
  try {
    const row = db.instance.prepare('SELECT value FROM meta WHERE key = ?').get(LAST_TRY_KEY);
    return row ? JSON.parse(row.value) : null;
  } catch {
    return null;
  }
}

// Why a trial ended without the slot, as a short code (kept in the database)
// and as words (the app log). "No connection" is what was seen at the checks:
// a session that came and went between two of them leaves no trace.
// A candidate that names Core 0.21 or later and finished its handshake
// without wtxidrelay, which every such Core sends: the peer list calls it
// Disguised (skipsWtxidrelay in peers/disguise.go, the same rule). A manual
// slot is for a real node, so it is out at once (13.11.2026: a
// "/Satoshi:25.1.0/" without wtxidrelay and sendaddrv2 sat in a trial
// waiting for headers). anySent, as there: on a node too old to get
// wtxidrelay from anyone, nobody is struck.
const CLAIMED_CORE = /^\/Satoshi:([0-9]+)\.([0-9]+)/;
function disguised(session, peers) {
  if (!session) return false;
  const m = CLAIMED_CORE.exec(session.subver || '');
  if (!m) return false;
  const maj = Number(m[1]);
  const mnr = Number(m[2]);
  const release = maj === 0 || maj >= 22;
  if (!release || (maj === 0 && mnr < 21)) return false;
  const recv = session.bytesrecv_per_msg || {};
  const anySent = peers.some((p) => p.bytesrecv_per_msg && p.bytesrecv_per_msg.wtxidrelay > 0);
  return anySent && recv.verack > 0 && !(recv.wtxidrelay > 0) && Number(session.version) >= 70016;
}

// A candidate that says it follows another chain - the service bits the
// peer list marks as a fork (peer-rules.js otherChainOf), and not on our tip
// either: out at once, as a disguised one is (15.11.2026: a BLAKE2b-fork Knots
// node held a slot three minutes for nothing).
// Only with a height Core knows for it: before its first headers (-1, the
// first seconds) a real node on our tip may carry the bits too.
function otherChain(session, chain) {
  if (!session || !(Number(session.synced_headers) >= 0)) return null;
  const names = Array.isArray(session.servicesnames) ? session.servicesnames.join(',') : '';
  return otherChainOf(names, Number(session.synced_headers), Number(chain.headers ?? chain.blocks), session.subver);
}

// The peer list's word on a peer (peers-service.js peerKinds: kind, fork),
// by address key: it judges a Core name by more than the one rule above -
// services the named version never had, a protocol number it never sent, and
// after ten minutes connected how the peer behaves (peers/disguise.go). On
// 15.11.2026 a peer the list called Disguised had passed its trial and sat in
// a manual slot. Empty when the peers service is down: then only Core's own
// signs count, as before.
function verdictsByKey(kinds) {
  const out = new Map();
  if (kinds) for (const [addr, m] of kinds) out.set(keyOf(addr), m);
  return out;
}
const listSaysDisguised = (m) => Boolean(m) && m.kind === 'Disguised';
// The log line for a Disguised peer: the peer list's reason when it gave one
// ("claims Bitcoin Core 27.1, but it lacks NETWORK_LIMITED, ..."), so the log
// shows what gave it away (16.11.2026), else the general line.
const disguisedNote = (m) => (m && m.why ? `Disguised: ${m.why}` : 'Disguised: not the Core it names');
const WTXID_NOTE = "Disguised: it names Core but left out wtxidrelay when connecting, which every Core since 0.21 sends";

function whyNot(session, fake = false, fork = null) {
  if (fake) return { code: 'disguised', text: 'connected, but it is not the Core it names' };
  // The chain's name rides along in the stored reason: other_chain:BLAKE2b fork.
  if (fork) return { code: `other_chain:${fork}`, text: `connected, but it follows another chain (${fork})` };
  if (!session) return { code: 'no_connection', text: 'no connection seen' };
  if (!servesBlocks(session.services)) return { code: 'no_services', text: 'connected, but serves no blocks' };
  if (!(Number(session.synced_headers) >= 0)) return { code: 'height_unknown', text: 'connected, but Core had no header height for it' };
  return { code: 'height_wrong', text: 'connected, but not on our chain tip' };
}

/**
 * Settle the candidates on trial: keep each Core holds a working connection
 * to, wait while its trial runs, take it out once that is over. `force`
 * settles now either way - the slot fill is being switched off. Returns how
 * many ended which way, and the addresses taken out (`gone`).
 */
async function settle(peers, chain, now = Date.now(), { force = false, kinds = null } = {}) {
  const out = { confirmed: 0, reverted: 0, left: 0, waiting: 0, gone: [] };
  const verdicts = verdictsByKey(kinds);
  for (const trial of trials()) {
    const r = await settleOne(trial, peers, chain, now, force, (session) => (session ? verdicts.get(keyOf(trial.address)) : null));
    out[r] += 1;
    if (r === 'reverted') out.gone.push(trial.address);
  }
  return out;
}

async function settleOne(trial, peers, chain, now, force, verdictOf = () => null) {
  const row = db.instance.prepare('SELECT kept FROM trusted_peer WHERE address = ?').get(trial.address);
  if (!row && now - trial.startedAt < ADDING_MS) return 'waiting';
  // Given the padlock by hand while on trial: a person's decision, and it
  // stands.
  if (row && row.kept) {
    dropTrial(trial.address);
    // Why the one before went, until no candidate is left on trial.
    if (!trials().length) writeLastTry(null);
    return 'left';
  }
  // Taken out by hand - or the app stopped, or the add failed, after Core was
  // told: Core's list may still hold it, and the next sync would adopt it as
  // somebody else's addnode, padlocked. Told again to drop it - through
  // removeTrustedPeer, which remembers a remove Core did not take, so the sync
  // repeats it rather than adopting the candidate. Before the trial closes:
  // the trial is what keeps the sync off it until then.
  if (!row) {
    await peerSync.removeTrustedPeer(trial.address, { disconnect: false });
    dropTrial(trial.address);
    // Ended by a person: the one before is no longer the peer before.
    writeLastTry(null);
    return 'left';
  }
  const session = manualSession(peers, trial.address);
  // Only while Core holds the candidate: the list's word is about a session.
  // Its Disguised only: its fork mark comes without the wait for a first
  // header height that otherChain() keeps (review, 15.11.2026).
  const listed = verdictOf(session);
  const byRule = disguised(session, peers);
  const fake = byRule || listSaysDisguised(listed);
  const fork = fake ? null : otherChain(session, chain);
  if (!fake && !fork && works(session, chain, trial.tip)) {
    dropTrial(trial.address);
    // Why the one before went, until no candidate is left on trial.
    if (!trials().length) writeLastTry(null);
    markTried(trial.address, true, now, { reason: 'ok', startedAt: trial.startedAt });
    logAction({ action: 'promote', address: trial.address, note: trial.note });
    logger.info('slot fill: Core connected, the candidate keeps the slot', { address: trial.address });
    return 'confirmed';
  }
  // How long the verdict waits depends on what Core has to show for it so
  // far: no connection at all, a connection with no header height yet, or a
  // height that is not ours.
  if (!force && !fake && !fork && now < decideAt(trial, session)) return 'waiting';

  // Out of the set first, then the trial is closed: a crash in between leaves
  // a trial whose candidate is gone, which the next pass reads as 'left'.
  // Not in the dashboard's log: a candidate Core could not use is the slot
  // fill's own business, and the next one is tried anyway.
  await peerSync.removeTrustedPeer(trial.address);
  dropTrial(trial.address);
  const why = whyNot(session, fake, fork);
  const minutes = Math.max(1, Math.round((now - trial.startedAt) / 60000));
  if (!force) {
    markTried(trial.address, false, now, { reason: why.code, startedAt: trial.startedAt });
    // In the log too when the candidate was not what it said - another
    // chain, or not the Core it names: rare, and worth seeing (15.11.2026).
    // One that just did not answer stays in the list of tries only.
    if (fork || fake) {
      const note = fork ? `Follows another chain (${fork})` : listed && listed.why ? disguisedNote(listed) : byRule ? WTXID_NOTE : disguisedNote(null);
      logAction({ action: 'rejected', address: trial.address, note });
    }
    const { region, country } = spread.placeOf({ address: trial.address });
    writeLastTry({ at: now, reason: why.code, minutes, country: country || null, region: spread.REGION_NAMES[region] || null });
  } else {
    writeLastTry(null);
  }
  logger.info('slot fill: candidate not kept', {
    address: trial.address, what: trial.what, reason: force ? 'switched off' : `${why.text} in ${minutes} min`,
  });
  return 'reverted';
}

/**
 * A peer Slot Fill added that the peer list calls Disguised after its trial:
 * some of the list's rules need ten minutes of the peer's behaviour, longer
 * than a trial runs. Out again, held against for a week, and a row in the
 * log, as at a trial. Only Slot Fill's own adds that nobody locked, and only
 * while Core holds a manual session to it - the verdict is about that
 * session. A peer a person added or locked is theirs. Returns the addresses
 * taken out.
 */
const ADDED_LABEL = 'slot fill:%';
async function dropUnmasked(kinds, peers, stillOn = () => true, now = Date.now()) {
  const verdicts = verdictsByKey(kinds);
  if (!verdicts.size) return [];
  const onTrial = new Set(trials().map((t) => keyOf(t.address)));
  const rows = db.instance.prepare('SELECT address FROM trusted_peer WHERE kept = 0 AND label LIKE ?').all(ADDED_LABEL);
  const still = db.instance.prepare('SELECT 1 FROM trusted_peer WHERE address = ? AND kept = 0 AND label LIKE ?');
  const out = [];
  for (const { address } of rows) {
    if (onTrial.has(keyOf(address)) || !manualSession(peers, address)) continue;
    const listed = verdicts.get(keyOf(address));
    if (!listSaysDisguised(listed)) continue;
    // Each removal waits for Core: switched off, locked or removed by hand
    // meanwhile, the rest is left alone (as retire.js does).
    if (!stillOn() || !still.get(address, ADDED_LABEL)) break;
    await peerSync.removeTrustedPeer(address);
    markTried(address, false, now, { reason: 'disguised' });
    logAction({ action: 'rejected', address, note: disguisedNote(listed) });
    logger.info('slot fill: added peer taken out again, the peer list calls it Disguised', { address });
    out.push(address);
  }
  return out;
}

// Added by Slot Fill this recently: the peer list's verdict is asked every
// minute, not every ten (index.js watchTrial), as its behaviour rules need
// ten minutes of the peer.
const WATCH_ADDED_MS = 30 * 60 * 1000;
function recentlyAdded(now = Date.now()) {
  const row = db.instance.prepare("SELECT MAX(tried_at) AS at FROM slot_fill_tried WHERE ok = 1 AND reason = 'ok'").get();
  return Boolean(row && row.at && now - row.at < WATCH_ADDED_MS);
}

// What the last look for a candidate found, for the line on the page that
// says what Slot Fill is doing (index.js, status): why nothing went in, and
// how many addresses were left after each step. Read right after start().
let lastLook = null;
const looked = (why, counts = {}, result = []) => { lastLook = { why, ...counts }; return result; };

// How many addresses Slot Fill could pick from at its last look, and how many
// per region, for the line on the page ("4,812 addresses to pick from"):
// those from "tried", Core has connected to them before - what the page says.
// A region a usable network could reach but nobody in the book sits in is
// there with 0, so the page can name it; Tor and I2P wherever Core may use them.
// Cleared at every look, so a look that could not read the book shows none
// rather than an old count beside a status that says the book is empty.
let lastBook = null;
// The picked regions as the last look found them, for the page: still
// candidates there, or back to the usual pick.
let lastPicks = null;
function noteBook(candidates, nets, now) {
  const book = candidates.filter((c) => !c.fromNew);
  const regions = {};
  if (nets.has('ipv4') || nets.has('ipv6')) for (const r of ['AF', 'AS', 'EU', 'NA', 'OC', 'SA']) regions[r] = 0;
  if (nets.has('onion')) regions.TOR = 0;
  if (nets.has('i2p')) regions.I2P = 0;
  for (const c of book) {
    const { region } = spread.placeOf(c);
    if (region in regions) regions[region] += 1;
  }
  lastBook = { at: now, total: book.length, regions };
}

/**
 * Put candidates on trial in the free slots: as many as are free, up to
 * MAX_TRIALS on trial at once. `peers` is getpeerinfo from this pass. Returns
 * the addresses put on trial, an empty list for none.
 */
async function start(ranking, peers, chain, now = Date.now(), { maxTrials = MAX_TRIALS } = {}) {
  const started = [];
  // As many on trial as may be: nothing to ask Core, and the page keeps what
  // the last look found (15 s checks while trials run, review 15.11.2026).
  if (trials().length >= maxTrials) return started;
  lastLook = null;
  lastBook = null;
  lastPicks = null;
  // A node still catching up has no tip to test a candidate against.
  if (chain.initialblockdownload) return looked('catching_up', {}, started);
  // Copied: each candidate put on trial below counts as one of them for the
  // next pick, so the next goes to another region and provider.
  const trusted = ranking.filter((p) => p.trusted);
  const max = config.maxManualPeers;
  // With every slot in use the book is still counted, for the page: how
  // much there would be to pick from (noteBook). Nothing is tried.
  const full = trusted.length >= max;
  const room = () => Math.min(maxTrials - trials().length, max - trusted.length);

  let networkInfo;
  try {
    networkInfo = await rpc.getNetworkInfo();
  } catch (err) {
    logger.debug('slot fill: getnetworkinfo failed', { error: err.message });
    return looked('core', {}, started);
  }
  // `nets`: where the usual pick looks. `offered`: what the page lists and
  // the reader may pick, Tor and I2P included whenever Core may use them.
  const nets = usableNetworks(networkInfo, peers);
  const offered = offeredNetworks(networkInfo, peers);
  // No outbound connection at all and nothing picked: the usual pick has no
  // network, and the page says so rather than "nothing found" (review).
  if (!offered.size || (!nets.size && !pickedRegions().length)) return looked('no_network', {}, started);
  const banned = await bannedMatcher();
  if (!banned) return looked('core', {}, started);

  // Core's own address too: an address book can hold the node itself.
  const inUse = new Set([
    ...peers.map((p) => hostFromAddress(p.addr)),
    ...trusted.map((p) => hostFromAddress(p.address)),
    ...(networkInfo.localaddresses || []).map((a) => hostFromAddress(String(a.address))),
  ]);
  const failed = recentlyFailed(now);
  const all = await bookEntries(now);
  const reachable = all.filter((c) => offered.has(c.network));
  const book = freshFirst(reachable
    .filter((c) => !inUse.has(hostFromAddress(c.address)) && !banned(c.address) && !failed.has(keyOf(c.address))), now);
  const counts0 = { known: all.length, reachable: reachable.length, untried: book.length };
  if (all.length) noteBook(book, offered, now);
  if (full) return looked('full', {}, started);
  if (!all.length) return looked('no_book', counts0, started);
  if (room() <= 0) return started;

  // Counted over all outbound connections, Core's own too (spread.js).
  const coreRegions = spread.coreRegionsOf(ranking);
  const coreKeys = spread.coreOperatorsOf(ranking);
  const coreCountries = spread.coreCountriesOf(ranking);
  const picked = new Set(pickedRegions());
  const inPicked = (c) => picked.has(spread.placeOf(c).region);
  // Where clearnet works, one Tor and one I2P connection are enough, Core's
  // own counted (10.11.2026): an I2P or Tor peer may sit anywhere, and those
  // networks are small, so a further one says less than a continent. A node
  // on Tor alone takes all it can get there.
  const clearnet = nets.has('ipv4') || nets.has('ipv6');

  // The order for the next pick, over the manual set as it stands with the
  // candidates already put on trial in this call.
  const heldCountries = () => new Set(trusted.map((p) => spread.placeOf(p).country).filter(Boolean));
  const order = () => {
    const outbound = spread.totalCounts(trusted, coreRegions);
    const gaps = new Set(Object.keys(spread.REGION_NAMES).filter((r) => (outbound[r] || 0) < spread.TARGET_PER_REGION));
    const inGap = (c) => gaps.has(spread.placeOf(c).region);
    const rankedAll = spread.rankForGap(book.filter((c) => !inUse.has(hostFromAddress(c.address))), trusted, coreRegions, coreCountries);
    // A picked Tor or I2P is the reader's word against that rule.
    const capped = (c) => {
      const { region } = spread.placeOf(c);
      return clearnet && (region === 'TOR' || region === 'I2P') && (outbound[region] || 0) >= 1 && !picked.has(region);
    };
    for (let i = rankedAll.length - 1; i >= 0; i--) if (capped(rankedAll[i])) rankedAll.splice(i, 1);
    // The usual pick only where Core is connected out now; a picked region
    // may be anywhere the page offers.
    const ranked = rankedAll.filter((c) => nets.has(c.network));
    const tried = ranked.filter((c) => !c.fromNew);
    const triedAll = rankedAll.filter((c) => !c.fromNew);
    // A missing region "tried" has nobody from may look in "new" (addrbook.js).
    // Nobody the provider rule would let in, that is: a region whose only
    // tried candidates are with providers already in use is still missing.
    const triedGaps = new Set(tried.filter((c) => inGap(c) && spread.admit(c, trusted, max, coreKeys)).map((c) => spread.placeOf(c).region));
    const stand = ranked.filter((c) => c.fromNew && inGap(c) && !triedGaps.has(spread.placeOf(c).region));
    // The missing regions first, from "tried", then from "new" where "tried"
    // has none; then a country no manual peer is in yet, "tried" first and
    // "new" after (review 16.11.2026: a second Australian from "tried" came
    // before New Zealand from "new"); a country again only when nothing else
    // is left, so that no slot stays empty for want of a perfect candidate.
    const fresh = (c) => !heldCountries().has(spread.placeOf(c).country || '-');
    const usual = [...tried.filter(inGap), ...stand,
      ...tried.filter((c) => !inGap(c) && fresh(c)), ...ranked.filter((c) => c.fromNew && !inGap(c) && fresh(c)),
      ...tried.filter((c) => !inGap(c) && !fresh(c))];
    // Picked regions (picks.js): theirs only, "tried" first, then "new" - and
    // the usual pick once none of them has a candidate the spread lets in.
    // In ranked order, so the spread among them holds; "new" only for a picked
    // region "tried" has nobody admissible in, as for the gaps above (review:
    // with Europe and Africa picked, Europe's thousands came first every time).
    //
    // In the picked regions the provider rule is half as strict, but only once
    // no candidate in any of them passes it whole (15.11.2026): an operator
    // Core's own outbound peers use may come in, one the manual peers already
    // have never. Decided over the picked regions together, not each alone.
    const pickKeys = (keys) => {
      const triedPicked = new Set(triedAll.filter((c) => inPicked(c) && spread.admit(c, trusted, max, keys)).map((c) => spread.placeOf(c).region));
      const list = picked.size ? rankedAll.filter((c) => inPicked(c) && (!c.fromNew || !triedPicked.has(spread.placeOf(c).region))) : [];
      return { list, left: list.some((c) => spread.admit(c, trusted, max, keys)) };
    };
    let mine = pickKeys(coreKeys);
    let relaxed = false;
    if (picked.size && !mine.left) {
      mine = pickKeys(new Set());
      relaxed = mine.left;
    }
    const pickedLeft = mine.left;
    const keys = pickedLeft && relaxed ? new Set() : coreKeys;
    return { ordered: pickedLeft ? mine.list : usual, pickedLeft, keys, outbound };
  };

  let attempts = 0;
  let fitting = 0;
  while (room() > 0) {
    const { ordered, pickedLeft, keys, outbound } = order();
    // The picks as the first look of this call found them, for the page.
    if (!started.length) lastPicks = picked.size ? { regions: [...picked], empty: !pickedLeft } : null;
    const fits = ordered.filter((c) => spread.admit(c, trusted, max, keys));
    if (!started.length) fitting = fits.length;
    if (!fits.length) return started.length ? started : looked('none_fit', { ...counts0, fitting }, started);
    const c = fits[0];
    // Tried once in this call, whatever came of it.
    inUse.add(hostFromAddress(c.address));
    const what = spread.describe(c);
    const { region } = spread.placeOf(c);
    // Where it is shows under its address in the log; the note says why: a
    // region with none, a country new to the manual peers, or as few as any.
    const { country } = spread.placeOf(c);
    const newCountry = country && !heldCountries().has(country);
    const why = pickedLeft ? `${spread.REGION_NAMES[region] || 'a region'} picked by you`
      : region && !outbound[region] ? `${spread.REGION_NAMES[region]} had none`
        : newCountry ? `${countryName(country)}, a new country`
          : region ? `${spread.REGION_NAMES[region]} had only ${outbound[region]}` : 'free slot';
    // Written before the add: if the app stops between the two, the next pass
    // finds the trial and either its candidate (and judges it) or nothing.
    addTrial({
      address: c.address,
      startedAt: now,
      tip: Number(chain.headers ?? chain.blocks),
      what,
      note: why,
      ...(c.fromNew ? { fromNew: true } : {}),
    });
    const result = await peerSync.addTrustedPeer(c.address, `slot fill: ${what}`, { neverDisconnect: true });
    if (result.ok) {
      logger.info('slot fill: candidate on trial', { address: c.address });
      started.push(c.address);
      trusted.push({ address: c.address, trusted: true });
      continue;
    }
    dropTrial(c.address);
    logger.info('slot fill: candidate not added', { address: c.address, error: result.error });
    // The set filled up under us (a peer added by hand meanwhile): nothing
    // wrong with the candidate, and nothing more to do this pass. Nor is a
    // refusal held against it: it may be Core not answering, and a week's
    // cooldown for that would cost good addresses.
    if (result.count >= max) return started.length ? started : looked('full', {}, started);
    attempts += 1;
    if (attempts >= ATTEMPTS_PER_PASS) return started.length ? started : looked('refused', { ...counts0, fitting }, started);
  }
  return started;
}

// How the candidate on trial stands, for the page: what Core has to show for
// it so far and how long the trial may still run - the same limits settle()
// judges by.
function trialView(trial, peers, now = Date.now()) {
  const session = manualSession(peers, trial.address);
  const headersOpen = Boolean(session) && servesBlocks(session.services) && !(Number(session.synced_headers) >= 0);
  const phase = !session ? 'connecting' : headersOpen ? 'headers' : 'checking';
  const { region, country } = spread.placeOf({ address: trial.address });
  return {
    address: trial.address, startedAt: trial.startedAt, phase, decideBy: decideAt(trial, session),
    // Where, in words a person reads: the country, or Tor, I2P, the region.
    country: country || null, region: spread.REGION_NAMES[region] || null,
  };
}

// After a gap (clock.js) the trial starts over: Core may have been down, with
// no connection to anyone, or the clock stepped, and either way the time so
// far says nothing about the candidate. The tip is taken anew too. Three
// times at most: a Core that keeps going away would otherwise keep one
// candidate on trial, holding the slot, for good.
const MAX_RESTARTS = 3;
function restart(chain, now = Date.now()) {
  let any = false;
  for (const trial of trials()) {
    const restarts = (trial.restarts || 0) + 1;
    if (restarts > MAX_RESTARTS) {
      logger.info('slot fill: trial not started over again, judged as it stands', { address: trial.address });
      continue;
    }
    replaceTrial({ ...trial, startedAt: now, tip: Number(chain.headers ?? chain.blocks), restarts });
    logger.info('slot fill: trial starts over after a gap', { address: trial.address, restarts });
    any = true;
  }
  return any;
}

// Take a candidate out at once, trial or not: the slot fill was switched off
// while this pass was adding it.
async function abandon(address, now = Date.now()) {
  await peerSync.removeTrustedPeer(address);
  dropTrial(address);
  logger.info('slot fill: candidate taken out, switched off while adding', { address, at: now });
}

module.exports = { MAX_RESTARTS, WATCH_ADDED_MS, dropUnmasked, recentlyAdded, MAX_TRIALS, NO_CONNECT_MS, settle, trials, start, restart, abandon, markTried, currentTrial, works, withoutSession, trialView, lastTry, lastLook: () => lastLook, lastBook: () => lastBook, lastPicks: () => lastPicks, recentlyFailedForTest: recentlyFailed };
