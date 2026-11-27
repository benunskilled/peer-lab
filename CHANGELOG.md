# Changelog

## 1.1.0 – 2026-11-28

- Stratum Race: choose from 128 public pools, solo and shared, from 25 countries; up to 20 race at once. Tabs for Racing, All, Solo and Shared, with search and a country filter. Tap a pool's tag to see what solo, shared and routed mean. A new install starts with none ticked; an existing one keeps its pools.
- Stratum Race: the block card shows the race pool by pool as jobs come in.
- Stratum Race: an unticked pool keeps its numbers and has them back when ticked again. Remove keeps the numbers too. Reset pool history under Data is what deletes.
- Stratum Race: a connection that goes quiet is replaced, and a pool that sends no job is tried less often. A pool still waiting for its first job is not charged a miss. After a restart, a pool one block behind no longer wins a race for the old block.
- Slot Fill: a first pill, Auto, counts every address it can use. It is on while no region is picked; a tap on it clears the picked regions.
- Labels say how sure they are: plain, "Probably" or "Named like". Many more programs are named.
- Fixes around peers added by hand and Core with its network switched off.

## 1.0.1 – 2026-11-20

- Stratum Race: a pool app found on this Umbrel is marked "✓ found" in the row of local pools, and one tap adds it.
- Slot Fill: the regions to pick from are always shown. Peers being tested are listed newest first. A peer that gives up its slot to one added by hand gets its own line in the log.
- Peers reached through Tor, I2P or Docker show that, instead of the proxy's own address.
- Three more programs are named: miningcore-btc-solo (a solo pool), DNN Node and pcoc Bari.
- A peer calling itself "/Satoshi:" without the closing "/" is shown as Disguised: Core always writes it.
- The jump bar is no longer see-through.
- Kind pills line up on tablet-width cards, and with a filter on they show the same line as on a phone, such as how much a syncing peer has taken.

## 1.0.0 – 2026-11-16

First release. Peer Lab joins Bitcoin Lab and Peer Map in one app: the peer list with its checks and the map, the first-block count per peer, Slot Fill and Stratum Race. It starts with its own empty data.
