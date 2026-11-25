'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('fs');
const os = require('os');
const path = require('path');

const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'peerlab-live-'));
process.env.SQLITE_PATH = path.join(tmpDir, 'test.db');
process.env.DATA_DIR = tmpDir;
process.env.LOG_LEVEL = 'error';

const db = require('../src/lib/db');
const race = require('../src/stratum-race');
const toggle = require('../src/lib/stratum-race-toggle');
const live = require('../src/lib/stratum-live');
const { encodings } = require('../src/lib/prevhash');

const hr = () => process.hrtime.bigint();
const hash = (n) => String(n).padStart(64, '0');

function watch(...pools) {
  race.active.clear();
  for (const p of pools) race.active.set(p.id, { conn: { stop() {}, connectedSince: Date.now() - 60_000 }, pool: p });
}

test.before(() => {
  db.open();
  toggle.setEnabled(true);
  race.setVerifier(async () => true);
});

test.after(async () => {
  await race.finalizeAllRaces();
});

test('each pool shows up in the live file as its job arrives, before the race is written', async () => {
  const a = { id: 1, label: 'A', host: 'a.example', port: 3333, is_default: 1 };
  const b = { id: 2, label: 'Mine', host: 'b.example', port: 3333, is_default: 0 };
  const c = { id: 3, label: 'C', host: 'c.example', port: 3333, is_default: 1 };
  watch(a, b, c);
  const at = Date.now();
  const t0 = hr();

  race.handleNotify(a, hash(11), t0, at);
  await live.flushed();
  let snap = live.read();
  assert.equal(snap.done, false);
  assert.deepEqual(snap.entries.map((e) => [e.label, e.rank]), [['A', 1]]);
  assert.equal(db.instance.prepare('SELECT COUNT(*) AS n FROM stratum_race').get().n, 0, 'nothing in the database while the race is open');

  race.handleNotify(b, hash(11), t0 + 3_000_000n, at);
  await live.flushed();
  snap = live.read();
  assert.deepEqual(snap.entries.map((e) => [e.label, e.own, e.rank, Math.round(e.latencyMs)]), [['A', false, 1, 0], ['Mine', true, 2, 3]]);

  // Closing adds the pool that sent nothing, as a miss.
  await race.finalizeRace(hash(11));
  await live.flushed();
  snap = live.read();
  assert.equal(snap.done, true);
  assert.deepEqual(snap.entries.map((e) => [e.label, e.latencyMs]), [['A', 0], ['Mine', snap.entries[1].latencyMs], ['C', null]]);
});

test('the page gets every spelling of the hash, the card\'s shape, and the Core stop when known', () => {
  const now = Date.now();
  const snap = { prevhash: hash(12), createdAt: now - 1000, done: false, entries: [{ label: 'A', own: false, latencyMs: 0, rank: 1 }, { label: 'B', latencyMs: null, rank: null }] };
  const page = live.forPage(snap, now - 1200, now);
  assert.deepEqual(page.hashes, encodings(hash(12)));
  assert.deepEqual(page.stratum.entries, [
    { label: 'A', own: false, latency_ms: 0, rank: 1, miss: false },
    { label: 'B', own: false, latency_ms: null, rank: null, miss: true },
  ]);
  assert.equal(page.stratum.core_ms, -200, 'Core had the block 200 ms before the first job');
  assert.equal(live.forPage(snap, null, now).stratum.core_ms, undefined, 'no Core stop before Core has the block');
});

test('an old or broken snapshot shows nothing', () => {
  const now = Date.now();
  assert.equal(live.forPage({ prevhash: hash(13), createdAt: now - live.MAX_AGE_MS - 1, entries: [] }, null, now), null);
  assert.equal(live.forPage({ prevhash: 'not a hash', createdAt: now, entries: [] }, null, now), null);
  fs.writeFileSync(live.FILE, '{ not json');
  assert.equal(live.read(), null);
});
