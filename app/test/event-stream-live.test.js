'use strict';

// The dashboard's half of the live Stratum race: the file the race worker
// leaves (lib/stratum-live.js) goes out on /api/events as 'stratum', and once
// Core has the block it carries the Core stop too.

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('fs');
const os = require('os');
const path = require('path');
const http = require('http');

const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'peerlab-sse-live-'));
process.env.SQLITE_PATH = path.join(tmpDir, 'test.db');
process.env.DATA_DIR = tmpDir;
process.env.LOG_LEVEL = 'error';
process.env.PEERS_HEALTH_URL = 'off';

const db = require('../src/lib/db');
const live = require('../src/lib/stratum-live');
const { encodings } = require('../src/lib/prevhash');
const { server, watchLiveStratum } = require('../src/dashboard-server');

const H = '00000000000000000001a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f7';
let baseUrl;

test.before(async () => {
  db.open();
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  baseUrl = `http://127.0.0.1:${server.address().port}`;
  watchLiveStratum();
});

test.after(() => {
  fs.unwatchFile(live.FILE);
  server.close();
  fs.rmSync(tmpDir, { recursive: true, force: true });
});

// Collects 'stratum' events from one stream until `want` of them have come.
function stratumEvents(want, ms = 4000) {
  return new Promise((resolve, reject) => {
    const got = [];
    let buf = '';
    const req = http.get(`${baseUrl}/api/events`, (res) => {
      res.setEncoding('utf8');
      res.on('data', (chunk) => {
        buf += chunk;
        let i;
        while ((i = buf.indexOf('\n\n')) >= 0) {
          const frame = buf.slice(0, i);
          buf = buf.slice(i + 2);
          if (!frame.startsWith('event: stratum')) continue;
          got.push(JSON.parse(frame.split('\ndata: ')[1]));
          if (got.length >= want) { req.destroy(); resolve(got); }
        }
      });
    });
    req.on('error', () => {});
    setTimeout(() => { req.destroy(); reject(new Error(`only ${got.length} of ${want} stratum events`)); }, ms).unref();
  });
}

test('the live race reaches the page, pool by pool, with the Core stop once Core has the block', async () => {
  const createdAt = Date.now();
  const spelling = encodings(H)[3];
  const events = stratumEvents(2);
  await new Promise((r) => setTimeout(r, 300)); // the stream is open
  live.publish({ prevhash: spelling, createdAt, done: false, entries: [{ label: 'A', own: false, latencyMs: 0, rank: 1 }] });
  await live.flushed();
  await new Promise((r) => setTimeout(r, 600)); // the file watcher's turn
  db.instance.prepare('INSERT INTO relay_race (block_hash, block_height, detected_at, first_count) VALUES (?, ?, ?, 0)').run(H, 900000, createdAt + 30);
  live.publish({ prevhash: spelling, createdAt, done: false, entries: [{ label: 'A', own: false, latencyMs: 0, rank: 1 }, { label: 'Mine', own: true, latencyMs: 41.7, rank: 2 }] });
  const [first, second] = await events;
  assert.ok(first.hashes.includes(H), 'the block card finds it under Core\'s spelling');
  assert.deepEqual(first.stratum.entries.map((e) => e.label), ['A']);
  assert.equal(first.stratum.core_ms, undefined, 'no Core stop before Core had the block');
  assert.deepEqual(second.stratum.entries.map((e) => [e.label, e.own]), [['A', false], ['Mine', true]]);
  assert.equal(second.stratum.core_ms, 30);
});

test('a page that connects late is sent the race so far at once', async () => {
  const [replayed] = await stratumEvents(1, 2000);
  assert.equal(replayed.stratum.entries.length, 2);
});
