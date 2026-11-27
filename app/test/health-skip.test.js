'use strict';

// The stratum worker skips a heartbeat while a race is timed - but never two
// in a row, or a pool keeping races open all the time would starve the
// healthcheck and get the worker restarted in a loop (review, 26.11.2026).

const test = require('node:test');
const assert = require('node:assert/strict');
const health = require('../src/lib/health');

test('a skipped beat is followed by a written one, whatever skip says', (t) => {
  t.mock.timers.enable({ apis: ['setInterval'] });
  let writes = 0;
  const db = { instance: { prepare: () => ({ run: () => { writes += 1; } }) } };
  const logger = { debug() {} };
  const stop = health.start(db, 'test', logger, undefined, () => true);
  assert.equal(writes, 1, 'the first beat at once');
  for (let i = 0; i < 6; i++) t.mock.timers.tick(health.HEARTBEAT_INTERVAL_MS);
  assert.equal(writes, 4, 'every second beat written while a race is always open');
  stop();
});
