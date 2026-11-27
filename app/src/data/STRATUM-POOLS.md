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

26.11.2026, a search for missing regional servers: Parasite's US servers in Wisconsin and Virginia added. Trustpool's address now answers from Moscow where it answered from Singapore the day before, so it is marked routed. The other names found (Kryptex US/BR/SG/AE, CloverPool EU/SG, BitFuFu Africa, Poolin Beijing/Frankfurt, SoloPool.com la/west/useast, SECPOOL ru2 and more) led to servers already listed or behind the same distributor. 85 on the list.

27.11.2026, the same search done more thoroughly: HeliosPool Australia, US East and US West, and LetsMine.it UK, DE, France and Poland added. The geo database puts HeliosPool US West in Virginia and swaps LetsMine's gb1 and de1; the operators' names were kept, backed by a 70 ms longer round trip for US West and a route into London for gb1. AtlasPool answers on one name from nine sites (anycast), so it is marked routed. Other ports on listed servers (TLS, high difficulty, rental) were left out, since they reach the same server. 92 on the list.

27.11.2026, a search for pools not on the list at all: 29 servers added after a live check (Cruxpool, Cédric Crispin in Thailand, Grand Pool in Kazakhstan, France and the US, NMMiner, 1Miner.Net, Les Chauffagistes, SoloPool.fr, 69Pool, EchoPool, BCMonster, Rabid Pool, Molepool, SoloPool.eu, MySoloPool in Oman, XP Pool in India, X39 Mining, MergedHash, SoloMineLab, FenixPool, BTC PoW Lab, Firepool, Sunnydecree, Coinovo, Solo100, SoloMiningHQ). Cruxpool's five regional names all led to one server in Germany, so it is listed once. Open Source Miners shares Bitaxe Pool's server; VargaMesh refuses the password the race sends; Solo-Pool.org, MisterFix and PyBLØCK did not answer. Several of the new ones run on home internet lines. 121 on the list.
