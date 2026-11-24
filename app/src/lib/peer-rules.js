'use strict';

/**
 * The peers service's rules for a peer it has no word on - not connected, or
 * the service down: which kind of software a user agent names, whether that
 * kind can pass a block on, and whether its service bits put it on a chain of
 * its own. Read from src/data/peer-rules.json, which the peers service's own
 * test writes from peers/clients.go and peers/forks.go, so the two judge a
 * peer alike (one table, two readers; 24.11.2026). Nothing here decides
 * anything the file does not say.
 */

const rules = require('../data/peer-rules.json');

const EMPTY_NAME = new RegExp(rules.emptyName);
const BRACKETED = new RegExp(rules.bracketed, 'g');
const KINDS = rules.kinds.map((k) => ({ ...k, re: new RegExp(k.pattern, k.flags) }));
const FORK_KINDS = new Set(rules.forkKinds);

/** The kind the peers service would give this user agent, or null for none. */
function kindOf(agent) {
  if (!agent || EMPTY_NAME.test(agent)) return null;
  const bare = agent.replace(BRACKETED, '');
  const noMempool = agent.replace(/mempool/gi, '');
  for (const k of KINDS) {
    const s = k.reads === 'bare' ? bare : k.reads === 'no-mempool' ? noMempool : agent;
    if (k.re.test(s)) return { kind: k.kind, relays: k.relays };
  }
  return null;
}

// What the ranking says of software that cannot pass a block on, by kind.
const CANNOT_RELAY = {
  'Other chain': 'a client of another chain',
  'Research scanner': 'a research scanner',
  Crawler: 'a network crawler',
  Indexer: 'an address indexer',
  Wallet: 'a wallet',
  'Lightning wallet': 'a Lightning wallet',
  'Other light client': 'a light client',
};

/** Why this user agent's software can never deliver a block first, or null. */
function cannotRelayReason(agent) {
  const k = kindOf(agent);
  if (!k || k.relays) return null;
  return CANNOT_RELAY[k.kind] || 'software that passes no blocks on';
}

/**
 * The fork a peer's service bits name, or null - the peers service's forkOf.
 * Only for nodes and unknown software: a wallet or crawler repeating a bit it
 * saw elsewhere is no fork. A live peer whose headers reach the tip follows
 * this chain whatever it offers; tip is the highest synced_headers of any
 * live peer, and a peer not live keeps the bits alone.
 */
function otherChainOf(services, syncedHeaders = null, tip = 0, agent = '') {
  const k = kindOf(agent);
  if (!FORK_KINDS.has(k ? k.kind : 'Other')) return null;
  const list = typeof services === 'string' ? services.split(',') : [];
  if (syncedHeaders != null && syncedHeaders >= 0 && tip > 0 && tip - syncedHeaders <= rules.forkTipSlack) return null;
  const lower = (agent || '').toLowerCase();
  for (const b of rules.forkBits) {
    if (!list.includes(b.service)) continue;
    if (b.exceptAgent && lower.startsWith(b.exceptAgent)) continue;
    return b.fork;
  }
  return null;
}

module.exports = { kindOf, cannotRelayReason, otherChainOf, examples: rules.examples };
