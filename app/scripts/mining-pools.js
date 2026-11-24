#!/usr/bin/env node
'use strict';

// Builds src/data/mining-pools.json from two public lists (see
// src/data/MINING-POOLS.md):
//   - mempool/mining-pools pools-v2.json, the maintained one, as the base;
//   - bitcoin-data/mining-pools pools.json (generated branch), for the tags
//     and addresses mempool's list does not have, "ckpool" among them.
// Where both name the same tag or address, mempool's name wins.
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

function merge(base, older) {
  const out = { payout_addresses: {}, coinbase_tags: {} };
  for (const key of Object.keys(out)) {
    for (const [k, v] of Object.entries(older[key] || {})) out[key][k] = { name: v.name, link: v.link || '' };
    Object.assign(out[key], base[key]);
    out[key] = Object.fromEntries(Object.entries(out[key]).sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0)));
  }
  return out;
}

async function main() {
  const base = fromMempool(await get(MEMPOOL));
  const older = await get(BITCOIN_DATA);
  const out = merge(base, older);
  fs.writeFileSync(OUT, JSON.stringify(out, null, 2) + '\n');
  for (const key of Object.keys(out)) {
    const onlyOld = Object.keys(out[key]).filter((k) => !(k in base[key])).length;
    console.log(`${key}: ${Object.keys(out[key]).length} (${Object.keys(base[key]).length} from mempool, ${onlyOld} only in bitcoin-data)`);
  }
}

if (require.main === module) main().catch((e) => { console.error(e.message); process.exit(1); });

module.exports = { fromMempool, merge };
