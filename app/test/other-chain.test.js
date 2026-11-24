'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const { otherChainOf: withAgent } = require('../src/lib/peer-rules');

// Unknown software ("Other") unless a test names one.
const otherChainOf = (services, synced = null, tip = 0, agent = '/some-node:1.0/') => withAgent(services, synced, tip, agent);

test('a fork bit alone marks another chain', () => {
  assert.equal(otherChainOf('NETWORK,WITNESS,UNKNOWN[2^28]'), 'BLAKE2b fork');
  assert.equal(otherChainOf('NETWORK,UNKNOWN[2^27]'), 'BIP110 fork');
  assert.equal(otherChainOf('NETWORK,UNKNOWN[2^27],UNKNOWN[2^28]'), 'BLAKE2b fork');
  assert.equal(otherChainOf('NETWORK,UNKNOWN[2^26]'), null);
  assert.equal(otherChainOf(null), null);
});

test("satd's bit 28 is intact blocks, not BLAKE2b", () => {
  const intact = 'WITNESS,NETWORK_LIMITED,UNKNOWN[2^28]';
  assert.equal(otherChainOf(intact, null, 0, '/satd:0.7.2/'), null);
  assert.equal(otherChainOf(intact, null, 0, '/Satoshi:29.4.1/Knots:20260508/'), 'BLAKE2b fork');
  assert.equal(otherChainOf('NETWORK,UNKNOWN[2^27],UNKNOWN[2^28]', null, 0, '/satd:0.7.2/'), 'BIP110 fork');
});

test('headers at the tip say it follows this chain after all', () => {
  const bip110 = 'NETWORK,UNKNOWN[2^27]';
  assert.equal(otherChainOf(bip110, 972597, 972597), null);
  assert.equal(otherChainOf(bip110, 972591, 972597), null);
  assert.equal(otherChainOf(bip110, 972590, 972597), 'BIP110 fork');
  assert.equal(otherChainOf(bip110, 961170, 972597), 'BIP110 fork');
  // Not known yet, or no live tip to compare with: the bit decides.
  assert.equal(otherChainOf(bip110, -1, 972597), 'BIP110 fork');
  assert.equal(otherChainOf(bip110, null, 972597), 'BIP110 fork');
  assert.equal(otherChainOf(bip110, 972597, 0), 'BIP110 fork');
});

test('a peer with no name is not judged by its bits, as in the peers service', () => {
  for (const nothing of ['', null, '/', '//', '/:1.0/']) {
    assert.equal(withAgent('NETWORK,UNKNOWN[2^27]', null, 0, nothing), null, JSON.stringify(nothing));
  }
});
