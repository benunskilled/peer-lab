'use strict';

// What a peer did per session, kept for counting later (20.11.2026): the
// message byte counters, last_transaction, last_block and the address
// counters. Written with a new session and then at most every five minutes,
// and added to a database from before 1.0.2 without touching its rows.

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('fs');
const os = require('os');
const path = require('path');
const Database = require('better-sqlite3');

const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'peerlab-counters-'));
const dbPath = path.join(tmpDir, 'test.db');
process.env.SQLITE_PATH = dbPath;
process.env.DATA_DIR = tmpDir;
process.env.LOG_LEVEL = 'error';

// A database as 1.0.1 left it: peer_session without the new columns, one row
// in it (the peer table comes from the schema as before).
{
  const old = new Database(dbPath);
  old.pragma('foreign_keys = OFF');
  old.exec(`CREATE TABLE peer_session (id INTEGER PRIMARY KEY AUTOINCREMENT, peer_id INTEGER NOT NULL REFERENCES peer(id),
      core_peer_id INTEGER, direction TEXT NOT NULL, connection_type TEXT NOT NULL, network TEXT, subver TEXT,
      services TEXT, relay_txes INTEGER, synced_headers INTEGER, hb_to INTEGER, hb_from INTEGER,
      started_at INTEGER NOT NULL, ended_at INTEGER, min_ping_ms REAL, last_ping_ms REAL);
    INSERT INTO peer_session (peer_id, direction, connection_type, started_at, ended_at) VALUES (1, 'inbound', 'inbound', 1, 2);`);
  old.close();
}

const db = require('../src/lib/db');
const { upsertSessions } = require('../src/peer-profiler');

test.before(() => {
  db.open();
});

const A = '203.0.113.81:8333';

function corePeer(extra = {}) {
  return {
    addr: A,
    id: 7,
    inbound: true,
    connection_type: 'inbound',
    subver: '/Satoshi:32.1.0/',
    conntime: Math.floor(Date.now() / 1000) - 3600,
    bytesrecv_per_msg: { version: 126, verack: 24, ping: 64 },
    bytessent_per_msg: { version: 127, inv: 61000 },
    last_transaction: 0,
    last_block: 0,
    addr_processed: 0,
    addr_rate_limited: 0,
    ...extra,
  };
}

const row = () => db.instance.prepare(
  `SELECT ps.* FROM peer_session ps JOIN peer p ON p.id = ps.peer_id WHERE p.address = ? AND ps.ended_at IS NULL`,
).get(A);

test('a database from before gets the columns, and its rows stay', () => {
  const cols = db.instance.prepare('PRAGMA table_info(peer_session)').all().map((c) => c.name);
  for (const c of ['msg_recv', 'msg_sent', 'last_tx', 'last_block', 'addr_processed', 'addr_rate_limited', 'counters_at']) {
    assert.ok(cols.includes(c), `${c} missing`);
  }
  const old = db.instance.prepare(`SELECT * FROM peer_session WHERE peer_id = 1`).get();
  assert.equal(old.direction, 'inbound');
  assert.equal(old.msg_recv, null);
});

test('a new session is written with its counters, then only every five minutes', () => {
  const t0 = Date.now();
  upsertSessions([corePeer()], t0);
  let r = row();
  assert.deepEqual(JSON.parse(r.msg_recv), { version: 126, verack: 24, ping: 64 });
  assert.equal(JSON.parse(r.msg_sent).inv, 61000);
  assert.equal(r.last_tx, 0);
  assert.equal(r.counters_at, t0);

  // A minute later: not rewritten for the counters alone.
  upsertSessions([corePeer({ bytesrecv_per_msg: { version: 126, inv: 900 }, last_transaction: 1763650000 })], t0 + 60_000);
  r = row();
  assert.equal(JSON.parse(r.msg_recv).inv, undefined);
  assert.equal(r.counters_at, t0);

  // Five minutes on: the latest reading.
  upsertSessions([corePeer({ bytesrecv_per_msg: { version: 126, inv: 900 }, last_transaction: 1763650000 })], t0 + 300_000);
  r = row();
  assert.equal(JSON.parse(r.msg_recv).inv, 900);
  assert.equal(r.last_tx, 1763650000);
  assert.equal(r.counters_at, t0 + 300_000);
});

test('a field Core leaves out stays empty, not zero', () => {
  db.instance.exec(`DELETE FROM peer_session WHERE peer_id != 1`);
  const p = corePeer();
  delete p.addr_processed;
  delete p.bytessent_per_msg;
  upsertSessions([{ ...p, id: 8 }], Date.now());
  const r = row();
  assert.equal(r.addr_processed, null);
  assert.equal(r.msg_sent, null);
});

test('a session that ends before its next write keeps its last reading', () => {
  db.instance.exec(`DELETE FROM peer_session WHERE peer_id != 1`);
  const t0 = Date.now();
  upsertSessions([corePeer({ id: 9 })], t0);
  upsertSessions([corePeer({ id: 9, bytesrecv_per_msg: { version: 126, getdata: 61 } })], t0 + 30_000);
  // Gone at the next poll, two minutes after it started.
  upsertSessions([], t0 + 120_000);
  const r = db.instance.prepare(
    `SELECT ps.* FROM peer_session ps JOIN peer p ON p.id = ps.peer_id WHERE p.address = ? ORDER BY ps.id DESC`,
  ).get(A);
  assert.ok(r.ended_at != null);
  assert.equal(JSON.parse(r.msg_recv).getdata, 61);
});
