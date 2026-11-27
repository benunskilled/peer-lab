'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('fs');
const path = require('path');

/**
 * The dashboard marks a peer whose software cannot pass a block on "never
 * first"; the peer list marks the same peers with ⊘. The two used to keep
 * their own copies of the rules and drifted once - one peer a relay in the
 * list and a "never first" wallet in the ranking. Since 24.11.2026 there is
 * one table (peers/clients.go); the peers service's test writes it to
 * src/data/peer-rules.json and fails when the file is out of date, and this
 * test checks the dashboard reads that file the way Go does: every example in
 * it carries the kind Go gave it.
 */

const rules = require('../src/lib/peer-rules');

test('the dashboard reads every rule the way the peers service does', () => {
  const examples = Object.entries(rules.examples);
  assert.ok(examples.length >= 250, 'the examples must actually be there');
  for (const [agent, goKind] of examples) {
    const k = rules.kindOf(agent);
    // Go's "No name" is a name with nothing in it; here that is no kind.
    const kind = k ? k.kind : goKind === 'No name' ? 'No name' : 'Other';
    assert.equal(kind, goKind, `${JSON.stringify(agent)} is read differently here`);
  }
});

test('a peer that sent no user agent at all is given the benefit of the doubt', () => {
  for (const nothing of ['', null, undefined, '//', '/:1.1.0-dev+c1771608d/']) {
    assert.equal(rules.cannotRelayReason(nothing), null, 'not knowing is not the same as knowing it cannot');
  }
});

test('a word in an operator\'s brackets does not change what a Core node is', () => {
  for (const agent of ['/Satoshi:29.0.0(my-monitor)/', '/Satoshi:29.0.0(electrum-backend)/', '/Satoshi:28.0.0(neutrino)/']) {
    assert.equal(rules.cannotRelayReason(agent), null, agent);
  }
  // "pool" inside "mempool" is no pool, a pool in the brackets is.
  assert.equal(rules.kindOf('/Satoshi:28.0.0(mempool.space)/').kind, 'Node (Core)');
  assert.equal(rules.kindOf('/Satoshi:29.1.0(PyBLOCK-POOL)/Knots:20250903/').kind, 'Pool node');
});

test('the reason shown names the family the Go side named', () => {
  const expectations = [
    ['/Bitcoin XT:0.11.0/', 'another chain'],
    ['/node.cs.ac.uk:1.0/', 'research scanner'],
    ['/uni-goettingen.de/', 'research scanner'],
    ['/mempool.space:1.0/', 'indexer'],
    ['/electrumx:1.16/', 'indexer'],
    ['/EMPORION:0.3/', 'light client'],
    ['/SYMMARTYS:0.3/', 'light client'],
    ['/wuji-headers:0.1/', 'light client'],
    ['/Bitflash:btc-light/', 'light client'],
    ['/21datalab-crawler:0.1/', 'crawler'],
    ['nebula/2.4.1-65580d4', 'crawler'],
    ['/bip110-observer:0.1/', 'crawler'],
    ['/bitcoinj:0.15/', 'wallet'],
    ['/neutrino:0.17.1/', 'Lightning wallet'],
  ];
  for (const [agent, fragment] of expectations) {
    const reason = rules.cannotRelayReason(agent) || '';
    assert.ok(reason.includes(fragment), `${agent}: "${reason}" should mention "${fragment}"`);
  }
});

test('the page holds no rules of its own', () => {
  const page = fs.readFileSync(path.join(__dirname, '..', 'src', 'dashboard', 'public', 'app.js'), 'utf8');
  assert.equal(/KIND_RULES|cannotRelayReason|kindRuleIndex/.test(page), false);
});
