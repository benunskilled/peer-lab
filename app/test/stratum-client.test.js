'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const { StratumPoolConnection } = require('../src/lib/stratum-client');

test('emits notify with prevhash on a well-formed mining.notify line', () => {
  const conn = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'test' });
  let captured = null;
  conn.on('notify', (payload) => { captured = payload; });

  const notify = {
    id: null,
    method: 'mining.notify',
    params: ['job1', 'deadbeef'.repeat(8), '', '', [], '20000000', '1d00ffff', '5f5e100', true],
  };
  conn._handleChunk(Buffer.from(`${JSON.stringify(notify)}\n`), process.hrtime.bigint());

  assert.ok(captured);
  assert.equal(captured.prevhash, 'deadbeef'.repeat(8));
  assert.equal(captured.cleanJobs, true);
});

// Both clocks travel with the job: hrtime for the gaps between pools, the wall
// clock for when it happened, taken at the same instant the chunk arrived.
test('a notify carries the wall-clock time of its arrival', () => {
  const conn = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'test' });
  let captured = null;
  conn.on('notify', (payload) => { captured = payload; });
  const notify = { method: 'mining.notify', params: ['j', 'ab'.repeat(32), '', '', [], '', '', '', false] };
  conn._handleChunk(Buffer.from(`${JSON.stringify(notify)}\n`), process.hrtime.bigint(), 1789900000123);
  assert.equal(captured.receivedAtMs, 1789900000123);
});

// Which job is the pool's opening state dump and which one is an announcement
// is something only this end can tell, from the connection it arrived on - so
// the client says, and stratum-race.js decides what to do about it.
test('a notify says whether it is the first of its connection', () => {
  const conn = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'test' });
  const seen = [];
  conn.on('notify', (payload) => seen.push(payload.firstAfterConnect));

  const line = (hash) => `${JSON.stringify({ method: 'mining.notify', params: ['j', hash, '', '', [], '', '', '', true] })}\n`;
  conn._handleChunk(Buffer.from(line('ab'.repeat(32))), process.hrtime.bigint());
  conn._handleChunk(Buffer.from(line('cd'.repeat(32))), process.hrtime.bigint());

  assert.deepEqual(seen, [true, false]);

  // A reconnect starts the count over - 'connect' resets it - because the pool
  // dumps its current job to the new session exactly as it did to the first.
  conn.notifyCount = 0;
  conn._handleChunk(Buffer.from(line('ef'.repeat(32))), process.hrtime.bigint());
  assert.deepEqual(seen, [true, false, true]);
});

test('ignores malformed JSON lines without throwing', () => {
  const conn = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'test' });
  assert.doesNotThrow(() => conn._handleChunk(Buffer.from('{not json\n'), process.hrtime.bigint()));
});

test('a line that is valid JSON but no object is ignored, not a crash', () => {
  // "null" parses fine; reading .method off it used to throw inside the socket handler.
  const conn = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'test' });
  for (const line of ['null', '5', '"x"', 'true']) {
    assert.doesNotThrow(() => conn._handleChunk(Buffer.from(`${line}\n`), process.hrtime.bigint()));
  }
});

test('buffers a message split across two chunks', () => {
  const conn = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'test' });
  let captured = null;
  conn.on('notify', (payload) => { captured = payload; });

  const hash = 'a'.repeat(64);
  const notify = { method: 'mining.notify', params: ['j', hash, '', '', [], '', '', '', false] };
  const line = `${JSON.stringify(notify)}\n`;
  const mid = Math.floor(line.length / 2);

  conn._handleChunk(Buffer.from(line.slice(0, mid)), process.hrtime.bigint());
  assert.equal(captured, null, 'should not fire until the line is complete');
  conn._handleChunk(Buffer.from(line.slice(mid)), process.hrtime.bigint());

  assert.ok(captured);
  assert.equal(captured.prevhash, hash);
});

test('ignores non-notify methods', () => {
  const conn = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'test' });
  let fired = false;
  conn.on('notify', () => { fired = true; });
  conn._handleChunk(Buffer.from(`${JSON.stringify({ id: 1, result: true, error: null })}\n`), process.hrtime.bigint());
  assert.equal(fired, false);
});

test('emits authorizeResult for the mining.authorize response (id 2)', () => {
  const conn = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'test' });
  let captured = null;
  conn.on('authorizeResult', (payload) => { captured = payload; });

  conn._handleChunk(Buffer.from(`${JSON.stringify({ id: 2, result: true, error: null })}\n`), process.hrtime.bigint());
  assert.deepEqual(captured, { ok: true, error: null });

  conn._handleChunk(Buffer.from(`${JSON.stringify({ id: 2, result: false, error: [24, 'unauthorized-worker', null] })}\n`), process.hrtime.bigint());
  assert.equal(captured.ok, false);
  assert.ok(captured.error);
});

test('a mining.notify whose prevhash is not a block hash is refused', () => {
  // The prevhash is an identity everything downstream trusts: it keys the open
  // race, the stratum_race row and the stale-job check. A pool is somebody
  // else's server, so the shape is checked here rather than assumed.
  const conn = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'test' });
  let fired = 0;
  let refused = 0;
  conn.on('notify', () => { fired += 1; });
  conn.on('protocolError', () => { refused += 1; });

  const send = (prevhash) => conn._handleChunk(
    Buffer.from(`${JSON.stringify({ method: 'mining.notify', params: ['j', prevhash, '', '', [], '', '', '', false] })}\n`),
    process.hrtime.bigint(),
  );

  send('not-a-hash');
  send('a'.repeat(63));
  send('a'.repeat(65));
  send('g'.repeat(64));
  send(null);
  send(12345);
  assert.equal(fired, 0, 'none of these open a race');
  assert.equal(refused, 6, 'and each one is reported rather than silently dropped');

  send('b'.repeat(64));
  assert.equal(fired, 1, 'a real one still gets through');
});

test('a pool that never sends a newline does not grow the buffer without limit', () => {
  const conn = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'test' });
  let reported = null;
  conn.on('protocolError', (info) => { reported = info; });
  // No socket assigned, so the destroy below is a no-op - what is under test
  // is that the buffer is released and the connection asked to go.
  for (let i = 0; i < 3; i += 1) {
    conn._handleChunk(Buffer.from('x'.repeat(64 * 1024)), process.hrtime.bigint());
  }

  assert.ok(reported, 'the pool is reported');
  assert.equal(reported.reason, 'line exceeds maximum size');
  assert.equal(conn.buffer.length, 0, 'and the buffer is not still holding it');
});

test('a single oversized line that arrives whole is refused too, not parsed', () => {
  const conn = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'test' });
  let reported = null;
  let parsed = 0;
  conn.on('protocolError', (info) => { reported = info; });
  conn.on('notify', () => { parsed += 1; });
  const huge = JSON.stringify({ method: 'mining.notify', params: ['x'.repeat(200 * 1024)] });
  conn._handleChunk(Buffer.from(huge + '\n'), process.hrtime.bigint());
  assert.ok(reported, 'the pool is reported');
  assert.equal(reported.reason, 'line exceeds maximum size');
  assert.equal(parsed, 0, 'and the line never reached the parser');
  assert.equal(conn.buffer.length, 0);
});

test('a long burst of complete lines is not mistaken for one oversized line', () => {
  const conn = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'test' });
  let fired = 0;
  let reported = 0;
  conn.on('notify', () => { fired += 1; });
  conn.on('protocolError', () => { reported += 1; });

  // Well over the limit in total, but every line is terminated.
  const line = `${JSON.stringify({ method: 'mining.notify', params: ['j', 'c'.repeat(64), '', '', [], '', '', '', false] })}\n`;
  conn._handleChunk(Buffer.from(line.repeat(2000)), process.hrtime.bigint());

  assert.equal(fired, 2000, 'every line is handled');
  assert.equal(reported, 0, 'and nothing is reported');
});

// A connection counts as working only once it brought a job: ckpool accepted
// the socket and took 20 s to two minutes to authorise (25.11.2026).
test('readySince is when the connection brought its first job, and only then', () => {
  const conn = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'test' });
  assert.equal(conn.readySince, null, 'no job yet');
  const line = (hash) => `${JSON.stringify({ method: 'mining.notify', params: ['j', hash, '', '', [], '', '', '', true] })}\n`;
  conn._handleChunk(Buffer.from(line('ab'.repeat(32))), process.hrtime.bigint(), 1789900000100);
  conn._handleChunk(Buffer.from(line('cd'.repeat(32))), process.hrtime.bigint(), 1789900000900);
  assert.equal(conn.readySince, 1789900000100, 'the first job, not the latest');
  conn.stop();
  assert.equal(conn.readySince, null, 'gone with the connection');
});

const notifyLine = (hash) => Buffer.from(`${JSON.stringify({ method: 'mining.notify', params: ['j', hash, '', '', [], '', '', '', false] })}\n`);

test('a pool silent for four of its usual gaps, and at least a minute, counts as gone', () => {
  const conn = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'test' });
  const t0 = 1789900000000;
  conn._handleChunk(notifyLine('ab'.repeat(32)), process.hrtime.bigint(), t0);
  conn._handleChunk(notifyLine('ab'.repeat(32)), process.hrtime.bigint(), t0 + 20_000);
  conn._handleChunk(notifyLine('ab'.repeat(32)), process.hrtime.bigint(), t0 + 40_000);
  assert.equal(conn.silenceLimitMs(), null, 'two gaps are not a rhythm yet');
  conn._handleChunk(notifyLine('ab'.repeat(32)), process.hrtime.bigint(), t0 + 60_000);
  assert.equal(conn.silenceLimitMs(), 80_000, 'every 20 s: four gaps');
  assert.equal(conn.silentTooLong(t0 + 60_000 + 79_000), false);
  assert.equal(conn.silentTooLong(t0 + 60_000 + 81_000), true);

  // A pool that sends work only on a new block is given forty minutes.
  const slow = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'slow' });
  for (let i = 0; i < 4; i++) slow._handleChunk(notifyLine('cd'.repeat(32)), process.hrtime.bigint(), t0 + i * 600_000);
  assert.equal(slow.silenceLimitMs(), 2_400_000);

  // And a fast one never less than a minute.
  const fast = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'fast' });
  for (let i = 0; i < 4; i++) fast._handleChunk(notifyLine('ef'.repeat(32)), process.hrtime.bigint(), t0 + i * 6_000);
  assert.equal(fast.silenceLimitMs(), 60_000);
});

test('a connection that held is dialled again at once, and the log says the pool closed it', async () => {
  const net = require('net');
  let accepted = 0;
  const sockets = [];
  const server = net.createServer((s) => { accepted += 1; sockets.push(s); });
  await new Promise((r) => server.listen(0, '127.0.0.1', r));
  const conn = new StratumPoolConnection({ host: '127.0.0.1', port: server.address().port, label: 'local' });
  const closed = [];
  conn.on('disconnect', (info) => closed.push(info));
  try {
    conn.start();
    await new Promise((r) => conn.once('connect', r));
    for (let i = 0; i < 50 && accepted < 1; i++) await new Promise((r) => setTimeout(r, 10));
    conn._connectedAt = Date.now() - 61_000; // it has stood for over a minute
    const back = new Promise((r) => conn.once('connect', r));
    sockets[0].end();
    await back;
    assert.equal(closed[0].reason, 'closed by the pool');
    assert.equal(closed[0].nextTryMs, 0);
    for (let i = 0; i < 50 && accepted < 2; i++) await new Promise((r) => setTimeout(r, 10));
    assert.equal(accepted, 2, 'the server saw the second connection');
  } finally {
    conn.stop();
    for (const s of sockets) s.destroy();
    server.close();
  }
});

test('a burst of jobs after each block does not teach a rhythm of a few hundred ms', () => {
  const conn = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'bursty' });
  const t0 = 1789900000000;
  for (let b = 0; b < 4; b++) {
    for (const d of [0, 300, 900]) conn._handleChunk(notifyLine('ab'.repeat(32)), process.hrtime.bigint(), t0 + b * 600_000 + d);
  }
  assert.equal(conn.silenceLimitMs(), 4 * (600_000 - 900), 'only the gaps between blocks count');
});

test('a connection that never brings a job is replaced too, after five minutes at the least', () => {
  const conn = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'quiet' });
  const t0 = 1789900000000;
  for (let i = 0; i < 4; i++) conn._handleChunk(notifyLine('ab'.repeat(32)), process.hrtime.bigint(), t0 + i * 20_000);
  // A new connection: nothing yet.
  conn._lastNotifyAt = null;
  conn._connectedAt = t0 + 100_000;
  assert.equal(conn.silentTooLong(t0 + 100_000 + 90_000), false, 'eighty seconds is its rhythm, but the first job may take longer');
  assert.equal(conn.silentTooLong(t0 + 100_000 + 301_000), true);
});

test('a pool new to this worker that never brings a job is replaced after five minutes too', () => {
  // No rhythm learned yet: it used to be let off until the six-hour idle
  // timeout (R-2611-05).
  const conn = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'new and quiet' });
  const t0 = 1789900000000;
  conn._connectedAt = t0;
  assert.equal(conn.silenceLimitMs(), null);
  assert.equal(conn.silentTooLong(t0 + 299_000), false);
  assert.equal(conn.silentTooLong(t0 + 301_000), true);
});

test('a deeply nested error in the reply to our login is passed on as short text', () => {
  const conn = new StratumPoolConnection({ host: 'example.invalid', port: 3333, label: 'nested' });
  let got;
  conn.on('authorizeResult', (r) => { got = r; });
  const depth = 50_000; // fits the line limit, parses, and overflowed the logger's JSON.stringify
  conn._handleLine(`{"id":2,"result":false,"error":${'['.repeat(depth)}${']'.repeat(depth)}}`, process.hrtime.bigint(), Date.now());
  assert.equal(got.ok, false);
  assert.equal(typeof got.error, 'string');
  assert.ok(got.error.length <= 200);
});

test('a connection closed for bringing no job waits longer each time, not dialled again at once', async () => {
  const net = require('net');
  const server = net.createServer(() => {}); // accepts, never answers
  await new Promise((r) => server.listen(0, '127.0.0.1', r));
  const conn = new StratumPoolConnection({ host: '127.0.0.1', port: server.address().port, label: 'mute' });
  const waits = [];
  conn.on('disconnect', (d) => waits.push(d.nextTryMs));
  conn.on('connect', () => {
    conn._connectedAt = Date.now() - 10 * 60_000; // stood ten minutes, no job
    conn._closeReason = 'no job for 600 s';
    conn.socket.destroy();
  });
  try {
    conn.start();
    for (let i = 0; i < 50 && waits.length < 1; i++) await new Promise((r) => setTimeout(r, 20));
    assert.ok(waits[0] >= 60_000, 'a minute at least, though it stood more than a minute');
    assert.ok(conn.reconnectDelayMs >= 60_000, 'and the next wait is a minute or more');
  } finally {
    conn.stop();
    server.close();
  }
});
