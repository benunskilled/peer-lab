#!/usr/bin/env node
'use strict';

// Builds src/data/mining-pools.json from two public lists (see
// src/data/MINING-POOLS.md):
//   - mempool/mining-pools pools-v2.json, the maintained one: the main list;
//   - bitcoin-data/mining-pools pools.json (generated branch, not rebuilt since
//     June 2026): only what mempool lacks, kept apart as "fallback". pool-id.js
//     asks it only when nothing in the main list matches, so a stale address
//     there cannot beat a pool's own tag ("ckpool" stays for solo CKPool).
// A pool mempool renamed (BTC.com is CloverPool) gets the new name in the
// fallback too, read off the tags both lists share.
//   node scripts/mining-pools.js

const fs = require('fs');
const path = require('path');

const MEMPOOL = 'https://raw.githubusercontent.com/mempool/mining-pools/master/pools-v2.json';
const BITCOIN_DATA = 'https://raw.githubusercontent.com/bitcoin-data/mining-pools/generated/pools.json';
const OUT = path.join(__dirname, '..', 'src', 'data', 'mining-pools.json');

async function get(url) {
  const res = await fetch(url);
  if (!res.ok) throw new Error(`${url}: HTTP ${res.status}`);
  return res.json();
}

// mempool's pools-v2.json: [{ name, link, tags: [], addresses: [] }, ...]
function fromMempool(list) {
  if (!Array.isArray(list)) throw new Error('pools-v2.json is not a list');
  const out = { payout_addresses: {}, coinbase_tags: {} };
  for (const pool of list) {
    const entry = { name: pool.name, link: pool.link || '' };
    for (const tag of pool.tags || []) out.coinbase_tags[tag] = entry;
    for (const address of pool.addresses || []) out.payout_addresses[address] = entry;
  }
  return out;
}

// Old name -> mempool's name, where both lists hold the same tag under
// different names and every shared tag of that old name agrees. Tags only:
// a tag is the pool's own name for itself, while an address mempool moved to
// another pool (three of CKPool's went to MiningKings) says nothing about a
// rename.
function renames(base, older) {
  const seen = {};
  for (const [k, v] of Object.entries(older.coinbase_tags || {})) {
    if (!base.coinbase_tags[k]) continue;
    (seen[v.name] = seen[v.name] || new Set()).add(base.coinbase_tags[k].name);
  }
  for (const [from, s] of Object.entries(seen)) if (s.has(from)) delete seen[from];
  return Object.fromEntries(Object.entries(seen).filter(([, s]) => s.size === 1).map(([from, s]) => [from, [...s][0]]));
}

const sorted = (o) => Object.fromEntries(Object.entries(o).sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0)));

function merge(base, older) {
  const names = renames(base, older);
  // The new name's link too, taken from the main list.
  const links = {};
  for (const key of ['payout_addresses', 'coinbase_tags']) for (const v of Object.values(base[key])) links[v.name] = links[v.name] || v.link;
  const fallback = { payout_addresses: {}, coinbase_tags: {} };
  for (const key of Object.keys(fallback)) {
    for (const [k, v] of Object.entries(older[key] || {})) {
      if (k in base[key]) continue;
      const name = names[v.name] || v.name;
      fallback[key][k] = { name, link: names[v.name] ? links[name] || '' : v.link || '' };
    }
    fallback[key] = sorted(fallback[key]);
  }
  return { payout_addresses: sorted(base.payout_addresses), coinbase_tags: sorted(base.coinbase_tags), fallback };
}

async function main() {
  const base = fromMempool(await get(MEMPOOL));
  const older = await get(BITCOIN_DATA);
  const out = merge(base, older);
  fs.writeFileSync(OUT, JSON.stringify(out, null, 2) + '\n');
  for (const key of ['payout_addresses', 'coinbase_tags']) {
    console.log(`${key}: ${Object.keys(out[key]).length} from mempool, ${Object.keys(out.fallback[key]).length} fallback from bitcoin-data`);
  }
  console.log('renamed in the fallback:', JSON.stringify(renames(base, older)));
}

if (require.main === module) main().catch((e) => { console.error(e.message); process.exit(1); });

module.exports = { fromMempool, merge, renames };
