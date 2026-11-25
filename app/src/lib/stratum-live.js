'use strict';

/**
 * The Stratum race as it happens, for the block card.
 *
 * The race worker writes a race to the database only once it is over (see
 * writeRace in stratum-race.js: a write that meets another process's lock
 * would stand the event loop still and smear every timestamp behind it). So
 * the block card used to wait the whole window - eight seconds - plus a
 * refresh before it showed a single pool.
 *
 * This is the side door. After every pool that reports, the worker leaves a
 * small JSON file beside the database: which pools have sent a job for this
 * block so far, and how far behind the first. Written asynchronously
 * (libuv's thread pool, never the event loop), after the timestamp was taken,
 * so it cannot move one. The dashboard watches the file and passes it on to
 * open pages over its event stream; the page lays it into the block card
 * until the recorded race arrives and replaces it.
 *
 * Nothing here is history. A lost or stale file costs a live view, never a
 * number: the database stays the only record.
 */

const fs = require('fs');
const path = require('path');
const config = require('./config');
const prevhashLib = require('./prevhash');

const FILE = path.join(path.dirname(config.sqlitePath), 'stratum-live.json');
// A snapshot older than this is not shown: the race closes after eight
// seconds and the card's moment passes in a minute or two.
const MAX_AGE_MS = 2 * 60 * 1000;
// Bounds what a page is sent, whatever ends up in the file.
const MAX_ENTRIES = 64;

// One write at a time, and only the newest waits: pools arriving within a
// millisecond of each other cost one write each at most, never a queue.
let writing = false;
let pending = null;

function publish(snapshot) {
  if (writing) {
    pending = snapshot;
    return;
  }
  writing = true;
  const tmp = `${FILE}.tmp`;
  fs.promises
    .writeFile(tmp, JSON.stringify(snapshot))
    .then(() => fs.promises.rename(tmp, FILE))
    .catch(() => { /* a live view lost, nothing else - see above */ })
    .finally(() => {
      writing = false;
      if (pending) {
        const next = pending;
        pending = null;
        publish(next);
      }
    });
}

// Resolves once the last write has landed. For tests only.
function flushed() {
  return new Promise((resolve) => {
    const check = () => (writing ? setTimeout(check, 5) : resolve());
    check();
  });
}

function read() {
  let raw;
  try {
    raw = fs.readFileSync(FILE, 'utf8');
  } catch {
    return null;
  }
  try {
    const s = JSON.parse(raw);
    if (!s || typeof s.prevhash !== 'string' || !Number.isFinite(s.createdAt) || !Array.isArray(s.entries)) return null;
    return s;
  } catch {
    return null;
  }
}

const num = (v) => (Number.isFinite(v) ? v : null);

/**
 * What a page is sent: the hash in every spelling (the block card knows the
 * block by Core's spelling, the race by the pool's), and the entries in the
 * shape the peer list's last_block.stratum has, so the card draws both the
 * same way. `detectedAt` is when Core announced the block, if it has: the
 * race's zero laid next to it gives the route's Core stop.
 */
function forPage(snapshot, detectedAt, now = Date.now()) {
  if (!snapshot || now - snapshot.createdAt > MAX_AGE_MS) return null;
  const hashes = prevhashLib.encodings(snapshot.prevhash);
  if (!hashes.length) return null;
  const entries = [];
  for (const e of snapshot.entries.slice(0, MAX_ENTRIES)) {
    if (!e || typeof e.label !== 'string') continue;
    const latency = num(e.latencyMs);
    entries.push({ label: e.label.slice(0, 80), own: Boolean(e.own), latency_ms: latency, rank: num(e.rank), miss: latency === null });
  }
  const stratum = { entries };
  if (Number.isFinite(detectedAt) && detectedAt > 0) stratum.core_ms = detectedAt - snapshot.createdAt;
  return { hashes, done: Boolean(snapshot.done), stratum };
}

module.exports = { publish, read, forPage, flushed, FILE, MAX_AGE_MS };
