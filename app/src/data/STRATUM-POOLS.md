# Stratum Race: the public pools to choose from

`stratum-pools.json` is the list the Stratum Race card offers under "Choose pools". A fresh install races none of them; the user ticks the ones they want (at most 20 at once). Installs from before 25.11.2026 keep the eight pools they were given, which are on this list too (`"since": "1.0.0"`).

## What is on it

Public Bitcoin pools that send work to anyone over plain stratum V1 (TCP): they accept `mining.authorize` with a Bitcoin address they have never seen (`1BitcoinEaterAddressDontSendf59kuE.worker1`) and send `mining.notify` for the current block. Pools that need an account first (F2Pool, Foundry, Luxor, Kano at the time of checking) are not on it.

- `name`: what the results table shows. Unique, so two servers of one pool can be told apart.
- `country`, `region`: where the server address was on the day of checking, looked up by hand in Peer Lab's own `geo/` table, not what the pool says. `null` with `"routed": true` when the address belongs to a distributor in front of the pool and says nothing about where its servers are.
- `kind`: `solo` (the finder gets the block) or `shared`.
- `source`: the pool's own page that lists the address.
- `loginRefused`: the pool answers the login with "no" and sends jobs all the same. Only Braiins Pool (shared), on trial since 26.11.2026: a big name, kept while its jobs keep coming.

One entry per server: names that led to the same address on the day of checking (a pool's "Asia" host that resolved to its European server, for example) are listed once. The check script does not test this again.

## Checking it

    node scripts/check-stratum-pools.js

connects to every pool as Stratum Race does, for 45 seconds, and says which ones pass. Before a release: run it, take out what fails twice on different days, and set `checked`. A pass shows the pool works for a minute; whether it keeps a connection that never submits a share open for days only running it shows.

Collected 25.11.2026 from the pools' own pages (and two ChatGPT searches whose results were each connected to), 94 addresses tried, 64 on the list.

26.11.2026: a third search, 18 more addresses tried; 12 new servers added (the rest led to servers already listed). Three of their pages (Ghosthour, NerdMiner.de, Bitronics) show the address only with JavaScript; it was read there in a browser. EMCD Russia (kz.emcd.network) taken out after failing on 25.11. and 26.11.; 75 on the list.

26.11.2026, a search for big pools and the Americas: BTC-B2 Pools US East and West, PowerPool Solo US and EU, EMCD America (behind Cloudflare) and Braiins Pool (see `loginRefused`) added; 81 on the list. Foundry, F2Pool and Luxor still refuse an unregistered login, MARA Pool has no public endpoint, and Binance Pool, UltimusPool and SBI Crypto have closed.

26.11.2026, from the pool list of another open-source race tool (proofofmike/stratum-race): Blitzpool (Switzerland), HeliosPool (France) and SoloHash Canada added after a live check. OCEAN taken out: its terms say a user must be mining and must hold the keys to the address used, and the race does neither. Anyone may still add it by hand. 83 on the list.
