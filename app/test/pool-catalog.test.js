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
  // No network in the tests (as in dashboard-api.test.js): every name resolves.
  require('dns').promises.lookup = async () => ({ address: '192.0.2.1', family: 4 });
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

test('a tick adds the pool under the list\'s name, an untick takes it out but keeps its numbers', async () => {
  const p = list.pools.find((x) => x.host === 'jp-btc.solofury.com');
  let r = await api('/api/pools/catalog', { host: p.host, port: p.port, enabled: true });
  assert.equal(r.status, 200);
  const row = () => db.instance.prepare('SELECT * FROM stratum_pool WHERE host = ?').get(p.host);
  assert.equal(row().label, p.name);
  assert.equal(row().enabled, 1);
  assert.equal(row().is_default, 1, 'a public pool, not the owner\'s own');
  // A race it took part in: an untick keeps it, out of the race and the table.
  const race = db.instance.prepare('INSERT INTO stratum_race (prevhash, created_at) VALUES (?, ?)').run('ab'.repeat(32), Date.now()).lastInsertRowid;
  db.instance.prepare('INSERT INTO stratum_observation (race_id, pool_id, latency_ms, rank) VALUES (?, ?, 0, 1)').run(race, row().id);
  r = await api('/api/pools/catalog', { host: p.host, port: p.port, enabled: false });
  assert.equal(r.status, 200);
  assert.deepEqual([row().chosen, row().enabled], [0, 0], 'kept, unticked');
  assert.equal(db.instance.prepare('SELECT COUNT(*) AS n FROM stratum_observation WHERE race_id = ?').get(race).n, 1);
  assert.ok(!(await api('/api/pools')).body.some((x) => x.host === p.host), 'not in the table, not even "Off"');
  let shown = (await api('/api/pools/catalog')).body.pools.find((x) => x.host === p.host);
  assert.deepEqual([shown.enabled, shown.paused, shown.id], [false, false, null]);
  assert.equal((await api('/api/pools/catalog')).body.racing, 0);
  // The table's switch cannot bring it back past the list.
  const id = row().id;
  r = await fetch(`${baseUrl}/api/pools/${id}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ enabled: true }) });
  assert.equal(r.status, 404);
  r = await api('/api/pools/catalog', { host: p.host.toUpperCase(), port: p.port, enabled: true });
  assert.deepEqual([row().chosen, row().enabled], [1, 1], 'ticked again: back, with its numbers');
  assert.equal(row().id, id);
  const back = (await api('/api/pools')).body.find((x) => x.host === p.host);
  assert.equal(back.seen, 1);
  assert.equal(db.instance.prepare('SELECT COUNT(*) AS n FROM stratum_pool WHERE host = ?').get(p.host).n, 1);
});

test('paused by the table\'s switch: still ticked, marked paused; an untick takes it out', async () => {
  const p = list.pools.find((x) => x.host === 'jp-btc.solofury.com');
  const id = db.instance.prepare('SELECT id FROM stratum_pool WHERE host = ?').get(p.host).id;
  let r = await fetch(`${baseUrl}/api/pools/${id}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ enabled: false }) });
  assert.equal(r.status, 200);
  let shown = (await api('/api/pools/catalog')).body.pools.find((x) => x.host === p.host);
  assert.deepEqual([shown.enabled, shown.paused, shown.id], [true, true, id]);
  // A tick on it changes nothing: it stays paused.
  await api('/api/pools/catalog', { host: p.host, port: p.port, enabled: true });
  assert.equal(db.instance.prepare('SELECT enabled FROM stratum_pool WHERE id = ?').get(id).enabled, 0);
  await api('/api/pools/catalog', { host: p.host, port: p.port, enabled: false });
  shown = (await api('/api/pools/catalog')).body.pools.find((x) => x.host === p.host);
  assert.deepEqual([shown.enabled, shown.paused, shown.id], [false, false, null]);
});

test('paused, then unticked and ticked again: back running, under the list\'s name', async () => {
  const p = list.pools.find((x) => x.host === 'jp-btc.solofury.com');
  const row = () => db.instance.prepare('SELECT * FROM stratum_pool WHERE host = ?').get(p.host);
  await api('/api/pools/catalog', { host: p.host, port: p.port, enabled: true });
  await fetch(`${baseUrl}/api/pools/${row().id}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ enabled: false }) });
  await api('/api/pools/catalog', { host: p.host, port: p.port, enabled: false });
  db.instance.prepare('UPDATE stratum_pool SET label = ? WHERE host = ?').run('An old name', p.host);
  await api('/api/pools/catalog', { host: p.host, port: p.port, enabled: true });
  assert.deepEqual([row().chosen, row().enabled, row().label], [1, 1, p.name]);
  // Unticked, then the same address added by hand: back, not "already exists".
  await api('/api/pools/catalog', { host: p.host, port: p.port, enabled: false });
  const r = await api('/api/pools', { label: 'Mine', host: p.host, port: p.port });
  assert.equal(r.status, 200);
  assert.deepEqual([row().chosen, row().enabled, row().label, row().is_default], [1, 1, 'Mine', 1], 'a pool on the list stays a list pool');
  // Removed in the table: out, numbers kept, a tick in the list brings it back.
  const id = row().id;
  let d = await fetch(`${baseUrl}/api/pools/${id}`, { method: 'DELETE', headers: { 'Content-Type': 'application/json' } });
  assert.equal(d.status, 200);
  assert.deepEqual([row().chosen, row().enabled, row().id], [0, 0, id]);
  d = await fetch(`${baseUrl}/api/pools/${id}`, { method: 'DELETE', headers: { 'Content-Type': 'application/json' } });
  assert.equal(d.status, 404, 'already out');
  await api('/api/pools/catalog', { host: p.host, port: p.port, enabled: true });
  assert.equal(row().chosen, 1);
  await api('/api/pools/catalog', { host: p.host, port: p.port, enabled: false });
});

test('a pool in the table in another spelling is not added a second time', async () => {
  // A 1.0 database kept the host as typed; the form now writes it in lower case.
  db.instance.prepare(`INSERT INTO stratum_pool (label, host, port, enabled, is_default, created_at) VALUES ('Legacy', 'Legacy.Example', 3333, 1, 0, 1)`).run();
  const r = await api('/api/pools', { label: 'Again', host: 'legacy.example', port: 3333 });
  assert.equal(r.status, 409);
  assert.equal(db.instance.prepare("SELECT COUNT(*) AS n FROM stratum_pool WHERE lower(host) = 'legacy.example'").get().n, 1);
  db.instance.prepare("DELETE FROM stratum_pool WHERE lower(host) = 'legacy.example'").run();
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
  // Paused by the table's switch: it no longer counts, and one more can race.
  const on = (await api('/api/pools/catalog')).body.pools.find((p) => p.enabled);
  await fetch(`${baseUrl}/api/pools/${on.id}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ enabled: false }) });
  r = await api('/api/pools/catalog', { host: off.host, port: off.port, enabled: true });
  assert.equal(r.status, 200, 'one paused, one more can race');
  // The table's own switch keeps to the same limit.
  r = await fetch(`${baseUrl}/api/pools/${on.id}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ enabled: true }) });
  assert.equal(r.status, 409);
  // An untick frees a place too.
  const other = (await api('/api/pools/catalog')).body.pools.find((p) => p.enabled && !p.paused);
  await api('/api/pools/catalog', { host: other.host, port: other.port, enabled: false });
  r = await fetch(`${baseUrl}/api/pools/${on.id}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ enabled: true }) });
  assert.equal(r.status, 200);
});
