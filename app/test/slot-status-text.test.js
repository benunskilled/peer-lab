'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('fs');
const path = require('path');
const vm = require('vm');

// app.js is a browser script; the status line is lifted out of the real file
// and run as it is written there.
function slotStatusText(extra = '') {
  const src = fs.readFileSync(path.join(__dirname, '../src/dashboard/public/app.js'), 'utf8');
  const start = src.indexOf('const minutes = ');
  const fnStart = src.indexOf('function slotStatusText', start);
  assert.ok(start > 0 && fnStart > 0, 'app.js must still define minutes and slotStatusText');
  const end = src.indexOf('\n}', fnStart) + 2;
  const prelude = "const MAX_MANUAL_PEERS = 8; const escapeHtml = (s) => String(s).replace(/[&<>\\x22']/g, (c) => `&#${c.charCodeAt(0)};`); const countryName = (cc) => ({ VE: 'Venezuela', PE: 'Peru' })[cc] || cc;";
  return vm.runInNewContext(`${prelude}\n${extra}\n${src.slice(start, end)}\nslotStatusText;`);
}

// The page draws the line again every half minute from the status the last
// pass wrote, up to ten minutes earlier; the time left is counted from then.
test('the time until an offline peer frees its slot counts down between passes', () => {
  const text = slotStatusText();
  const at = 1_000_000_000_000;
  const st = { at, state: 'full', slots: { used: 8, max: 8, offline: [{ where: 'Europe', freesInMs: 30 * 60 * 1000 }] } };
  assert.match(text(st, at).rest, /about 30 min/);
  assert.match(text(st, at + 10 * 60 * 1000).rest, /about 20 min/, 'ten minutes later, ten minutes less');
});

// The top row says what Slot Fill does now and, under it, why the candidate
// before went again (14.11.2026: it came and went in the list without a word).
test('while testing, the top row also says why the peer before went', () => {
  const text = slotStatusText();
  const at = 1_000_000_000_000;
  const st = {
    at, state: 'trial', slots: { used: 8, max: 8, offline: [] },
    trial: { address: 'x', startedAt: at, phase: 'connecting', decideBy: at + 5 * 60 * 1000, country: 'PE' },
    lastTry: { at: at - 1000, reason: 'no_connection', country: 'VE', region: 'South America' },
  };
  const t = text(st, at);
  assert.match(t.head, /Testing a new peer in Peru/);
  assert.match(t.rest, /An earlier peer in Venezuela couldn't be reached\./);
  // Plain text, escaped once where it goes into the page: escaped here too,
  // the trial rows showed "couldn&#39;t" (15.11.2026).
  assert.doesNotMatch(t.rest, /&#/);
  assert.doesNotMatch(text({ ...st, lastTry: undefined }, at).rest, /earlier peer/, 'nothing when nothing went');
});

test('a peer before on an unknown reason or place still reads as a sentence', () => {
  const text = slotStatusText();
  const at = 1_000_000_000_000;
  const st = { at, nextAt: at + 600000, state: 'looking', slots: { used: 7, max: 8, offline: [] }, look: { why: 'none_fit' },
    lastTry: { at, reason: 'something_new', country: null, region: null } };
  assert.match(text(st, at).rest, /An earlier peer was not kept\.$/);
});

// Up to three on trial, shown newest first: the earlier peer is said under the
// newest only, the top row (17.11.2026: "the peer before, in Australia" stood
// above a candidate in Australia still being tested).
test('with several on trial, only the newest says why an earlier peer went', () => {
  const text = slotStatusText();
  const at = 1_000_000_000_000;
  const old = { address: 'a', startedAt: at - 60000, phase: 'connecting', decideBy: at + 60000, country: 'PE' };
  const fresh = { address: 'b', startedAt: at, phase: 'connecting', decideBy: at + 120000, country: 'VE' };
  const st = { at, state: 'trial', slots: { used: 8, max: 8, offline: [] }, trial: old, trials: [old, fresh],
    lastTry: { at: at - 1000, reason: 'no_connection', country: 'VE', region: 'South America' } };
  assert.match(text(st, at, fresh).rest, /An earlier peer in Venezuela/);
  assert.doesNotMatch(text(st, at, old).rest, /earlier peer/);
});

test('the status text is plain text: a place with & or an apostrophe is not escaped in it', () => {
  const text = slotStatusText();
  const at = 1_000_000_000_000;
  const st = { at, state: 'full', slots: { used: 8, max: 8, offline: [{ where: "Côte d'Ivoire & Co", freesInMs: 30 * 60 * 1000 }] } };
  assert.match(text(st, at).rest, /Côte d'Ivoire & Co/);
});

// The status is from the last pass, up to ten minutes old; the manual list is
// fresh. A slot freed since (removed by hand) shows at once, and a peer the
// pass saw offline but that is back is not called offline (21.11.2026: the
// tile, the slot count and this line said three different things).
test('a slot freed since the last pass shows at once', () => {
  const text = slotStatusText(`const lastPeerRanking = Array.from({ length: 7 }, (_, i) => ({ address: 'a' + i, trusted: true, live: true })); const justGone = new Map();`);
  const st = { at: 1, state: 'full', slots: { used: 8, max: 8, offline: [] } };
  assert.equal(text(st, 2).head, '7 of 8 slots in use.');
});

test('a peer the last pass saw offline but that is back is not called offline', () => {
  const text = slotStatusText(`const lastPeerRanking = Array.from({ length: 8 }, (_, i) => ({ address: 'a' + i, trusted: true, live: true })); const justGone = new Map();`);
  const st = { at: 1, state: 'full', slots: { used: 8, max: 8, offline: [{ address: 'a3', where: 'South America', freesInMs: 3e6 }] } };
  const t = text(st, 2);
  assert.equal(t.head, 'All 8 slots are in use.');
  assert.doesNotMatch(t.rest, /offline/);
});
