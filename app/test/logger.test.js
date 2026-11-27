'use strict';

// A log line never ends the process (R-2611-05: a deeply nested value from a
// pool overflowed JSON.stringify and the worker exited).

const test = require('node:test');
const assert = require('node:assert/strict');

process.env.LOG_LEVEL = 'info';
const logger = require('../src/lib/logger').make('test');

test('details that cannot be written are said so, not thrown', (t) => {
  const lines = [];
  t.mock.method(console, 'log', (l) => lines.push(l));
  let deep = [];
  for (let i = 0; i < 200_000; i++) deep = [deep];
  const cycle = {};
  cycle.self = cycle;
  assert.doesNotThrow(() => logger.warn('deep', { error: deep }));
  assert.doesNotThrow(() => logger.info('cycle', cycle));
  assert.equal(lines.length, 2);
  assert.match(lines[1], /details could not be written/);
});
