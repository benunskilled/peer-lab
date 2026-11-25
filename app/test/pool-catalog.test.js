'use strict';

// Choose pools: the public pools to tick for Stratum Race (lib/pool-catalog.js,
// data/stratum-pools.json), and the two routes behind the ticks.

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('fs');
const os = require('os');
const path = require('path');

const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'peerlab-catalog-'));
process.env.SQLITE_PATH = path.join(tmpDir, 'test.db');
process.env.DATA_DIR = tmpDir;
process.env.LOG_LEVEL = 'error';
process.env.PEERS_HEALTH_URL = 'off';

const db = require('../src/lib/db');
const list = require('../src/data/stratum-pools.json');
const { server } = require('../src/dashboard-server');

let baseUrl;
test.before(async () => {
  db.open();
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  baseUrl = `http://127.0.0.1:${server.address().port}`;
});
test.after(() => {
  server.close();
  fs.rmSync(tmpDir, { recursive: true, force: true });
});

async function api(pathname, body) {
  const res = await fetch(`${baseUrl}${pathname}`, body === undefined ? {} : {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body),
  });
  return { status: res.status, body: await res.json().catch(() => null) };
}

test('the list: every pool once, with a country or marked routed, a kind, and where it is written', () => {
  assert.match(list.checked, /^\d{4}-\d{2}-\d{2}$/);
  const seen = new Set();
  const names = new Set();
  for (const p of list.pools) {
    const key = `${p.host}:${p.port}`;
    assert.ok(!seen.has(key), `${key} twice`);
    seen.add(key);
    assert.ok(!names.has(p.name), `name ${p.name} twice: the results table could not tell them apart`);
    names.add(p.name);
    assert.match(p.host, /^[a-z0-9.-]+$/, key);
    assert.ok(Number.isInteger(p.port) && p.port > 0 && p.port < 65536, key);
    assert.ok(['solo', 'shared'].includes(p.kind), key);
    if (p.routed) assert.equal(p.country, null, `${key}: routed means the country is not known`);
    else assert.match(p.country || '', /^[A-Z]{2}$/, key);
    // A page of the pool's own, or one of the eight older defaults.
    assert.ok((p.source && p.source.startsWith('https://')) || p.since === '1.0.0', `${key} has no source`);
  }
});

test('a fresh install races nothing, and the list shows every pool unticked', async () => {
  const { status, body } = await api('/api/pools/catalog');
  assert.equal(status, 200);
  assert.equal(body.racing, 0);
  assert.equal(body.pools.length, list.pools.length);
  assert.ok(body.pools.every((p) => !p.enabled && p.id === null));
});

test('a tick adds the pool under the list\'s name, an untick pauses it and keeps it', async () => {
  const p = list.pools.find((x) => x.host === 'jp-btc.solofury.com');
  let r = await api('/api/pools/catalog', { host: p.host, port: p.port, enabled: true });
  assert.equal(r.status, 200);
  const row = () => db.instance.prepare('SELECT * FROM stratum_pool WHERE host = ?').get(p.host);
  assert.equal(row().label, p.name);
  assert.equal(row().enabled, 1);
  assert.equal(row().is_default, 1, 'a public pool, not the owner\'s own');
  r = await api('/api/pools/catalog', { host: p.host, port: p.port, enabled: false });
  assert.equal(r.status, 200);
  assert.equal(row().enabled, 0, 'paused, its history kept');
  r = await api('/api/pools/catalog', { host: p.host.toUpperCase(), port: p.port, enabled: true });
  assert.equal(row().enabled, 1, 'ticked again: the same row');
  assert.equal(db.instance.prepare('SELECT COUNT(*) AS n FROM stratum_pool WHERE host = ?').get(p.host).n, 1);
});

test('only a pool on the list can be ticked', async () => {
  const r = await api('/api/pools/catalog', { host: 'evil.example', port: 3333, enabled: true });
  assert.equal(r.status, 400);
  assert.equal(db.instance.prepare("SELECT COUNT(*) AS n FROM stratum_pool WHERE host = 'evil.example'").get().n, 0);
  assert.equal((await api('/api/pools/catalog', { host: list.pools[0].host, port: list.pools[0].port })).status, 400, 'enabled must be said');
});

test('at most twenty race at once; paused ones do not count', async () => {
  const racing = () => db.instance.prepare('SELECT COUNT(*) AS n FROM stratum_pool WHERE enabled = 1').get().n;
  for (const p of list.pools) {
    if (racing() >= 20) break;
    await api('/api/pools/catalog', { host: p.host, port: p.port, enabled: true });
  }
  assert.equal(racing(), 20);
  const off = (await api('/api/pools/catalog')).body.pools.find((p) => !p.enabled);
  let r = await api('/api/pools/catalog', { host: off.host, port: off.port, enabled: true });
  assert.equal(r.status, 409);
  const on = (await api('/api/pools/catalog')).body.pools.find((p) => p.enabled);
  await api('/api/pools/catalog', { host: on.host, port: on.port, enabled: false });
  r = await api('/api/pools/catalog', { host: off.host, port: off.port, enabled: true });
  assert.equal(r.status, 200, 'one paused, one more can race');
  // The table's own switch keeps to the same limit.
  r = await fetch(`${baseUrl}/api/pools/${on.id}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ enabled: true }) });
  assert.equal(r.status, 409);
});
