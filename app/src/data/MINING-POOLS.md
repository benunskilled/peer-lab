# mining-pools.json

Which pool mined a block is read off the block itself — the coinbase
transaction carries the address the reward is paid to and a short text the
miner writes. Matching either against this list is how it is done without
asking any service; see [`src/lib/pool-id.js`](../lib/pool-id.js).

Sources, both MIT licensed, merged by [`scripts/mining-pools.js`](../../scripts/mining-pools.js):

- **[mempool/mining-pools](https://github.com/mempool/mining-pools)**
  (© mempool and contributors), `pools-v2.json`: the list mempool.space
  uses and keeps up to date. It is the base, and where both lists name the
  same tag or address, its name wins (BTC.com is CloverPool there, Ocean.xyz
  is OCEAN).
- **[bitcoin-data/mining-pools](https://github.com/bitcoin-data/mining-pools)**
  (© 2019–2021 btc.com, © 2021– 0xB10C and contributors), `pools.json` from
  its `generated` branch, last rebuilt on 30 June 2026: only for what the
  first list lacks, "ckpool" among it, so a solo CKPool block keeps its name.

Neither is only the big pools: solo CKPool, Bitsolo and a handful of
individuals who once found a block are in there too.

Current copy (24 November 2026): 224 coinbase tags and 246 payout addresses;
17 tags and 26 addresses of them come only from bitcoin-data. Checked against
ten consecutive blocks on one real node in September 2026: ten of ten
attributed — Foundry three times, F2Pool three times, ViaBTC, Braiins, MARA,
SpiderPool. Four of those matched by payout address, six only by tag, which
is why both are used.

## Refresh

```sh
node scripts/mining-pools.js
npm test
```

The tests check that the list still holds what it is supposed to: enough
entries to be the real thing, the pools that find most blocks today, and
"ckpool" from the older list.

## What it cannot do

A miner nobody has listed stays unknown, and that is a normal answer rather
than a fault — Peer Lab then shows the text out of the coinbase as the
miner's own words instead of putting a name on it.

The payout address is checked before the tag because it is where the money
actually went, but a pool paying through a third party will not match it. The
tag is text the miner writes himself: good enough to attribute blocks, and
still a claim rather than a measurement.

## mining-pools-extra.json

Coinbase tags and payout addresses neither list above has yet, kept apart so a refresh never
overwrites them; on the same tag, the extra list wins. Most are pools that run
ckpool, which writes "ckpool" into every coinbase: without a tag of their own,
Satoshi Radio or mutatrum would show as CKPool. The longest
tag wins, so "/pool.mutatrum.com/" beats "ckpool" in the same text.

Where each tag comes from:

- AtlasPool, Satoshi Radio, SoloHash, SoloMining.de, 2Miners, mutatrum,
  Blitzpool, MintSolo: the text in a mining job each pool
  handed out to a stratum subscribe, October 2026.
- SoloLuck, "/sololuck.io/": the pool's own entry in an open pull request to
  the older list ([bitcoin-data/mining-pools#132](https://github.com/bitcoin-data/mining-pools/pull/132),
  tag only, a ckpool-based solo pool with no block yet in November 2026).
- F2Pool's lowercase tags "/f2pool/m" and "/f2pool.kz/c" (F2Pool's
  Kazakhstan branch): the list has only "F2Pool", and the match is exact.
  Both pay 546 sat to 1K6KoYC69NnafWJ7YgtrpwJxBLiijWqwa6 like F2Pool's other
  blocks (971868, 883290), and mempool.space names them F2Pool.
- NiceHash, bc1q85z0lxe86pkcg2fmppfnykat2375s0pnxjx9sc: block 932129 pays its
  whole reward there and says "/NiceHashMining/" in its coinbase; 971652 and
  971719 pay the same address with no text.
- NiceHash, bc1qevfvr7n7hqu56eu5cqunk22nuwds0dkjgqqxe6: eleven blocks from
  972547 (22 October 2026) on, no text in the coinbase. Transaction
  2dd5141f43db83117ac7149f25206c96e6328ccad174917d9668f3ff9731ec41 (block
  975056) spends six of their rewards together with two from the NiceHash
  address above, so one wallet holds both.
- NiceHash EasyMining, bc1q28d6w54p3xc03a2j5ppmkua09akep9vtke20ny: takes
  exactly 3 % of the reward in blocks 972373, 974087 and 974683, the finder
  the rest. NiceHash announced 972373 and 974087 as EasyMining blocks
  (x.com/NiceHashMining/status/2112875109207724109 and .../2116771269089656836).

When mempool/mining-pools takes one of them in, its entry here can go
(Braiins Solo, Parasite, Public Pool, FutureBit Apollo Solo and MARA's
"Sv2/MARA" went on 24 November 2026).
