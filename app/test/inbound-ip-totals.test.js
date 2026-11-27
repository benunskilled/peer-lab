'use strict';

// An inbound peer comes back from a new port almost every time; the peer list
// counts its blocks over every port its IP came in from (peer-ranking.js,
// addIpTotals). The ranking itself keeps counting per address.

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('fs');
const os = require('os');
const path = require('path');

const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'peerlab-ipsum-'));
process.env.SQLITE_PATH = path.join(tmpDir, 'test.db');
process.env.DATA_DIR = tmpDir;
process.env.LOG_LEVEL = 'error';

const db = require('../src/lib/db');
const queries = require('../src/lib/queries');

test.before(() => {
  db.open();
});

function peer(address, { direction, network = 'ipv4', live = false, eligible = 0, first = 0 }) {
  const now = Date.now();
  const id = db.instance.prepare('INSERT INTO peer (address, first_seen_at) VALUES (?, ?)').run(address, now).lastInsertRowid;
  db.instance.prepare(`INSERT INTO peer_session (peer_id, direction, connection_type, network, subver, started_at, ended_at)
    VALUES (?, ?, ?, ?, '/Satoshi:31.0.0/', ?, ?)`)
    .run(id, direction, direction === 'inbound' ? 'inbound' : 'outbound-full-relay', network, now - 60000, live ? null : now - 1000);
  if (eligible) db.instance.prepare('INSERT INTO peer_relay_stats (peer_id, eligible, first) VALUES (?, ?, ?)').run(id, eligible, first);
}

test('a live inbound peer carries the blocks of every port its IP came in from', () => {
  peer('203.0.113.10:40001', { direction: 'inbound', eligible: 300, first: 2 });
  peer('203.0.113.10:40002', { direction: 'inbound', eligible: 20, first: 1 });
  peer('203.0.113.10:40003', { direction: 'inbound', live: true, eligible: 5 });
  // The same IP as a node Core dialled is another matter, and so is a longer
  // address that only starts the same.
  peer('203.0.113.10:8333', { direction: 'outbound', eligible: 900, first: 50 });
  peer('203.0.113.100:40001', { direction: 'inbound', eligible: 70, first: 7 });
  // Every Tor peer comes in from the proxy's address: never added up.
  peer('127.0.0.1:50001', { direction: 'inbound', network: 'onion', eligible: 400, first: 4 });
  peer('127.0.0.1:50002', { direction: 'inbound', network: 'onion', live: true, eligible: 3 });

  const rows = Object.fromEntries(queries.peerRanking().map((r) => [r.address, r]));
  const live = rows['203.0.113.10:40003'];
  assert.equal(live.ipEligible, 325);
  assert.equal(live.ipFirst, 3);
  assert.equal(live.eligible, 5, 'the ranking keeps its own count');
  assert.equal(live.first, 0);
  assert.equal(rows['127.0.0.1:50002'].ipEligible, undefined);
});
