package main

import (
	"regexp"
	"strings"
	"time"
)

// What a piece of software is, in a sentence, for the peers somebody is
// likely to ask about. Only software that says who it is: the kind already
// covers the rest.
// label is the text cut to one line for the list; name, where set, is the
// chip the peer is counted under, for one program under many names.
//
// How sure a sentence may sound, the same for every entry (13.11.2026):
//   - plainly what it is: the exact string is in the project's source, or
//     the tag in brackets after Core's name is in the project's own setup
//     (its bitcoin.conf, install script or guide);
//   - "Probably ...": something technical points there (service bits only
//     that software sets, a fixed name in a source) but the name alone is
//     not proof;
//   - "Named like ...": only the name matches (23.11.2026: "Names ...",
//     "Named after ..." and "Labelled by ..." are gone, three levels only).
//
// Under one wording, the same as README's "How sure a label is".
//
// What one node saw a peer do is never the evidence here: it goes in the
// comment, with its date, and the page shows it live under the sentence.
var aboutRules = []aboutRule{
	// sipa's seeder sends "/bitcoin-seeder:0.01/" (bitcoin.cpp in
	// sipa/bitcoin-seeder, L97 at 92d2b46). Any other name with "seeder" in it
	// only says so.
	// Source: sipa/bitcoin-seeder bitcoin.cpp L97 @91e021f (checked 2026-11-24)
	{regexp.MustCompile(`^/bitcoin-seeder:`), "A DNS seeder: collects addresses of reachable nodes for new nodes to start from", "DNS seeder", ""},
	// generic-seeder: sAppName "generic-seeder" (main.cpp L23), sent as
	// "/generic-seeder/" (bitcoin.cpp L80) in team-exor/generic-seeder, a seeder
	// for Bitcoin and coins cloned from it (checked 23.11.2026).
	// Source: team-exor/generic-seeder main.cpp L23 @6cf84cd (checked 2026-11-24)
	{regexp.MustCompile(`^/generic-seeder/`), "generic-seeder, a DNS seeder for Bitcoin and coins cloned from it", "generic-seeder", ""},
	// ua = b"\x10/mrm-census:0.2/" (crawler.py L40 in toomanymao/mrm-node-census,
	// commit 230796b of 26.07.2026, checked 23.11.2026).
	// Source: toomanymao/mrm-node-census crawler.py L40 @230796b (checked 2026-11-24)
	{regexp.MustCompile(`^/mrm-census:`), "mrm-node-census, a small crawler that maps reachable nodes, onion ones too", "mrm-node-census crawler", ""},
	{regexp.MustCompile(`(?i)seeder|dnsseed`), "Named like a DNS seeder, which collects addresses of reachable nodes for new nodes to start from", "Named like a DNS seeder", ""},
	// BTC Nodes (btcnodes.io), Rodrigo Martínez's successor to Bitnodes
	// (bitcointalk.org topic 5582226). Its public crawler config still says
	// /bitnodes.io:0.3/; that "/BTC-Nodes:<date>/Sonar/", ".../Probe/" and
	// ".../Delegator/" are its own was not found in a source, so only the name.
	// Before the census rule, which would take Sonar.
	{regexp.MustCompile(`^/BTC-Nodes:`), "Named like BTC Nodes (btcnodes.io), the successor to Bitnodes that maps reachable nodes", "Named like BTC Nodes", ""},
	// 21 Data Lab's own crawler speaks Bitcoin P2P and asks nodes for their
	// peer lists (21datalab.com/de/methodik); its user agent is not given
	// there, so only the name (11.11.2026).
	{regexp.MustCompile(`^/21datalab-crawler:`), "Named like 21 Data Lab's crawler, which counts reachable Bitcoin nodes (21datalab.com)", "Named like 21 Data Lab", ""},
	// Nebula, Dennis Trautwein's network crawler (ProbeLab), crawls Bitcoin
	// since 2.4.0 (github.com/dennis-tra/nebula/releases). Who runs a given
	// crawl it does not say. "nebula/2.4.1-…" and "/nebula:v3.0.0-…/".
	// Source: dennis-tra/nebula bitcoin/crawler.go L345 @2ace7f4 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/?nebula[/:]`), "Nebula, a network crawler by Dennis Trautwein (ProbeLab) that also maps Bitcoin nodes", "Nebula crawler", ""},
	// A Knots node under its operator's own name, offering service bits 27
	// and 28. Bit 27 is BIP110's (src/protocol.h at knots20260508), which
	// Bitcoin Purity sets too; bit 28, NODE_BLAKE2B, is only on Knots'
	// 29.x-knots branch (src/protocol.h at 5811235), not in knots20260508.
	{regexp.MustCompile(`^/GrayersDad:`), "Probably Bitcoin Knots under a name its operator chose", "Probably Knots", ""},
	// Before the census rule, which would take it for "monitor" (koyn-sync).
	// Source: elkrem/koyn-sync src/main/java/io/elkrem/koynsync/Main.java L257 @c9b9ef1 (checked 2026-11-24)
	{regexp.MustCompile(`/PeerMonitor:`), "Koyn's header tool, which collects block headers for the Koyn Arduino library", "Koyn header tool", ""},
	// Census and seed monitors whose names hold one of the census words, each
	// with its source (14.11.2026): USER_AGENT in lib/observatory.rb
	// (azuchi/btc-node-observatory), OUR_USER_AGENT in dns_monitor/handshake.py
	// (ndungudedan/dns-monitor, and willcl-ark's fork), user_agent in
	// config.toml (echennells/dns-seed-monitor).
	// Source: azuchi/btc-node-observatory lib/observatory.rb L12 @36b19bf (checked 2026-11-24)
	{regexp.MustCompile(`^/btc-node-observatory:`), "Bitcoin Node Observatory, Shigeyuki Azuchi's daily census of reachable nodes", "Node Observatory census", ""},
	// Source: ndungudedan/dns-monitor dns_monitor/handshake.py L35 @e293a73 (checked 2026-11-24)
	{regexp.MustCompile(`^/dns-monitor:`), "dns-monitor, a tool that checks the nodes Bitcoin's DNS seeds hand out", "DNS seed monitor", ""},
	// user_agent = "/DCTRL-YVR-seed-monitor:0.1.0/" (config.toml L14 in
	// echennells/dns-seed-monitor, 23.11.2026).
	// Source: echennells/dns-seed-monitor config.toml L14 @28bc2e7 (checked 2026-11-24)
	{regexp.MustCompile(`^/DCTRL-YVR-seed-monitor:`), "DCTRL YVR's DNS seed monitor: checks the nodes Bitcoin's DNS seeds hand out", "DCTRL seed monitor", ""},
	// Only the word: no source for any of these names.
	{regexp.MustCompile(`(?i)census|survey|observatory|monitor|sonar`), "Named like a node census, which counts and watches reachable nodes", "Named like a node census", ""},
	// One address, 142 sessions in four days, each gone after its version
	// message. Elastum was an exchange; elastum.io no longer resolves.
	{regexp.MustCompile(`(?i)/Elastum:`), "Named like Elastum, a former exchange; talks to nodes through bitcoinj", "Named like Elastum", ""},
	// 26-27.10.2026: one address, five sessions in 40 minutes under protocol
	// 70001 with no services, saying our node is 127.0.0.1:8332. Two ended after
	// the version message; three took 7 to 16 MB of blocks each.
	// That is btc-rpc-proxy fetching a block the node lacks from a peer: it
	// sends "BTC RPC Proxy v<version>" (src/fetch_blocks.rs L50 in
	// Start9Labs/btc-rpc-proxy, 0.5 to 0.8.x, the fork StartOS runs; L49 in
	// Martin Habovštiak's Kixunil/btc-rpc-proxy up to 0.4.1; 15.11.2026).
	// Any other form was found nowhere and only names itself.
	// Source: Kixunil/btc-rpc-proxy src/fetch_blocks.rs L49 @dc95a40 (checked 2026-11-24)
	{regexp.MustCompile(`^BTC RPC Proxy v[0-9]`), "BTC RPC Proxy, which guards Bitcoin Core's RPC and fetches blocks the node lacks from peers", "BTC RPC Proxy", ""},
	{regexp.MustCompile(`^BTC RPC Proxy `), "Named like an RPC proxy for Bitcoin; no public source", "Named like an RPC proxy", ""},
	// Bisq's userAgent option defaults to "Bisq" (common/.../config/Config.java
	// in bisq-network/bisq), handed to bitcoinj in WalletsSetup.java.
	// Source: bisq-network/bisq common/src/main/java/bisq/common/config/Config.java L753 @bdb202b (checked 2026-11-24)
	{regexp.MustCompile(`(?i)/Bisq:`), "Bisq, a decentralised exchange; its wallet runs on bitcoinj", "Bisq exchange wallet", ""},
	// Source: forrestv/p2pool p2pool/bitcoin/p2p.py L35 @53c438b (checked 2026-11-24)
	{regexp.MustCompile(`^/P2Pool:`), "P2Pool, a peer-to-peer mining pool: miners share the work without a pool operator", "P2Pool mining pool", ""},
	// Source: BitcoinRocks-Core/BitcoinRocks CMakeLists.txt L29 @94ad5e5 (checked 2026-11-24)
	{regexp.MustCompile(`^/BitcoinRocks:`), "BitcoinRocks, an experimental fork of Core using RocksDB and compressed block storage", "BitcoinRocks, a Core fork", ""},
	// alexar76/symmartys (formerly emporion): symmartys/bitcoin_headers.py
	// sends "/SYMMARTYS:0.3/"; EMPORION is the older name, still in
	// emporion/bitcoin_headers.py at commit 922e4bc (17.10.2026).
	// Source: alexar76/emporion emporion/bitcoin_headers.py L46 @v0.4.0 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/EMPORION:`), "EMPORION, a Bitcoin header client for verifying OpenTimestamps proofs; later renamed SYMMARTYS", "EMPORION header client", ""},
	// const ua = Buffer.from('/wuji-headers:0.1/') in indexer/bitcoin-p2p.mjs
	// (MM-sheng/wuji, 15.11.2026).
	// Source: MM-sheng/wuji indexer/bitcoin-p2p.mjs L317 @7ebbb42 (checked 2026-11-24)
	{regexp.MustCompile(`^/wuji-headers:`), "WUJI's indexer: follows Bitcoin's block headers to build an index usable on Ethereum", "WUJI header client", ""},
	// const USER_AGENT in sdk/rust/pcoc-gossip/src/wire.rs in groundinglab/pcoc.
	// Source: groundinglab/pcoc sdk/rust/pcoc-gossip/src/wire.rs L17 @9c07a7f (checked 2026-11-24)
	{regexp.MustCompile(`^/pcoc-bari:`), "pcoc Bari, a timestamping service that follows Bitcoin's block headers", "pcoc Bari header client", ""},
	// UA_NAME("Prometheus") in src/clientversion.cpp, version 30.4.1 in
	// CMakeLists.txt (CryptoIceMLH/bitcoin-prometheus, 15.11.2026).
	// Source: CryptoIceMLH/bitcoin-prometheus src/clientversion.cpp L23 @d1b302b (checked 2026-11-24)
	{regexp.MustCompile(`^/Prometheus:`), "BTC-Prometheus, a full node forked from Bitcoin Core", "BTC-Prometheus node", ""},
	// Source: alexar76/symmartys symmartys/bitcoin_headers.py L46 @b70b1eb (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/SYMMARTYS:`), "SYMMARTYS, formerly EMPORION: follows Bitcoin headers to verify OpenTimestamps proofs", "SYMMARTYS header client", ""},
	// Source: nyonson/bitcoin-peers connection/src/connection/configuration.rs L15 @ce17e9f (checked 2026-11-24)
	{regexp.MustCompile(`^/bitcoin-peers:`), "bitcoin-peers, a Rust toolkit to connect to and survey Bitcoin peers", "bitcoin-peers crawler", ""},
	// Source: logos-co/logos-bitcoin-peer-list crawler/src/p2p.rs L21 @29e73f8 (checked 2026-11-24)
	{regexp.MustCompile(`^/logos-peer-list:`), "Logos peer list, a crawler that checks onion peers for the Logos wallet", "Logos peer list", ""},
	// Source: chjj/mako include/mako/net.h L116 @41ef104 (checked 2026-11-24)
	{regexp.MustCompile(`^/mako:`), "mako, a Bitcoin full node written in C", "mako node (C)", ""},
	// let ua = b"/bip110-crawler:0.1.0/" (src/p2p.rs L421 in
	// donaldevine/bip110-crawler at f412989, checked 23.11.2026).
	// Source: donaldevine/bip110-crawler src/p2p.rs L421 @f412989 (checked 2026-11-24)
	{regexp.MustCompile(`^/bip110-crawler:`), "bip110-crawler, a crawler that checks which peers follow BIP-110", "bip110-crawler", ""},
	// Castor, Floresta's Utreexo bridge, answers with "/bridge:0.1.3/" (src/node.rs
	// L1541 in getfloresta/castor at 6713d5b, checked 23.11.2026).
	// Source: getfloresta/castor src/node.rs L1541 @eba7b22 (checked 2026-11-24)
	{regexp.MustCompile(`^/bridge:`), "Castor, Floresta's Utreexo bridge: serves Utreexo proofs to other nodes", "Castor Utreexo bridge", ""},
	// Source: decentralizedb/bip110-observer app/crawler.py L134 @588e0a8 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/bip110-observer:`), "BIP-110 Observer, a crawler that collects peer user agents and advertised services", "BIP-110 Observer crawler", ""},
	// Only the name points to favillakey.io; the site documents no P2P
	// connections, so the kind stays Other.
	{regexp.MustCompile(`^/FavillaKEY:`), "Named like Favilla KEY, a Nostr and Bitcoin signing device (favillakey.io); what it connects for is not documented", "Named like Favilla KEY", ""},
	// Source: bitcoin-ghost/ghost ghost-core/src/clientversion.cpp L24 @8248dbf (checked 2026-11-24)
	{regexp.MustCompile(`^/Ghost:`), "Bitcoin Ghost, a fork of Core v30 that filters data-carrying transactions; its operators can join a decentralised mining pool", "Bitcoin Ghost (Core fork)", ""},
	// Source: hashhog/camlcoin lib/peer.ml L1019 @482c676 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/CamlCoin:`), "CamlCoin, a Bitcoin full node written in OCaml (hashhog)", "CamlCoin node (OCaml)", ""},
	// Source: hashhog/rustoshi crates/network/src/peer.rs L1815 @f9969a1 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/rustoshi:`), "rustoshi, a Bitcoin full node written in Rust (hashhog)", "rustoshi node (Rust)", ""},
	// Source: hashhog/lunarblock src/peer.lua L877 @84f5cfd (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/LunarBlock:`), "LunarBlock, a Bitcoin full node written in Lua (hashhog)", "LunarBlock node (Lua)", ""},
	// Source: hashhog/haskoin src/Haskoin/Network.hs L893 @e4f6b9e (checked 2026-11-24)
	{regexp.MustCompile(`^/Haskoin:`), "A Bitcoin full node written in Haskell (hashhog)", "Haskoin node (Haskell)", ""},
	// Source: hashhog/beamchain src/beamchain_peer.erl L1745 @18729c8 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/beamchain:`), "beamchain, a Bitcoin full node written in Erlang (hashhog)", "beamchain node (Erlang)", ""},
	// Source: hashhog/blockbrew cmd/blockbrew/main.go L1692 @a76b27f (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/blockbrew:`), "blockbrew, a Bitcoin full node written in Go (hashhog)", "blockbrew node (Go)", ""},
	// Source: hashhog/clearbit src/p2p.zig L46 @6510c80 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/clearbit:`), "clearbit, a Bitcoin full node written in Zig (hashhog)", "clearbit node (Zig)", ""},
	// Source: hashhog/nimrod src/network/messages.nim L23 @240a00d (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/nimrod:`), "nimrod, a Bitcoin full node written in Nim (hashhog)", "nimrod node (Nim)", ""},
	// Source: hashhog/hotbuns src/consensus/params.ts L549 @0cfeed4 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/hotbuns:`), "hotbuns, a Bitcoin full node written in TypeScript on Bun (hashhog)", "hotbuns node (TypeScript)", ""},
	// Source: hashhog/ouroboros src/ouroboros/p2p_messages.py L270 @1264ed2 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/(bitcoin-hybrid|ouroboros):`), "ouroboros, a Bitcoin full node written in Rust and Python (hashhog)", "ouroboros node (Rust)", ""},
	// crates/rbitcoin-primitives/src/lib.rs in rbitcoin-org/rbitcoin (moved
	// from reardencode).
	// Source: rbitcoin-org/rbitcoin crates/rbitcoin-primitives/src/lib.rs L82-89 @5d2ec82 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/rbitcoin:`), "rbitcoin, a Bitcoin full node written in Rust (reardencode)", "rbitcoin node (Rust)", ""},
	// Source: AmazingAng/rbtc crates/rbtc-net/src/message.rs L282 @4f30f67 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/rbtc:`), "rbtc, Bitcoin Core rewritten in Rust", "rbtc, Core in Rust", ""},
	// The original repository, bitcoinecho/bitcoin-echo, is gone (404,
	// 14.11.2026); a kept copy sets the name.
	// Source: jesterhodl/bitcoin-echo src/protocol/peer.c L31 @9196919 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/BitcoinEcho:`), "Bitcoin Echo, a full node in plain C, meant to be frozen as it is", "Bitcoin Echo node (C)", ""},
	// Source: BobClawblaw/bitcoinmachinecode asm/version.inc L33 @98d1276 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/BitcoinMachineCode:`), "Bitcoin Machine Code, a node written in x86 assembly, entirely AI-generated", "Bitcoin Machine Code node", ""},
	// Source: brunoerg/exnode lib/exnode/node.ex L306 @241d057 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/exnode:`), "exnode, a Bitcoin full node written in Elixir that checks scripts with Bitcoin Core's kernel library", "exnode (Elixir node)", ""},
	// Comb Downloader NG, the Haircomb wallet's block fetcher, says
	// "/Natasha:<its version>/" only with -natasha ("Useful for servers",
	// main.go L63); without it, "/Satoshi:29.0.0/" (util.go L8-9). Its source
	// is on Codeberg, read here from the Go module proxy's copy.
	// Source: codeberg.org/watashi564/combdownloader-ng main.go L81-82 @698cc05 (checked 2026-11-24)
	{regexp.MustCompile(`^/Natasha:`), "Comb Downloader NG in server mode: fetches Bitcoin blocks and headers for the Haircomb wallet and passes them on", "Comb Downloader (Natasha)", ""},
	// Source: kernel-node/kernel-node src/net/peer.rs L470 @ac5ad5c (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/kernel-node:`), "kernel-node, a node built on libbitcoinkernel, Core's validation engine as a library", "kernel-node (Core kernel)", ""},
	// Source: yuvicc/bitcoinkernel-node src/node/p2p.cpp L104 @0d3d474 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/bitcoinkernel-node:`), "bitcoinkernel-node, a small full node built on libbitcoinkernel, Core's validation engine as a library", "bitcoinkernel-node", ""},
	// satd's crawl sessions say "/satd:<version>(crawler)/" (user_agent in
	// node/src/net/crawl.rs in epochbtc/satd at c3be00a, checked 23.11.2026).
	// Source: epochbtc/satd node/src/net/crawl.rs L226 @e0e8c28 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/satd:[^/(]*\(crawler`), "satd in crawler mode: maps reachable nodes", "satd crawler", ""},
	// Source: epochbtc/satd node/src/lib.rs L38 @e0e8c28 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/satd:`), "satd, a Bitcoin full node written in Rust that works like Core", "satd node (Rust)", ""},
	// Source: hemilabs/heminetwork cmd/tbcd/tbcd.go L283 @c76e7e7 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/tbcd:`), "tbcd, the Bitcoin daemon of the Hemi network", "tbcd, Hemi's daemon", ""},
	// "soak" is nowhere in hemilabs/heminetwork. Hemi's developers describe a
	// "/btcwire:0.5.0/hemi-soak:1.0/ fleet peer" that served a partner's node a
	// fork of ~2,100 headers with no work (PR #1174 in hemilabs/heminetwork,
	// merged 16.09.2026; bnoc.xyz/t/170), but do not say who runs it.
	{regexp.MustCompile(`(?i)hemi-soak`), "Named like a Hemi soak test; Hemi's developers saw one serve their node a fork without work", "Named like hemi-soak", ""},
	// paulscode's release of 28.10.2026, Knots carried on after it hard-forked
	// to BLAKE2b: src/clientversion.cpp at tag v31.1.knots-sha256.20261018 in
	// paulscode/knots-sha256 (not on master).
	// Oracle Knots (MarcanoFilms/oracle-knots, src/clientversion.cpp at
	// 5486108): "OracleKnots:" after its own name, on the BLAKE2b chain whose
	// first block is 961640 (src/kernel/chainparams.cpp). With -uaprivacy it
	// sends Knots' plain string; then only service bit 28 tells (forks.go).
	// Source: MarcanoFilms/oracle-knots src/clientversion.cpp L74 @ae2f88d (checked 2026-11-24)
	{regexp.MustCompile(`OracleKnots:`), "Oracle Knots, a Knots fork on the BLAKE2b chain (hard fork at block 961,640)", "Oracle Knots (BLAKE2b)", ""},
	// The standalone BIP-110 client (dathonohm/bitcoin, src/clientversion.cpp at
	// 78dfa44): ua += "UASF-BIP110:0.4/". Its rules split off at 961632.
	// Source: dathonohm/bitcoin src/clientversion.cpp L77 @78dfa44 (checked 2026-11-24)
	{regexp.MustCompile(`UASF-BIP110:`), "Knots with the BIP-110 UASF, on the chain that split off at block 961,632", "BIP-110 UASF chain", ""},
	// Two forks of 2017 on Core's name: the BIP148 UASF client, "UASF-Segwit:1.0
	// (BIP148)/" (src/clientversion.cpp in UASF/bitcoin, branch 0.14-BIP148),
	// and btc1, SegWit2x, with the comment "2x" (src/init.cpp in btc1/bitcoin).
	// Source: UASF/bitcoin src/clientversion.cpp L106 @v0.14.2-uasfsegwit1.0 (checked 2026-11-24)
	{regexp.MustCompile(`UASF-Segwit:`), "The BIP148 UASF client of 2017, which enforced SegWit", "BIP148 UASF (2017)", ""},
	// Source: btc1/bitcoin src/init.cpp L1239 @v1.14.5 (checked 2026-11-24)
	{regexp.MustCompile(`\(2x\)`), "btc1, the SegWit2x client of 2017; its fork was called off", "btc1 SegWit2x (2017)", ""},
	// Source: paulscode/knots-sha256 src/clientversion.cpp L74 @e9c534b (checked 2026-11-24)
	{regexp.MustCompile(`/KnotsSHA256:`), "Knots (SHA256), paulscode's build of Knots that stays on the SHA256 chain", "Knots (SHA256), paulscode", ""},
	// spruned's copy of pycoinnet: subversion=b"/pycoinnet/" in
	// spruned/dependencies/pycoinnet/version.py L81, used unchanged by
	// spruned/daemon/bitcoin_p2p/p2p_connection.py L114 (mempoolco/spruned,
	// checked 23.11.2026). pycoinnet itself says "/Notoshi/" (below).
	// Source: mempoolco/spruned spruned/dependencies/pycoinnet/version.py L81 @630aad3 (checked 2026-11-24)
	{regexp.MustCompile(`^/pycoinnet/`), "spruned, a Bitcoin client without a blockchain of its own: it fetches any block from peers when asked", "spruned (no own chain)", ""},
	// APP_NAME "blueberry", APP_VERSION "2026.08.17" (peers/.../net/UserAgent.kt
	// in BlueWallet/Blueberry.kmp), sent as "/${name}:${version}/"
	// (session/VersionHandshake.kt in GladosBlueWallet/bip324.kmp).
	// Source: BlueWallet/Blueberry.kmp peers/src/commonMain/kotlin/io/bluewallet/blueberry/peers/net/UserAgent.kt L3 @c8071fc (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/blueberry:`), "Blueberry, BlueWallet's new wallet that talks to nodes directly", "Blueberry (BlueWallet)", ""},
	// Labels a node's setup puts in brackets after Core's name. Apollo's own
	// bitcoin.conf sets it (jstefanop/apolloapi-v2, backend/default-configs).
	// Source: jstefanop/apolloapi-v2 backend/default-configs/bitcoin.conf L5 @c33a0be (checked 2026-11-24)
	{regexp.MustCompile(`\(FutureBit-Apollo-Node\)`), "A node on FutureBit Apollo, a home Bitcoin miner with a built-in full node", "FutureBit Apollo node", ""},
	// docker/my-dojo/bitcoin/restart.sh: -uacomment="Samourai Dojo $DOJO_VERSION_TAG"
	// (the original at code.samourai.io is offline; Umbrel's copy,
	// getumbrel/umbrel-samourai-dojo, has it).
	// Source: getumbrel/umbrel-samourai-dojo docker/my-dojo/bitcoin/restart.sh L16 @f4a417e (checked 2026-11-24)
	{regexp.MustCompile(`\(Samourai Dojo `), "A node run by Samourai Dojo, the self-hosted backend for Samourai Wallet and other light wallets", "Samourai Dojo node", ""},
	// docker/my-dojo/bitcoin/restart.sh in git.ronindojo.io/Ronin/dojo:
	// -uacomment="RoninDojo $RONINDOJO_VERSION_TAG" (v2.4.1; older tags use
	// $DOJO_VERSION_TAG, and an empty tag leaves "(RoninDojo )").
	// Source: git.ronindojo.io/Ronin/dojo docker/my-dojo/bitcoin/restart.sh L19 @69136f3 (checked 2026-11-24)
	{regexp.MustCompile(`\(RoninDojo `), "A node run by RoninDojo, an installer and UI for Samourai Wallet's Dojo", "RoninDojo node", ""},
	// The bitcoin.conf in MiniBolt's Core and Knots pages: uacomment=MiniBolt node.
	// Source: minibolt-guide/minibolt bitcoin/bitcoin/bitcoin-client.md L311 @fd980f0 (checked 2026-11-24)
	// Source: minibolt-guide/minibolt bitcoin/bitcoin/btc-rpc-explorer.md L412 @53f972e (checked 2026-11-24)
	{regexp.MustCompile(`(?i)\((Official )?MiniBolt[ _]node\)|; MiniBolt node\)`), "A node set up with MiniBolt, a step-by-step guide to building a Bitcoin node", "MiniBolt guide node", ""},
	// A 2017 Docker image of Bitcore set uacomment=bitcore (m00re/bitcore-docker);
	// these run newer Core, so the text goes by the name only.
	{regexp.MustCompile(`\(bitcore\)`), "Named like BitPay's Bitcore, an index server that runs on a node", "Named like Bitcore", ""},
	// userAgent = "/winnow:0.1/" (Sources/WalletCore/Network/Transport/
	// PeerConnection.swift in winnowwallet/winnow).
	// Source: winnowwallet/winnow Sources/WalletCore/Network/Transport/PeerConnection.swift L82 @162abb7 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/winnow:`), "Winnow, an iPhone wallet that talks to nodes directly", "Winnow iPhone wallet", ""},
	// Source: horizontalsystems/bitcoin-kit-android bitcoincore/src/main/kotlin/io/horizontalsystems/bitcoincore/network/messages/VersionMessage.kt L18 @1f581d1 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/BitcoinKit:`), "BitcoinKit, a wallet library (Unstoppable Wallet on Android and others)", "BitcoinKit wallet library", ""},
	// Source: horizontalsystems/BitcoinCore.Swift Sources/BitcoinCore/Classes/Network/Peer/Peer.swift L71 @5b49f42 (checked 2026-11-24)
	{regexp.MustCompile(`^/WalletKit:`), "Unstoppable Wallet on iPhone", "Unstoppable Wallet iOS", ""},
	// CLIENT_NAME("Ladybug") (src/clientversion.cpp in rkarpuzov/Ladybug-pupa):
	// the node of a coin of its own on Bitcoin Core, with Bitcoin's network
	// magic. Its Android wallet says "Ladybug Wallet" on bitcoinj instead.
	// Source: rkarpuzov/Ladybug-pupa src/clientversion.cpp L16 @8b8be89 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/Ladybug:`), "Ladybug, an altcoin built on Bitcoin Core", "Ladybug altcoin", ""},
	// HODL Wallet puts its library's name first: "/Liviano:<version>/hodlwallet:2.0/"
	// (HodlWallet/Core/Services/WalletService.cs in hodlwallet/hodlwallet,
	// Liviano/Version.cs in hodlwallet/liviano).
	// Source: hodlwallet/hodlwallet HodlWallet/Core/Services/WalletService.cs L64 @f1f7284 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)/hodlwallet:`), "HODL Wallet, a phone wallet", "HODL Wallet", ""},
	// "Yuki / {YUKI_VERSION}" (src/network/outbound_messages.rs in imperviousinc/yuki).
	// Source: imperviousinc/yuki src/network/outbound_messages.rs L49 @d077223 (checked 2026-11-24)
	{regexp.MustCompile(`^Yuki /`), "Yuki, a light client by Impervious built on Kyoto", "Yuki light client", ""},
	// Source: VerifiedXBlockchain/VerifiedX-Core VerifiedXCore/Bitcoin/Attestation/Net/BtcSources.cs L100 @e814c14 (checked 2026-11-24)
	{regexp.MustCompile(`^/VerifiedX:`), "A VerifiedX validator: a sidechain that holds bitcoin and follows Bitcoin's chain for it", "VerifiedX validator", ""},
	// format!("/LionSeed:{}/", ...) (src/crawl.rs in LionThunderFingers/LionSeed),
	// "serves only verified fork nodes" by its description.
	// Source: LionThunderFingers/LionSeed src/crawl.rs L38 @9d75a7b (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/LionSeed:`), "LionSeed, a DNS seeder that lists only the nodes of one fork", "LionSeed DNS seeder", ""},
	// gnoban connects to peers itself, but under python-bitcoinlib's name: its
	// BitcoinMsgver() keeps the library's subversion (gnoban.py L640-669 in
	// caesrcd/gnoban, 23.11.2026), so a peer saying "gnoban" only borrows it.
	{regexp.MustCompile(`(?i)^/gnoban:`), "Named like gnoban, a tool that scores nodes and bans the ones its operator does not want", "Named like gnoban", ""},
	// "libbtc 0.1", no slashes (src/net.c in libbtc/libbtc).
	// Source: libbtc/libbtc src/net.c L293 @bd386ee (checked 2026-11-24)
	{regexp.MustCompile(`^libbtc `), "libbtc, a small C library for Bitcoin wallets and tools", "libbtc C library", ""},
	// "/haskoin-btc:<version>/" (src/Haskoin/Network/Constants.hs in
	// jprupp/haskoin-core, formerly haskoin/haskoin-core); haskoin-node, which Haskoin Store runs on, sends it.
	// Source: jprupp/haskoin-core src/Haskoin/Network/Constants.hs L91 @240283f (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/haskoin-btc:`), "Haskoin, a Haskell library for Bitcoin wallets and tools", "Haskoin library (Haskell)", ""},
	// Light clients and wallets that speak to nodes themselves, each string
	// as its source sets it (13.11.2026). Active: bitfyn (internal/p2p/p2p.go
	// in pin2t/bitfyn, on btcd's wire), Twotbit (PeerConnection.swift in
	// pin2t/twotbit-macos, peer.h in pin2t/twotbit-qt), bitc
	// (core/bitc-defs.h in mx4/bitc, "/bitc:0.1.0" without the closing
	// slash). Older: BDK 0.x's compact filter backend
	// (src/blockchain/compact_filters/peer.rs at v0.30.2), mSIGNA
	// (src/networksync.cpp in ciphrex/mSIGNA), BitcoinSPV
	// (Protocol/WSMessageVersion.m and Global/WSConfig.m in
	// keeshux/BitcoinSPV), pycoinnet's "/Notoshi/"
	// (pycoinnet/helpers/standards.py in richardkiss/pycoinnet), pyspv
	// (pyspv/__init__.py and network.py in sarchar/pyspv), chainhorn
	// (chainhorn/contrib/core in brain-zhang/chainhorn), Murmel
	// (src/constructor.rs in rust-bitcoin/murmel), BitcoinSwift
	// (PeerController.swift in DoubleSha/BitcoinSwift), Koyn's header tool
	// (Main.java in elkrem/koyn-sync, on bitcoinj). EngineCore
	// (crates/networks/bitcoin/src/peer.rs in Cattens-Roses/cryptogate), a
	// terminal wallet by its README; bitcoin-net, Webcoin's P2P library, which
	// says "/node.js:<version>/bitcoin-net:5.2.0/", or the browser's own name
	// first (src/peer.js in mappum/bitcoin-net); BitcoinWallet by ScripterRon
	// (Parameters.java at v1.3 in shadders/BitcoinWallet) and Toporin's fork,
	// "/BitcoinWallet:3.0.1/BitcoinCore:2.1.1/"; rust-bitcoin-spv's command
	// line client (src/bin/main.rs in TheBlueMatt/rust-bitcoin-spv).
	// Source: pin2t/bitfyn internal/p2p/p2p.go L86 @31754c7 (checked 2026-11-24)
	{regexp.MustCompile(`^/btcwire:[^/]*/bitfyn:`), "bitfyn, a desktop wallet that follows the chain by compact block filters", "bitfyn wallet", ""},
	// Source: pin2t/twotbit-macos Core/Sources/TwotbitKit/P2P/PeerConnection.swift L32 @19e2e00 (checked 2026-11-24)
	{regexp.MustCompile(`^/twotbit:`), "Twotbit, a desktop wallet that follows the chain by compact block filters", "Twotbit wallet", ""},
	// Source: mx4/bitc core/bitc-defs.h L9 @b8f9d4f (checked 2026-11-24)
	{regexp.MustCompile(`^/bitc:`), "bitc, a small Bitcoin wallet written in C", "bitc wallet (C)", ""},
	// Source: bitcoindevkit/bdk src/blockchain/compact_filters/peer.rs L114 @77c95b9 (checked 2026-11-24)
	{regexp.MustCompile(`^MagicalBitcoinWallet$`), "BDK, the Bitcoin Dev Kit wallet library, in its old compact filter backend", "BDK (old filter backend)", ""},
	// Source: ciphrex/mSIGNA src/networksync.cpp L358 @b79c55e (checked 2026-11-24)
	{regexp.MustCompile(`^Wallet v0\.1$`), "mSIGNA, a multisig desktop wallet", "mSIGNA wallet", ""},
	// Source: keeshux/bitcoinspv BitcoinSPV/Sources/Global/WSConfig.m L30 @53ec533 (checked 2026-11-24)
	{regexp.MustCompile(`^/BitcoinSPV:`), "BitcoinSPV, a light wallet library for iPhone and Mac", "BitcoinSPV library", ""},
	// Source: richardkiss/pycoinnet pycoinnet/helpers/standards.py L64 @7c354cb (checked 2026-11-24)
	{regexp.MustCompile(`^/Notoshi/$`), "pycoinnet, Richard Kiss's Python library for the Bitcoin protocol", "pycoinnet library", ""},
	// Source: sarchar/pyspv pyspv/__init__.py L29-32 @33395ae (checked 2026-11-24)
	{regexp.MustCompile(`^/pyspv:`), "pyspv, a Python library for light wallets", "pyspv library", ""},
	// Source: brain-zhang/chainhorn chainhorn/contrib/core/__init__.py L23 @4db18fe (checked 2026-11-24)
	{regexp.MustCompile(`^/chainhorn:`), "chainhorn, a Python light client", "chainhorn light client", ""},
	// Source: rust-bitcoin/murmel src/constructor.rs L61 @f8902dd (checked 2026-11-24)
	{regexp.MustCompile(`^/Murmel:`), "Murmel, a Rust light client", "Murmel light client", ""},
	// Source: Cattens-Roses/cryptogate crates/networks/bitcoin/src/peer.rs L203 @b07eefd (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/enginecore:`), "EngineCore, a terminal wallet for Bitcoin, Litecoin and Ethereum that follows Bitcoin by compact block filters", "EngineCore wallet", ""},
	// Source: mappum/bitcoin-net src/peer.js L66 @622a4eb (checked 2026-11-24)
	{regexp.MustCompile(`/bitcoin-net:`), "bitcoin-net, the P2P library of Webcoin, a JavaScript light client", "bitcoin-net (Webcoin)", ""},
	// Source: shadders/BitcoinWallet src/BitcoinWallet/Parameters.java L47 @99d423c (checked 2026-11-24)
	{regexp.MustCompile(`^/BitcoinWallet:`), "BitcoinWallet, a Java desktop wallet by ScripterRon", "BitcoinWallet (Java)", ""},
	// Source: TheBlueMatt/rust-bitcoin-spv src/bin/main.rs L78-81 @a5bc31f (checked 2026-11-24)
	{regexp.MustCompile(`^/rust-spv:`), "rust-bitcoin-spv, a Rust light client", "rust-bitcoin-spv", ""},
	// Source: DoubleSha/BitcoinSwift BitcoinSwift/PeerController.swift L70 @8756a45 (checked 2026-11-24)
	{regexp.MustCompile(`^/BitcoinSwift:`), "BitcoinSwift, a Swift library for the Bitcoin protocol", "BitcoinSwift library", ""},
	// josibake/sashimi exists ("a Bitcoin node as Unix tools", Zig), but no
	// "/sashimi:" is in its code (14.11.2026).
	{regexp.MustCompile(`(?i)^/sashimi:`), "Named like sashimi, small Unix tools for Bitcoin by josibake", "Named like sashimi", ""},
	// src/lib.rs in gnostr-org/gnostr-bitcoin: user_agent = "/rust-sha2-p2p-client/".
	// Source: gnostr-org/gnostr-bitcoin src/lib.rs L434 @71effa0 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/rust-sha2-p2p-client/`), "gnostr-bitcoin, a terminal monitor of the Bitcoin network from the gnostr project", "gnostr-bitcoin monitor", ""},
	{regexp.MustCompile(`(?i)^/Gnostr:`), "Named like gnostr, the git+nostr tools", "Named like gnostr", ""},
	// CLIENT_NAME("Aurum") (src/clientversion.cpp in aurumcoin-au/Aurumcoin-AU),
	// with Bitcoin's network magic.
	// Source: aurumcoin-au/Aurumcoin-AU src/clientversion.cpp L22 @96bad6d (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/Aurum:`), "Aurumcoin, an altcoin built on Bitcoin Core 0.12", "Aurumcoin altcoin", ""},
	// Source: saltduck/bitcoinpurity src/clientversion.cpp L79 @124fbc8 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)/Purity:`), "Bitcoin Purity, a fork of Knots that made BIP110 permanent in a hard fork at block 961637", "Bitcoin Purity hard fork", ""},
	// USER_AGENT = "Bitcoin Wallet" (wallet/src/de/schildbach/wallet/Constants.java
	// in bitcoin-wallet/bitcoin-wallet), on bitcoinj.
	// Source: bitcoin-wallet/bitcoin-wallet wallet/src/de/schildbach/wallet/Constants.java L127 @e1c558c (checked 2026-11-24)
	{regexp.MustCompile(`Bitcoin Wallet:`), "Bitcoin Wallet by Schildbach, an Android app", "Bitcoin Wallet (Android)", ""},
	// breadwallet-core sends "/bread:" with its own version, 2.1 since 2019
	// (BRPeer.h), also in BRD's later WalletKit (blockset-corp/walletkit).
	// Source: breadwallet/breadwallet-legacy BreadWallet/BRPeer.h L39 @951bb47 (checked 2026-11-24)
	{regexp.MustCompile(`^/bread:`), "BRD's wallet core: the BRD app, shut down in 2022, or a wallet built on its code", "BRD wallet core", ""},
	// "/breadwallet:%@/" (BreadWallet/BRPeer.h in breadwallet/breadwallet).
	// Source: breadwallet/breadwallet-legacy BreadWallet/BRPeer.h L39 @951bb47 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)breadwallet`), "BRD (breadwallet), a phone wallet shut down in 2022", "BRD wallet, closed 2022", ""},
	// peerGroup.setUserAgent("MultiBit", ...) (MultiBitService.java in
	// Multibit-Legacy/multibit).
	// Source: Multibit-Legacy/multibit src/main/java/org/multibit/network/MultiBitService.java L310 @21e2e9d (checked 2026-11-24)
	{regexp.MustCompile(`(?i)multibit`), "MultiBit, a desktop wallet discontinued in 2017", "MultiBit, ended 2017", ""},
	// "/Bither" + version + "/", no colon (message/VersionMessage.java in
	// Bither/bitherj).
	// Source: bither/bitherj bitherj/src/main/java/net/bither/bitherj/message/VersionMessage.java L124 @1dff0be (checked 2026-11-24)
	{regexp.MustCompile(`(?i)bither`), "Bither, a wallet for phone and desktop", "Bither wallet", ""},
	// Wasabi itself picks a Core name at random, "/Satoshi:30.2.0/" and others
	// (UserAgents in WalletWasabi/Helpers/Constants.cs): a peer saying "wasabi"
	// only borrows the name.
	{regexp.MustCompile(`(?i)wasabi`), "Named like Wasabi Wallet; Wasabi itself introduces itself as Bitcoin Core", "Named like Wasabi", ""},
	// On bitcoinj, so before its rule: HTTPSeed (crawler.kt in bitcoinj/httpseed)
	// and CoinView (Main.java in vs-uulm/CoinView), see the sources further down.
	// Source: bitcoinj/httpseed src/main/kotlin/crawler.kt L158 @40d2e88 (checked 2026-11-24)
	{regexp.MustCompile(`/HTTPSeed:`), "Cartographer (HTTPSeed), bitcoinj's crawler behind an HTTP seed, archived", "HTTPSeed crawler", ""},
	// Source: vs-uulm/CoinView collection/application/research/Main.java L24 @d3a9f95 (checked 2026-11-24)
	{regexp.MustCompile(`/UU-Network-Research:`), "CoinView, Ulm University's collector of transaction announcements", "CoinView (Ulm University)", ""},
	// The library alone, without a wallet app's name after it. After the
	// wallet rules above, which also start with bitcoinj.
	// Source: bitcoinj/bitcoinj core/src/main/java/org/bitcoinj/core/VersionMessage.java L53 @1c84f21 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/bitcoinj:`), "bitcoinj, a Java library that wallets and other tools use to talk to nodes", "bitcoinj Java library", ""},
	// neutrino sends a version fixed in its own code, not lnd's: 0.12.0-beta
	// from neutrino v0.12.0 to v0.16.2 (lnd 0.13 to 0.20), then 0.17.0, and
	// 0.17.1 from v0.17.1 on, v0.18.0 too (lnd 0.21, June 2026;
	// UserAgentVersion in neutrino.go in lightninglabs/neutrino).
	// Zeus 13.2 ships lnd 0.21.4, Blixt 0.9 lnd 0.20, and neither changes the
	// name; lnd on a server in neutrino mode says the same, so an app is only an example.
	// Source: lightningnetwork/lnd config.go L665 @v0.21.0-beta (checked 2026-11-24)
	{regexp.MustCompile(`(?i)/neutrino:0\.17\.`), "lnd 0.21 or newer, without a node of its own (neutrino), e.g. Zeus", "lnd 0.21+ (neutrino)", "lnd 0.21+"},
	// Source: lightninglabs/neutrino neutrino.go L49 @v0.12.0 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)/neutrino:0\.12\.0-beta/`), "Older lnd, without a node of its own (neutrino), e.g. Blixt or an older Zeus", "Older lnd (neutrino)", "older lnd"},
	// Source: lightninglabs/neutrino neutrino.go L52 @55dc1ec (checked 2026-11-24)
	{regexp.MustCompile(`(?i)/neutrino:`), "Neutrino: Lightning wallets on lnd, such as Blixt or Zeus", "lnd wallet (neutrino)", ""},
	// neutrino's own start and version under a name of its own
	// ("/btcwire:0.5.0/btcpost-neutrino:0.17.1/", 14.11.2026): its code,
	// renamed, by someone who has not published it.
	{regexp.MustCompile(`(?i)^/btcwire:[^/]*/[^/]+-neutrino:`), "Probably neutrino, lnd's light client, under another name: its btcwire start and its version", "Probably renamed neutrino", ""},
	// neutrino under a name its program sets, on btcd's wire library
	// (/btcwire:<v>/ in front): Bisoncraft's wallets set UserAgentName in their
	// copy of neutrino, Zap hands lnd --neutrino.useragentname.
	// Source: bisoncraft/caribou wallet/utxowallet/spv/neutrino.go L49 @fccbc64 (checked 2026-11-24)
	{regexp.MustCompile(`^/btcwire:[^/]*/CaribouSPV:`), "Caribou, a wallet by Bisoncraft that follows Bitcoin with its own build of neutrino", "Caribou wallet", ""},
	// Source: bisoncraft/utxowallet spv/neutrino.go L49 @422fc40 (checked 2026-11-24)
	{regexp.MustCompile(`^/btcwire:[^/]*/BisonSPV:`), "Bisoncraft's utxowallet, a wallet module on its own build of neutrino", "Bisoncraft utxowallet", ""},
	// Source: LN-Zap/zap-desktop services/neutrino/neutrino.js L371-372 @d89c6c0 (checked 2026-11-24)
	{regexp.MustCompile(`^/btcwire:[^/]*/zap-desktop:`), "Zap, a Lightning wallet for the desktop, on lnd without a node of its own (neutrino)", "Zap desktop (neutrino)", ""},
	// Source: LN-Zap/zap-iOS SwiftLnd/Lnd/LocalLndConfiguration.swift L77 @a1af8bf (checked 2026-11-24)
	{regexp.MustCompile(`^/btcwire:[^/]*/zap-ios:`), "Zap for iPhone, a Lightning wallet on lnd without a node of its own (neutrino)", "Zap iPhone (neutrino)", ""},
	// Only the word in the name ("/neutrino-checker:0.1/", 14.11.2026): no
	// source, so no wallet.
	{regexp.MustCompile(`(?i)neutrino`), "Named like Neutrino, lnd's light client", "Named like Neutrino", ""},
	// "/Rust BIP-157:<version>/rust-bitcoin:<version>/" (src/network/outbound.rs
	// in 2140-dev/kyoto).
	// Source: 2140-dev/kyoto src/network/outbound.rs L106 @4fa8cc7 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)BIP-157|kyoto`), "Kyoto, a light client for wallets built with BDK", "Kyoto light client (BDK)", ""},
	// n.agent = "/Bitflash:btc-light/" (src/btclight.cpp L424 in
	// bitflash-network/bitflash, 15.11.2026).
	// Source: bitflash-network/bitflash src/btclight.cpp L424 @134dc25 (checked 2026-11-24)
	{regexp.MustCompile(`^/Bitflash:btc-light/`), "Bitflash's wallet, which follows the chain by headers and compact filters (docs.bitflash.network)", "Bitflash wallet", ""},
	// "/Floresta:<version>/" (bin/florestad/build.rs in getfloresta/Floresta).
	// Source: getfloresta/Floresta bin/florestad/build.rs L22 @64238a2 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)floresta`), "Floresta, a lightweight Utreexo node", "Floresta Utreexo node", ""},
	// format!("/electrs:{}/", ELECTRS_VERSION) (src/p2p.rs in romanz/electrs,
	// up to v0.10.10; later versions have no P2P code).
	// Source: romanz/electrs src/p2p.rs L339 @v0.10.8 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)electrs`), "electrs, the address index behind Electrum-style wallets", "electrs address index", ""},
	// UA_NAME("CKCoinD") (src/clientversion.cpp in ckolivas/ckcoind, branch ck29).
	// Source: ckolivas/ckcoind src/clientversion.cpp L23 @4a0a9a2 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/CKCoinD:`), "ckcoind, a mining fork of Bitcoin Core used with ckpool", "ckcoind (ckpool fork)", ""},
	// "/rust-bitcoin:0.18/bluematt-tokio-client:0.1/" (src/peer.rs in
	// dongcarl/dnsseed-rust; TheBlueMatt/dnsseed-rust is gone).
	// Source: dongcarl/dnsseed-rust src/peer.rs L124 @2cf23b8 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)bluematt-tokio-client`), "dnsseed-rust, the crawler behind a DNS seeder", "dnsseed-rust crawler", ""},
	// Source: ckolivas/ckpool src/ckp2p.c L537 @e1da45f (checked 2026-11-24)
	{regexp.MustCompile(`(?i)ckp2p`), "ckp2p, ckpool's block relay: passes on blocks only, never transactions", "ckp2p block relay", ""},
	// AppInfo.cs and P2p/P2pFastPeer.cs in bugoao/miningcore-btc-solo.
	// Source: bugoao/miningcore-btc-solo src/MiningcoreBtcSolo/Util/AppInfo.cs L5-9 @8749652 (checked 2026-11-24)
	{regexp.MustCompile(`^/miningcore-btc-solo:`), "miningcore-btc-solo, a solo mining pool built on Miningcore", "miningcore-btc-solo pool", ""},
	// internal/bitcoin/p2p_client.go in Freakoverse/dnn-node: UserAgentName
	// "DNN-Node" on btcd's peer package, which puts /btcwire:<v>/ before it.
	// Source: Freakoverse/dnn-node internal/bitcoin/p2p_client.go L185 @c8c4775 (checked 2026-11-24)
	{regexp.MustCompile(`^/btcwire:[^/]*/DNN-Node:`), "DNN Node, a Nostr relay and naming service anchored to Bitcoin", "DNN Node", ""},
	// Relay networks with a name of their own: bloXroute's gateway builds
	// "bloXroute Gateway <version>" (bxgateway, 2022); Matt Corallo's
	// RelayNode adapters send /RelayNetworkClient:42/, /RelayNetworkServer:42/
	// and /RelayNetworkTerminator:42/0000000/ (c++/client.cpp and
	// bitcoindterminator.cpp in TheBlueMatt/RelayNode, 2013-2017); Falcon was
	// seen as /Cornell-Falcon-Network:0.1.0/ in a bips discussion (March 2017),
	// no code, so only the name.
	// Source: bloXroute-Labs/bxgateway src/bxgateway/gateway_constants.py L64 @4d09fdc (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^bloXroute Gateway`), "bloXroute Gateway, a relay network's gateway: passes on blocks and transactions", "bloXroute gateway", ""},
	// Source: TheBlueMatt/RelayNode c++/client.cpp L23 @c50d812 (checked 2026-11-24)
	// Source: TheBlueMatt/RelayNode c++/server.cpp L16 @c50d812 (checked 2026-11-24)
	// Source: TheBlueMatt/RelayNode c++/bitcoindterminator.cpp L17 @c50d812 (checked 2026-11-24)
	{regexp.MustCompile(`^/RelayNetwork(Client|Server|Terminator):`), "Matt Corallo's old Relay Network, which ran until 2017", "Old Relay Network", ""},
	{regexp.MustCompile(`^/Cornell-Falcon-Network:`), "Named like Falcon, Cornell's block relay network from 2016", "Named like Falcon", ""},
	// Before NBitcoin, whose name is inside "OpenBitcoin".
	// USER_AGENT = '/OpenBitcoin-crawler:0.1/' (crawl.mjs in openbitcoincom/node-crawler).
	// Source: openbitcoincom/node-crawler crawl.mjs L32 @384b00c (checked 2026-11-24)
	{regexp.MustCompile(`^/OpenBitcoin-crawler:`), "OpenBitcoin's crawler, a survey of reachable nodes for openbitcoin.com", "OpenBitcoin crawler", ""},
	// Source: MetacoSA/NBitcoin NBitcoin/Protocol/Payloads/VersionPayload.cs L69 @2d0fe93 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)NBitcoin`), "NBitcoin, a .NET Bitcoin library (behind BTCPay Server, among others)", "NBitcoin .NET library", ""},
	// "/bitcore:<version>/" is bitcore-p2p's default, the version its own
	// (lib/messages/commands/version.js in bitpay/bitcore-p2p), so any program on it says so: BitPay's node and
	// index server, or bitcore-spv's light client (throughnothing/bitcore-spv).
	// Source: bitpay/bitcore-p2p lib/messages/commands/version.js L40 @e99803e (checked 2026-11-24)
	{regexp.MustCompile(`(?i)/bitcore:`), "Bitcore, BitPay's Bitcoin code: its node and index server, or another program on its P2P library", "Bitcore (BitPay)", ""},
	// Core's private broadcast (v31): it opens a connection over Tor or I2P,
	// hands over one of its own transactions and leaves, introducing itself
	// with this fixed name and no services so nobody can tell which node sent
	// it (net_processing.cpp, IsPrivateBroadcastConn; PR #29415). The name is
	// bitcoin-submittx's (laanwj, 2015), a tool that hands over a transaction
	// and leaves, which took it from Jeff Garzik's pynode (2012): both send it
	// too (MY_SUBVERSION in each). Three sources, one string (15.11.2026).
	// The one seen on the node this was written for came from a Tor exit, v2,
	// offering nothing.
	// Source: jgarzik/pynode node.py L34 @1898048 (checked 2026-11-24)
	{regexp.MustCompile(`^/pynode:0\.0\.1/$`), "Bitcoin Core handing over a transaction privately, then leaving; the name is that of bitcoin-submittx and pynode, which send it too", "Core broadcast or pynode", ""},
	// m4ycon on bnoc (#198): it sweeps the P2A anchors of a fleet that minted
	// rune 1:0 in chains of transactions; on the node this was written for it
	// pushed unannounced 66-byte transactions, the size of such a sweep. The
	// name is in m4ycon's measurement setup, "user_agent": "/p2aburn:1.0/"
	// (data/experiment/final/params.py in
	// m4ycon/cmpct-block-reconstruction-degradation).
	// Source: m4ycon/cmpct-block-reconstruction-degradation data/experiment/final/params.py L98 @1a0ec59 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/p2aburn:`), "p2aburn, a bot that sweeps P2A anchors (outputs anyone may spend) behind a Runes minting fleet", "p2aburn anchor sweeper", ""},
	// strSubVer = b'/python-bitcoinlib:' + version (bitcoin/messages.py L123 in
	// petertodd/python-bitcoinlib, 23.11.2026).
	// Source: petertodd/python-bitcoinlib bitcoin/messages.py L123 @fbbe924 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)python-bitcoinlib`), "python-bitcoinlib, a Python library scripts use to talk to nodes", "python-bitcoinlib", ""},
	// PyBLOCK's install scripts set uaappend=PyBLOCK and uacomment=PyBLOCK Crew
	// (install-full-node.sh in curly60e/pyblock at 3d8be3a, deleted in June
	// 2026; still in PyBLOCK-Bitcoin/pyblock). "(PyBLOCK-POOL)" is in no
	// source: a label its operator chose, and its kind is Pool node.
	// Only those two are PyBLOCK's own (23.11.2026): "(PyBLOCK Crew)" and
	// PyBLOCK appended at the end. Any other bracket, "(PyBLOCK-BIP110)" too,
	// an operator may write by hand.
	// Source: PyBLOCK-Bitcoin/pyblock install-full-node.sh L458 @010f211 (checked 2026-11-24)
	// Source: PyBLOCK-Bitcoin/pyblock install-full-node.sh L455 @010f211 (checked 2026-11-24)
	{regexp.MustCompile(`\(PyBLOCK Crew\)|PyBLOCK/?$`), "PyBLOCK, a node dashboard", "PyBLOCK dashboard", ""},
	{regexp.MustCompile(`(?i)\(PyBLOCK|pyblock`), "Named like PyBLOCK, a node dashboard", "Named like PyBLOCK", ""},
	// Projects whose name matches; none of them publishes the exact string,
	// so "Named like".
	// Urbit reaches Bitcoin through its %btc-provider (github.com/urbit/urbit);
	// Symbiosis Finance runs a Bitcoin bridge (symbiosis.finance/bridge-btc);
	// DexCash Network is a blockchain of its own (dexcash.org). curl is the
	// download tool and does not speak Bitcoin's protocol, so a peer naming it
	// borrows the name.
	{regexp.MustCompile(`^urbit$`), "Named like Urbit's Bitcoin software (%btc-provider)", "Named like Urbit", ""},
	{regexp.MustCompile(`/Symbiosis Bitcoin Bridge:`), "Named like the Bitcoin bridge of Symbiosis Finance, on btcd's Go library", "Named like Symbiosis", ""},
	{regexp.MustCompile(`^/curl:`), "Named like curl, the download tool; curl itself does not speak Bitcoin's protocol", "Named like curl", ""},
	{regexp.MustCompile(`^/dexcash-nodes:`), "Named like DexCash Network, a blockchain of its own", "Named like DexCash", ""},
	// Bitnodes' crawler sends "/bitnodes.io:0.3/" (protocol.py in
	// ayeowch/bitnodes); any other name with Bitnodes in it, /alt-bitnodes:/
	// or /Metrika-Bitnodes:/, is only named like it. BTC Nodes (btcnodes.io),
	// built from Bitnodes' code, keeps the same default.
	// Source: ayeowch/bitnodes conf/crawl.conf.default L53 @036dc5c (checked 2026-11-24)
	// Source: brunneis/btcnodes conf/crawl.conf.default L55 @f73ab93 (checked 2026-11-24)
	{regexp.MustCompile(`^/bitnodes\.io:`), "Bitnodes' crawler, or one built from its code such as BTC Nodes: maps reachable nodes", "Bitnodes crawler", ""},
	// alt-bitnodes: USER_AGENT in deploy/install.sh (ifuensan/alt-bitnodes),
	// HackNodes' public node count. /bitnodes.earn.com:0.1/: the crawler of a
	// LORIA/Inria study (conf/crawl.conf.default in jpeisenbarth/bitnodes,
	// branch add_statistics; CONCORDIA, last run 2020).
	// Source: ifuensan/alt-bitnodes deploy/install.sh L27 @511a4de (checked 2026-11-24)
	{regexp.MustCompile(`^/alt-bitnodes:`), "alt-bitnodes, HackNodes' crawler built on Bitnodes, counting reachable nodes", "alt-bitnodes crawler", ""},
	// Source: ayeowch/bitnodes conf/crawl.conf.default L44 @1035ba3 (checked 2026-11-24)
	{regexp.MustCompile(`^/bitnodes\.earn\.com:`), "A Bitnodes crawler run for a LORIA/Inria research study", "Bitnodes (Inria study)", ""},
	{regexp.MustCompile(`(?i)bitnodes`), "Named like Bitnodes, the public node map", "Named like Bitnodes", ""},
	// Seen from one netcup address, offering nothing and gone after a second;
	// the name fits a node map, but nothing public says whose.
	{regexp.MustCompile(`(?i)GlobalNodeMap`), "Named like a node map; nothing public says whose", "Named like a node map", ""},
	// illuminodes/mantle: the node says "/mantle:0.1.0/", its replay tool,
	// which fetches mainnet blocks from peers to validate them, says
	// "/mantle-replay-iouring:0.1.0/" (crates/fetcher/src/lib.rs); beside it
	// "/mantle-discover:0.1.0/" collects addresses with getaddr
	// (replay/src/discover_peers.rs) and "/mantle-rank-peers:0.1.0/" times
	// how fast a node answers its version (replay/src/rank_peers.rs).
	// Source: illuminodes/mantle crates/mantle-node/src/main.rs L33 @bc1522a (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/mantle:`), "Mantle, a Bitcoin full node written from scratch in Rust (illuminodes)", "Mantle node (Rust)", ""},
	// Source: illuminodes/mantle crates/replay/src/fetcher.rs L494 @18bd7fa (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/mantle-replay`), "Mantle's replay tool: fetches blocks from nodes to check them with Mantle, a Bitcoin node written from scratch in Rust", "Mantle replay tool", ""},
	// Source: illuminodes/mantle replay/src/discover_peers.rs L173 @c3a4fe4 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/mantle-discover`), "Mantle's peer finder: asks nodes for addresses of other nodes", "Mantle peer finder", ""},
	// Source: illuminodes/mantle replay/src/rank_peers.rs L534 @c3a4fe4 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/mantle-rank-peers`), "Mantle's speed test: times how fast a node answers", "Mantle speed test", ""},
	// btcd's wire package sends "/btcwire:0.5.0/" when a program sets no name
	// of its own (wire/msgversion.go), and its peer package's example sets
	// "peer" 1.0.0 (peer/example_test.go). On the node this was written for,
	// 21. to 25.10.2026, the bare name asked for addresses and headers and
	// wanted no transactions.
	// Source: btcsuite/btcd wire/msgversion.go L20 @3cb9f60 (checked 2026-11-24)
	{regexp.MustCompile(`^/btcwire:[^/]*/$`), "A program on btcd's Go library that set no name of its own", "btcd library, no own name", ""},
	// btcd's example and wire default together (15.11.2026): exactly this.
	// Source: btcsuite/btcd peer/example_test.go L24 @2c46702 (checked 2026-11-24)
	{regexp.MustCompile(`^/btcwire:0\.5\.0/peer:1\.0\.0/$`), "A program built from the example code in btcd's Go library, keeping its name", "btcd sample code", ""},
	{regexp.MustCompile(`^/btcwire:[^/]*/peer:1\.0\.0/$`), "Probably a program built from the example code in btcd's Go library, keeping its name", "Probably btcd sample code", ""},
	// RESET (Trac-Systems/reset) is a coin clocked by Bitcoin's blocks; a
	// share of its reward goes to whoever brings in a new block first. Its
	// agents fetch from mempool.space unless told otherwise and its code has
	// no P2P client, so these names are a guess from the name, the behaviour
	// and the timing: its repository was created on 15.10.2026, and the
	// measured record of the node this was written for, from 13.9., shows the
	// first reset-* name on 20.10.; by 25.10. there were seven, asking only
	// for headers, most wanting no transactions. That is timing and behaviour,
	// not a source: "Named like" (13.11.2026).
	{regexp.MustCompile(`(?i)^/reset-`), "Named like RESET, a coin clocked by Bitcoin's blocks that pays a small reward to whoever brings in a new block first", "Named like RESET", "RESET"},
	// The Real Bitcoin (TRB), a bitcoind kept on the old 0.5.4 line: its
	// client name and version 99999, written as "/therealbitcoin.org:0.9.99.99/"
	// by FormatSubVersion (src/util.cpp L899-905).
	// Source: irnb/therealbitcoin src/knobs.h L4-5 @ec4f7f9 (checked 2026-11-24)
	{regexp.MustCompile(`^/therealbitcoin\.org:`), "The Real Bitcoin (TRB), a bitcoind kept on the old 0.5.4 line", "The Real Bitcoin (0.5.4)", ""},
	// JWRD Computing (jwrd.net) maintains its own bitcoind of the old 0.5
	// line. Its version 0.9.99.99 is TRB's 99999, so it is likely a TRB
	// descendant; nothing public says it sends this name.
	{regexp.MustCompile(`^/jwrd\.net:`), "Named like JWRD Computing's own bitcoind, a fork of an early version", "Named like JWRD", ""},
	// btcd's wire library with "Satoshi" put after it: two monitors do that,
	// each with its own version, so only the version tells them apart.
	// Source: mit-dci/pooldetective blockobserver/coinparam/bitcoin.go L96-97 @232317e (checked 2026-11-24)
	{regexp.MustCompile(`^/btcwire:[^/]*/Satoshi:0\.19/$`), "Pool Detective, by MIT's Digital Currency Initiative: watches what mining pools do", "MIT's Pool Detective", "Pool Detective"},
	// Source: CIRCL/pbtc peer/peer.go L52-53 @05437a4 (checked 2026-11-24)
	{regexp.MustCompile(`^/btcwire:[^/]*/Satoshi:0\.9\.3/$`), "pbtc, CIRCL's passive Bitcoin network monitor", "CIRCL's pbtc monitor", "pbtc"},
	{regexp.MustCompile(`^/btcwire:[^/]*/Satoshi:`), "Probably a network monitor on btcd's library that adds Core's name, like Pool Detective or CIRCL's pbtc", "Probably a btcd monitor", ""},
	// Source: mit-dci/lit uspv/init.go L130 @8c3d3b4 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)/lit:v`), "lit, a Lightning node by MIT's Digital Currency Initiative, no longer developed", "lit Lightning node (MIT)", ""},
	// observer/src/protocol.rs in 0xB10C/p2p-observer, work in progress in
	// October 2026; extractors/p2p in peer-observer/peer-observer, which a
	// node connects to itself when told to (addnode).
	// src/sync/stages/pull/peer.rs in maestro-org/maestro-symphony;
	// src/crawl.rs in achow101/dnsseedrs; Satscraft/bip324-probe.
	// /your-fork:/ has no sentence of its own (see kinds.go). The measured
	// record of the node this was written for, 13.9. to 25.10.2026: 241 sessions from 187 addresses, offering nothing, no
	// transactions, gone after 18 seconds at the median; the same handshake
	// as Bitnodes' crawler (version, sendaddrv2, verack, then getaddr). A
	// session that stays, like Bitnodes' own, asks for nothing: getpeerinfo
	// on 31.10.2026, one of 22 hours, no getaddr.
	// Source: maestro-org/maestro-symphony src/sync/stages/pull/peer.rs L36 @ef66a53 (checked 2026-11-24)
	{regexp.MustCompile(`^/symphony:`), "Maestro Symphony, a Bitcoin indexer and API server", "Maestro Symphony indexer", ""},
	// Source: achow101/dnsseedrs src/crawl.rs L176 @127da83 (checked 2026-11-24)
	{regexp.MustCompile(`^/crawlrs:`), "dnsseedrs, a DNS seeder by Ava Chow, a Bitcoin Core maintainer", "dnsseedrs (Ava Chow)", ""},
	// Source: Satscraft/bip324-probe src/v1.rs L43 @2cb7d7f (checked 2026-11-24)
	{regexp.MustCompile(`^/bip324-probe:`), "bip324-probe, a tool that checks whether nodes offer encrypted v2 connections", "bip324-probe v2 checker", ""},
	// Source: 0xB10C/p2p-observer observer/src/protocol.rs L111 @e265bb9 (checked 2026-11-24)
	{regexp.MustCompile(`^/p2p-observer:`), "p2p-observer, 0xB10C's tool that connects to nodes to watch the network", "0xB10C's p2p-observer", ""},
	// Source: peer-observer/peer-observer extractors/p2p/src/lib.rs L39 @8889014 (checked 2026-11-24)
	{regexp.MustCompile(`^/p2p-extractor:`), "peer-observer's p2p extractor: nodes are told to connect to it, and it records what they send", "peer-observer's extractor", ""},
	// Sources, 14.11.2026: AgenticBitcoin's USER_AGENT
	// (crates/abtc-adapters/src/network/mod.rs in casey-bowman/agentic-bitcoin),
	// SatoshiSharp's UserAgent (src/ConsoleApp/MainBlockDownload.cs in
	// tqfoley/BtcSatoshiSharp), NBXplorer's "NBXplorer-" + a random number
	// (NBXplorer/Backend/Indexer.cs in btcpayserver/NBXplorer).
	// Source: casey-bowman/agentic-bitcoin crates/abtc-adapters/src/network/mod.rs L42 @c192131 (checked 2026-11-24)
	{regexp.MustCompile(`^/AgenticBitcoin:`), "AgenticBitcoin, Bitcoin rewritten in Rust by AI agents", "AgenticBitcoin (AI-made)", ""},
	// Source: tqfoley/BtcSatoshiSharp src/ConsoleApp/MainBlockDownload.cs L110 @3ce6490 (checked 2026-11-24)
	{regexp.MustCompile(`^/SatoshiSharp:`), "SatoshiSharp, parts of Bitcoin written in C#", "SatoshiSharp (C#)", ""},
	// Source: btcpayserver/NBXplorer NBXplorer/Backend/Indexer.cs L284 @b3ab010 (checked 2026-11-24)
	{regexp.MustCompile(`^NBXplorer-`), "NBXplorer, the indexer behind BTCPay Server", "NBXplorer (BTCPay)", ""},
	// NodeConstants.scala in bitcoin-s/bitcoin-s; indexer.rs in stacks-core.
	// Source: bitcoin-s/bitcoin-s core/src/main/scala/org/bitcoins/core/api/node/constant/NodeConstants.scala L4 @a928f38 (checked 2026-11-24)
	{regexp.MustCompile(`^/bitcoin-s:`), "bitcoin-s, a Bitcoin toolkit in Scala with a light client on compact block filters", "bitcoin-s toolkit (Scala)", ""},
	// Source: stacks-network/stacks-core stackslib/src/burnchains/bitcoin/indexer.rs L54 @7c17b1d (checked 2026-11-24)
	{regexp.MustCompile(`^Stacks/`), "A Stacks node: follows Bitcoin's blocks for the Stacks chain built on it", "Stacks node", ""},
	// userAgentName "utreexod" (server.go in utreexo/utreexod); spvwallet's
	// UserAgent (config.go in OpenBazaar/spvwallet).
	// Source: utreexo/utreexod server.go L73 @ccd1afc (checked 2026-11-24)
	{regexp.MustCompile(`/utreexod:`), "utreexod, a Utreexo full node built on btcd", "utreexod Utreexo node", ""},
	// Source: cloudhead/nakamoto p2p/src/fsm.rs L70 @76ab7a3 (checked 2026-11-24)
	{regexp.MustCompile(`^/nakamoto:`), "nakamoto, a light client on compact block filters", "nakamoto light client", ""},
	// Source: OpenBazaar/spvwallet config.go L73 @225c519 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)/spvwallet:`), "spvwallet, the wallet of OpenBazaar, a marketplace closed in 2021", "OpenBazaar's spvwallet", ""},
	// Autarkysoft's C# toolkit (Configuration.cs); it pushes transactions
	// under "/Satoshi:0.22.0/" instead, so only its own name lands here.
	// Source: Autarkysoft/Denovo Src/Denovo/Models/Configuration.cs L26 @858aff3 (checked 2026-11-24)
	{regexp.MustCompile(`^/Denovo:`), "Denovo, a Bitcoin toolkit in C#: pushes transactions, mines on testnet", "Denovo toolkit (C#)", ""},
	// network.py in jimmysong/programmingbitcoin, and session7/network.py in
	// jimmysong/pb-exercises for the course.
	// Source: jimmysong/programmingbitcoin code-ch10/network.py L111 @1aa1a63 (checked 2026-11-24)
	{regexp.MustCompile(`^/programming(bitcoin|blockchain):`), "Code from Jimmy Song's book or course Programming Bitcoin", "Programming Bitcoin code", ""},
	// The example code of GladosBlueWallet/bip324 (examples/signet-getaddr.ts),
	// the TypeScript library Blueberry's transport was ported from.
	// Source: GladosBlueWallet/bip324 examples/signet-getaddr.ts L32 @8abbc6d (checked 2026-11-24)
	{regexp.MustCompile(`^/bip324-ts:`), "bip324-ts, a TypeScript library for encrypted v2 connections (its example code)", "bip324-ts library", ""},
	// dsn.kastel.kit.edu/bitcoin, the DSN group at KIT: the page says it
	// measures the network for research, but names no user agent. The domain
	// in the name points there.
	{regexp.MustCompile(`(?i)^/dsn\.[a-z]+\.kit\.edu/`), "Probably KIT Karlsruhe's Bitcoin network monitoring", "Probably KIT monitoring", "KIT"},
	// app/network/scanner.py in DogukanBaysal/DaeConn. The other two only from
	// 0xB10C's list of 15.05.2023, what he saw, not their code (entities.toml
	// in bitcoin-noc/network-entities, formerly banlist): "Names".
	// Source: DogukanBaysal/DaeConn app/network/scanner.py L90 @698f77c (checked 2026-11-24)
	{regexp.MustCompile(`^/connector/$`), "DaeConn, a research tool that measures the networks of Bitcoin and its forks", "DaeConn research tool", ""},
	// nodes.mom.market: MOM.market is Multiven's marketplace, and Multiven said
	// in March 2018 it monitored Bitcoin nodes (via nodes.multiven.io); nothing
	// names this subdomain or its crawler. bitcoinstats.com: Christian Decker
	// runs its DNS seed crawler (delvingbitcoin.org, 01.03.2024), but no source
	// sets this name (GPT research, 23.11.2026).
	{regexp.MustCompile(`^/nodes\.mom\.market:`), "Named like nodes.mom.market, a site that monitors Bitcoin nodes", "Named like mom.market", ""},
	{regexp.MustCompile(`^/btcwire:[^/]*/bitcoinstats\.com:`), "Named like bitcoinstats.com, a site with statistics on the Bitcoin network", "Named like bitcoinstats", ""},
	// btcd's wire library with the app name after it left empty.
	// Source: btcsuite/btcd wire/msgversion.go L257 @3cb9f60 (checked 2026-11-24)
	{regexp.MustCompile(`^/btcwire:[^/]*/:/$`), "A program on btcd's Go library that left its own name empty", "btcd library, empty name", ""},
	// Source: btcsuite/btcd server.go L67 @2c46702 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)btcd:`), "btcd, a full node written in Go", "btcd node (Go)", ""},
	// Sources, 14.11.2026: BC_USER_AGENT (include/bitcoin/system/preprocessor.hpp
	// in libbitcoin/libbitcoin-system), USER_AGENT (lib/net/common.js in
	// bcoin-org/bcoin), UserAgent (client/common/config.go in piotrnar/gocoin).
	// Source: libbitcoin/libbitcoin-network include/bitcoin/network/settings.hpp L32 @c66f70c (checked 2026-11-24)
	{regexp.MustCompile(`(?i)libbitcoin`), "libbitcoin, a full node written in C++, independent of Core", "libbitcoin node", ""},
	// Source: bcoin-org/bcoin lib/net/common.js L144 @a8dee6d (checked 2026-11-24)
	{regexp.MustCompile(`(?i)/bcoin:`), "bcoin, a full node written in JavaScript", "bcoin node (JS)", ""},
	// Source: piotrnar/gocoin client/common/config.go L504 @ca8789c (checked 2026-11-24)
	{regexp.MustCompile(`(?i)gocoin`), "Gocoin, a full node and wallet written in Go", "Gocoin node and wallet", ""},
	// gosuda/bitcoin-rs, crates/consensus/src/primitives/version.rs: "/bitcoin-rs:<version>/".
	// Source: gosuda/bitcoin-rs crates/consensus/src/primitives/version.rs L3 @9acd366 (checked 2026-11-24)
	{regexp.MustCompile(`^/bitcoin-rs:`), "bitcoin-rs, a full node written in Rust, independent of Core", "bitcoin-rs node (Rust)", ""},
	// Full nodes, each string as its source sets it (14.11.2026):
	// bitcoinpr-p2p/src/messages.rs in BitcoinPR/BitcoinPR; constants.py in
	// btclib-org/btclib-node; network_manager.rs in BTCDecoded/blvm-node;
	// crates/node/src/node.rs in coreyphillips/bitcoin-spec-node (validates,
	// serves no blocks); wire/types.odin in Galaxoid-Labs/forseti;
	// VersionMessage.swift in swift-bitcoin/swift-bitcoin; wolfe-types in
	// refined-element/BitcoinWolfe; peer_manager.rs in subcoin-project/subcoin;
	// new_peer.rs in jrawsthorne/rust-bitcoin-node; NodeSettings.cs and
	// ConnectionManager.cs in stratisproject/StratisBitcoinFullNode ("StratisNode:
	// <version> (<protocol>)", no slashes); protocol/version.rb in
	// mhanne/bitcoin-ruby.
	// Source: BitcoinPR/BitcoinPR bitcoinpr-p2p/src/messages.rs L24-33 @0fda9d1 (checked 2026-11-24)
	{regexp.MustCompile(`^/bitcoinpr:`), "BitcoinPR, a Bitcoin full node written in Rust", "BitcoinPR node (Rust)", ""},
	// Source: btclib-org/btclib-node src/btclib_node/constants.py L136 @d74075c (checked 2026-11-24)
	{regexp.MustCompile(`^/btclib:`), "btclib-node, a Bitcoin full node written in Python", "btclib node (Python)", ""},
	// Source: BTCDecoded/blvm-node src/network/network_manager.rs L2502 @65bf5dc (checked 2026-11-24)
	{regexp.MustCompile(`^/Bitcoin Commons:`), "Bitcoin Commons (BLVM), a Bitcoin full node written in Rust", "Bitcoin Commons (Rust)", ""},
	// Source: coreyphillips/bitcoin-spec-node crates/node/src/node.rs L536 @2396240 (checked 2026-11-24)
	{regexp.MustCompile(`^/bitcoin-spec-node:`), "bitcoin-spec-node, a research node in Rust that checks every block but serves none", "bitcoin-spec-node", ""},
	// Source: Galaxoid-Labs/forseti wire/types.odin L20 @069d3f6 (checked 2026-11-24)
	{regexp.MustCompile(`^/Forseti:`), "Forseti, an experimental Bitcoin full node written in Odin", "Forseti node (Odin)", ""},
	// Source: swift-bitcoin/swift-bitcoin src/bitcoin-transport/messages/VersionMessage.swift L5 @4a87c60 (checked 2026-11-24)
	{regexp.MustCompile(`^/SwiftBitcoin:`), "Swift Bitcoin, an experimental Bitcoin full node written in Swift", "Swift Bitcoin node", ""},
	// Source: refined-element/BitcoinWolfe crates/wolfe-types/src/lib.rs L22 @e419cec (checked 2026-11-24)
	{regexp.MustCompile(`^/BitcoinWolfe:`), "BitcoinWolfe, an experimental node in Rust on libbitcoinkernel, Core's validation engine", "BitcoinWolfe node", ""},
	// Source: subcoin-project/subcoin crates/subcoin-network/src/peer_manager.rs L77 @9999efc (checked 2026-11-24)
	{regexp.MustCompile(`^/Subcoin:`), "Subcoin, an experimental Bitcoin node written in Rust", "Subcoin node (Rust)", ""},
	// Source: jrawsthorne/rust-bitcoin-node src/net/new_peer.rs L206 @9c7d4f5 (checked 2026-11-24)
	{regexp.MustCompile(`^/rust_bitcoin_node:`), "rust-bitcoin-node, a Rust full node, no longer developed", "rust-bitcoin-node", ""},
	// Source: stratisproject/StratisBitcoinFullNode src/Stratis.Bitcoin/Configuration/NodeSettings.cs L135 @f52659f (checked 2026-11-24)
	{regexp.MustCompile(`^StratisNode:`), "Stratis Bitcoin Full Node, a node in C#, archived in 2021", "Stratis node (C#)", ""},
	// Source: mhanne/bitcoin-ruby lib/bitcoin/protocol/version.rb L21 @0977b05 (checked 2026-11-24)
	{regexp.MustCompile(`^/bitcoin-ruby:`), "bitcoin-ruby's node, a Ruby node from before SegWit", "bitcoin-ruby node", ""},
	// Crawlers and research tools, each with its source (14.11.2026):
	// crawler_config.yml in blakebjorn/opennodes; src/version.zig in wslcb/btz;
	// fluxture/bitcoin.py in crytic/fluxture (Trail of Bits); net.py in
	// justinmoon/crawler; cmd.go in shazow/btc-crawl; explorer/scripts/
	// crawler.py in bitcoinanatomy/bitcoinanatomy; backend/src/p2p/p2p.rs in
	// jochemin/crawly; src/protocol.rs in danielabrozzoni/new-p2p-crawler;
	// src/handshake/handshake.ts in OriginalEveres/bitcoin-node-scanner;
	// bitcoin_probe.py in dutenrapha/bitcoin-topology-discovery; crawl_btcd.go
	// in p2pobservatory/bitcoin-crawler L35 ("/crawler:0.1/", checked
	// 15.11.2026; no other sender found, 23.11.2026); src/main/kotlin/crawler.kt in
	// bitcoinj/httpseed; netmine.cfg in jameslitton/coinscope (University of
	// Maryland); setting.properties in Crepepepe/JBS (Chung-Ang University;
	// Judgegao/Bitcoin-Node-Scanner reuses it); Main.java in vs-uulm/CoinView
	// (Ulm University); txecho.go in thickzero/TxEcho.
	// Source: blakebjorn/opennodes crawler_config.yml L60 @b540e4f (checked 2026-11-24)
	{regexp.MustCompile(`^/open-nodes:`), "Open Nodes, a crawler that maps reachable nodes", "Open Nodes crawler", ""},
	// Source: wslcb/btz src/version.zig L38 @3fedab9 (checked 2026-11-24)
	{regexp.MustCompile(`^/btz:`), "btz, a crawler in Zig that asks nodes for addresses", "btz crawler", ""},
	// Source: crytic/fluxture fluxture/bitcoin.py L467 @9c384d4 (checked 2026-11-24)
	{regexp.MustCompile(`^fluxture$`), "Fluxture, Trail of Bits' crawler for peer-to-peer networks", "Fluxture (Trail of Bits)", ""},
	// Source: justinmoon/crawler net.py L129 @bc1f0b0 (checked 2026-11-24)
	{regexp.MustCompile(`^/buidl-bootcamp/`), "The crawler of Justin Moon's BUIDL Bootcamp, a teaching DNS seed from 2019", "BUIDL Bootcamp crawler", ""},
	// Source: shazow/btc-crawl cmd.go L32 @601a21d (checked 2026-11-24)
	{regexp.MustCompile(`^/btc-crawl:`), "btc-crawl, a crawler by Andrey Petrov (shazow)", "btc-crawl crawler", ""},
	// Source: bitcoinanatomy/bitcoinanatomy explorer/scripts/crawler.py L45 @3d9c5d3 (checked 2026-11-24)
	{regexp.MustCompile(`^/btc-crawler:`), "Anatomy of Bitcoin's crawler, for its map of reachable nodes", "Anatomy of Bitcoin", ""},
	// Source: jochemin/crawly backend/src/p2p/p2p.rs L538 @797a536 (checked 2026-11-24)
	{regexp.MustCompile(`^Crawly$`), "Crawly, a crawler behind a node-map app by jochemin", "Crawly crawler", ""},
	// Source: danielabrozzoni/new-p2p-crawler src/protocol.rs L21 @811a35b (checked 2026-11-24)
	{regexp.MustCompile(`^/new-p2p-crawler:`), "new-p2p-crawler, Daniela Brozzoni's crawler of reachable nodes", "new-p2p-crawler", ""},
	// Source: OriginalEveres/bitcoin-node-scanner src/handshake/handshake.ts L157 @bd99850 (checked 2026-11-24)
	{regexp.MustCompile(`^/bitcoin-node-scanner:`), "bitcoin-node-scanner, a crawler that asks nodes for addresses", "bitcoin-node-scanner", ""},
	// Source: dutenrapha/bitcoin-topology-discovery bitcoin_probe.py L13 @ff1cfe0 (checked 2026-11-24)
	{regexp.MustCompile(`^/topology-probe:`), "A topology crawler (bitcoin-topology-discovery) that asks nodes for addresses", "Topology crawler", ""},
	// Source: p2pobservatory/bitcoin-crawler crawl_btcd.go L35 @11dca5a (checked 2026-11-24)
	{regexp.MustCompile(`^/crawler:0\.1/$`), "The crawler of P2P Observatory (IMDEA Networks)", "P2P Observatory crawler", ""},
	// Source: jameslitton/coinscope netmine.cfg L40 @7edb2a9 (checked 2026-11-24)
	{regexp.MustCompile(`^/Coinscope-GH:`), "CoinScope, a University of Maryland study of Bitcoin's network", "CoinScope (UMD study)", ""},
	// Source: Crepepepe/JBS setting.properties L3 @664d28c (checked 2026-11-24)
	{regexp.MustCompile(`^/JBS:`), "JBS, a Bitcoin node scanner from university studies", "JBS research scanner", ""},
	// Source: thickzero/TxEcho txecho.go L748 @06ab7c2 (checked 2026-11-24)
	{regexp.MustCompile(`^/txecho:`), "TxEcho, a payment monitor that watches nodes for transactions and blocks", "TxEcho payment monitor", ""},
	// No public source. On the node this was written for, 28.10. to
	// 01.11.2026: 23 sessions from 16 addresses, all in Cloudflare's network
	// (AS13335), none sending BIP 331's sendpackages, gone after 11 s (median).
	// Only the name is said: what one node saw is no rule for all of them.
	{regexp.MustCompile(`^/cf-package-relay:`), "Named like package relay, which passes related transactions on together (BIP 331)", "Named like package relay", ""},
	// Bitcoin Cash clients carry its block size as "(EB8.0)" too, so they
	// come before Classic.
	// Source: Bitcoin-ABC/bitcoin-abc src/clientversion.cpp L19 @012f530 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)Bitcoin ABC`), "Bitcoin ABC, a Bitcoin Cash client, eCash's node since 2020", "Bitcoin ABC (BCH)", ""},
	// src/clientversion.cpp in BitcoinUnlimited/BitcoinUnlimited:
	// "BitcoinUnlimited" from 2015 (b126b10), a big-block client for Bitcoin,
	// whose Bitcoin edition ended in January 2018 (85f5bd6); "BUCash" for the
	// Bitcoin Cash edition from 25.07.2017 (849199f), "BCH Unlimited" with a
	// space from 02.04.2020 (4ce5b9e). Before the "(EB" rule: they send it too.
	// Source: BitcoinUnlimited/BitcoinUnlimited src/clientversion.cpp L16 @b126b10 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)^/BitcoinUnlimited:`), "Bitcoin Unlimited, a big-block client for Bitcoin; this edition ended in 2018", "Bitcoin Unlimited (old)", ""},
	// Source: BitcoinUnlimited/BitcoinUnlimited src/clientversion.cpp L20 @bucash1.1.1.1 (checked 2026-11-24)
	// Source: BitcoinUnlimited/BitcoinUnlimited src/clientversion.cpp L19 @05de381 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)BUCash|BCH ?Unlimited`), "Bitcoin Unlimited's Bitcoin Cash edition (BUCash, now BCH Unlimited)", "Bitcoin Unlimited (BCH)", ""},
	// Source: bitcoin-sv/bitcoin-sv src/clientversion.cpp L17 @6504a3a (checked 2026-11-24)
	{regexp.MustCompile(`(?i)Bitcoin SV`), "Bitcoin SV, the node of BSV, which split from Bitcoin Cash in 2018", "Bitcoin SV (BSV)", ""},
	// Source: bitcoinxt/bitcoinxt src/clientversion.cpp L23 @48bbdd4 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)Bitcoin XT`), "Bitcoin XT, a fork of Core for bigger blocks that went on to Bitcoin Cash", "Bitcoin XT", ""},
	// Source: SuperBitcoin/SuperBitcoin src/sbtccore/clientversion.cpp L17 @90b776e (checked 2026-11-24)
	{regexp.MustCompile(`(?i)SuperBitcoin`), "Super Bitcoin, a fork of Bitcoin from December 2017 with a chain of its own", "Super Bitcoin fork", ""},
	// USER_AGENT "/iriumd:1.9.66/" (src/p2p_wire.rs in iriumlabs/irium), the
	// one wire code it uses for Bitcoin's network too: version, getheaders,
	// headers (15.11.2026).
	// Source: iriumlabs/irium src/p2p_wire.rs L35 @85c2c0e (checked 2026-11-24)
	{regexp.MustCompile(`(?i)iriumd`), "Irium, an altcoin with a chain of its own; its node also reads Bitcoin's block headers", "Irium altcoin", ""},
	// v-dimension/vds-core sends "/MagicBean:x/" (CLIENT_NAME in
	// src/clientversion.cpp), so "VDS <version>" is only the name.
	// Bitcoin Finance's own announcement (bitcointalk topic 5500641, June
	// 2024) prints "Satoshi-BTF(BitcoinFinance)-Main-DCR:0.15.1" as its node's
	// version; no code of it is public, hence "Probably".
	{regexp.MustCompile(`^/Satoshi-BTF\(BitcoinFinance\)`), "Probably Bitcoin Finance (BTF), a fork of Bitcoin with a chain of its own", "Probably BTF fork", ""},
	{regexp.MustCompile(`(?i)vds[_ ]`), "Named like VDS, an altcoin with a chain of its own", "Named like VDS", ""},
	// CLIENT_NAME("Bitcoin Interest") and CLIENT_NAME("Bitcoin Clashic ABCD")
	// (src/clientversion.cpp in BitcoinInterestOfficial/BitcoinInterest and
	// Bitcoin-Clashic/bitcoin-clashic).
	// Source: BitcoinInterestOfficial/BitcoinInterest src/clientversion.cpp L16 @671384e (checked 2026-11-24)
	{regexp.MustCompile(`^/Bitcoin Interest:`), "Bitcoin Interest, a fork of Bitcoin with a chain of its own", "Bitcoin Interest fork", ""},
	// Source: Bitcoin-Clashic/bitcoin-clashic src/clientversion.cpp L16 @3d18496 (checked 2026-11-24)
	{regexp.MustCompile(`^/Bitcoin Clashic`), "Bitcoin Clashic, a fork of Bitcoin Cash from 2017", "Bitcoin Clashic (BCH)", ""},
	// Source: bitcoinclassic/bitcoinclassic src/Application.cpp L172 @4c27454 (checked 2026-11-24)
	{regexp.MustCompile(`(?i)Classic:`), "Bitcoin Classic: a Bitcoin Cash client", "Bitcoin Classic (BCH)", ""},
	// "(EB8)" is the block size a big-block client accepts: Classic, Unlimited
	// and others all send it, so alone it says only that much.
	{regexp.MustCompile(`\(EB[0-9]`), "Probably a big-block client: \"EB\" is the block size it accepts", "Probably big-block client", ""},
}

// walletClaim is the wallet a peer's name claims, from its about sentence:
// "Named like Elastum, a former exchange; ..." claims "the wallet Elastum".
// The hedge belongs to the sentence, not to the claim.
func walletClaim(about string) string {
	name, _, _ := strings.Cut(about, ",")
	for _, hedge := range []string{"Named like ", "Probably "} {
		name = strings.TrimPrefix(name, hedge)
	}
	return "the wallet " + name
}

// fakeAbout says, for a disguised peer, what it claims and what it probably is. The
// claim comes from its user agent, the guess from what Core observed of it -
// the one part nobody can write into a string.
func fakeAbout(subver, walletAbout string, services []string) string {
	claim := "Bitcoin Core"
	if v := coreVersion.FindStringSubmatch(subver); v != nil {
		claim += " " + v[1]
	}
	wallet := walletAbout != ""
	if strings.HasPrefix(walletAbout, "bitcoinj,") {
		// The library without an app name claims bitcoinj, not Core or a wallet.
		claim = claimOf(subver)
	} else if wallet {
		claim = walletClaim(walletAbout)
	}
	var guess string
	blocks, v2 := false, false
	for _, s := range services {
		blocks = blocks || s == "NETWORK"
		v2 = v2 || s == "P2P_V2"
	}
	switch {
	case wallet && blocks:
		guess = "a node or scanner behind an old wallet name - it offers blocks, which no wallet does"
	case wallet && v2:
		guess = "a node or scanner behind an old wallet name - it speaks v2, which no wallet of that age does"
	case misspeltCore(subver) != "":
		guess = "not Bitcoin Core - " + misspeltCore(subver)
		if services != nil && len(services) == 0 {
			guess += " and offers no services"
		}
	default:
		guess = "unknown software"
	}
	return "Claims: " + claim + "\nProbably: " + guess
}

// claimOf is the software a user agent names, readably: "/bcoin:v1.0.0/" is
// "bcoin v1.0.0", Core and Knots are spelled out.
func claimOf(subver string) string {
	if v := coreVersion.FindStringSubmatch(subver); v != nil {
		if strings.Contains(subver, "Knots:") {
			return "Bitcoin Knots (Core " + v[1] + ")"
		}
		return "Bitcoin Core " + v[1]
	}
	first, _, _ := strings.Cut(strings.Trim(subver, "/"), "/")
	return strings.Replace(first, ":", " ", 1)
}

// sessionLine is the line for a name none of the rules know, so no peer is left
// without one. Said as what we know, not as what it is: others may know it.
func sessionLine(services []string, relayTxes *bool, recv, sent map[string]int64, connTime int64, now time.Time) (line, short string, saysNoTx bool) {
	// A peer that has been connected for a while and asked for nothing is no
	// full node, whatever services it lists. Seen 21. to 26.10.2026: /your-fork:0.1/
	// and /alt-bitnodes:0.1/ staying for hours, hearing of transactions
	// and never fetching one, /pydroid-pool:3/ wanting neither.
	if quietAfterHandshake(recv, connTime, now) {
		if wantsTx(relayTxes) {
			// "Hears" only if we announced something to it.
			if sent["inv"] > 0 {
				return "Only listens: hears new transactions, fetches none", "Only listens", false
			}
			return "Only listens: asks for nothing", "Only listens", false
		}
		return "Only listens: wants no transactions, asks for nothing", "Only listens", true
	}
	// What it did, where Core counted it: a name nobody knows still says
	// little, its messages say more. Seen on the node this was written for,
	// 21. to 24.10.2026: /pwsh-pusher/ and /p2aburn/ pushing transactions
	// nobody asked for, others
	// fetching only headers, /Bitcoin:1.0.0/ asking for addresses each time,
	// /relay-geo:0.1/ fetching every transaction and announcing none.
	// A real node always offers a service, so one that offers none cannot be
	// the node reading below; nil means Core did not say.
	maybeNode := services == nil || len(services) > 0
	or := func(node, other string) string {
		if maybeNode {
			return node + ", or " + other
		}
		return other
	}
	switch {
	// BIP 35: "send me your whole mempool". Seen 22. to 26.10.2026 as /curl:8.5.0/,
	// twenty times from one address, each right after the handshake.
	case recv["mempool"] > 0:
		return "Asks for our mempool", "Asks for our mempool", false
	case recv["tx"] > 0 && recv["inv"] == 0:
		return "Pushes transactions without announcing them", "Pushes tx unannounced", false
	// Headers and no transactions is also what a real node does on a
	// connection for blocks only (2.2% of 2,553 Core and Knots sessions on
	// one node, 16. to 28.10.2026), so both readings are given. One that asks
	// for addresses too runs with blocksonly=1 if it is a node (noTxWhy).
	case refusesTx(relayTxes) && (recv["getheaders"] > 0 || recv["sendheaders"] > 0) && recv["getdata"] == 0 && recv["getaddr"] > 0:
		return "Asks for headers and addresses and wants no transactions: " + or("a node running blocksonly", "a program that tracks the chain"), "Headers, addresses only", true
	case refusesTx(relayTxes) && (recv["getheaders"] > 0 || recv["sendheaders"] > 0) && recv["getdata"] == 0:
		return "Asks for headers and wants no transactions: " + or("a block-only connection", "a program that tracks the chain"), "Asks only for headers", true
	// Asking for data alone is no sign of listening: a peer that fetches
	// blocks asks the same way. Only transactions sent to it count.
	case wantsTx(relayTxes) && recv["getdata"] > 0 && sent["tx"] > 0 && recv["inv"] == 0 && recv["tx"] == 0:
		// 0.5% of the real nodes above did the same: one that hears of
		// everything from others first has nothing to announce.
		return "Fetches transactions from your node and announces none: " + or("a node that hears of them elsewhere first", "a program collecting them"), "Takes tx, announces none", false
	case recv["getaddr"] > 0 && recv["getheaders"] == 0 && recv["getdata"] == 0:
		// 6.2% of the real nodes above did the same: one in sync asks for no headers.
		return "Asks for addresses, not for headers or blocks: " + or("a node already in sync", "a crawler"), "Asks only for addresses", false
	}
	// Offering blocks is only a guess at what it is, so what it did comes
	// first. Seen 04. to 06.11.2026: /spynode:0.1/ listing NETWORK, fetching
	// transactions for a day and a half and sending none back, was called a
	// full node.
	for _, s := range services {
		if s == "NETWORK" {
			return operatorNamedAbout, "Probably a full node", false
		}
	}
	return unknownAbout, "Unknown software", false
}

// otherAbout is sessionLine's sentence alone.
func otherAbout(services []string, relayTxes *bool, recv, sent map[string]int64, connTime int64, now time.Time) string {
	line, _, _ := sessionLine(services, relayTxes, recv, sent, connTime, now)
	return line
}

// The two lines of otherAbout that say nothing the peer did: a guess from
// its services, and no guess at all.
const (
	operatorNamedAbout = "Probably a full node under a name its operator chose"
	unknownAbout       = "Software with a name we don't recognise"
)

// observedPart is otherAbout's line without the guess at what such a peer
// is: under a sentence that already says what it is, only what it did
// belongs. "Only listens: asks for nothing" keeps what follows the colon,
// "Asks for headers ...: a program that tracks the chain" what comes before.
func observedPart(line string) string {
	if rest, ok := strings.CutPrefix(line, "Only listens: "); ok {
		return strings.ToUpper(rest[:1]) + rest[1:]
	}
	before, _, _ := strings.Cut(line, ": ")
	return before
}

// aboutRule is one line of aboutRules.
type aboutRule = struct {
	re          *regexp.Regexp
	text        string
	label, name string
}

// ruleOf is the first rule that knows the user agent. A rule reads the
// operator's brackets only when it asks for one itself (a "\(" in it, the
// rules for a setup's tag): a word an operator put there, "(my-monitor)" or
// "(neutrino)", does not describe a Core node as a census or a wallet -
// the kind reads the bare name the same way (readsBareName, 13.11.2026).
func ruleOf(subver string) (aboutRule, bool) {
	// No name, no sentence: the kind says "No name" too (classify).
	if emptyName.MatchString(subver) {
		return aboutRule{}, false
	}
	bare := bracketed.ReplaceAllString(subver, "")
	for _, r := range aboutRules {
		s := bare
		if strings.Contains(r.re.String(), `\(`) {
			s = subver
		}
		if r.re.MatchString(s) {
			return r, true
		}
	}
	return aboutRule{}, false
}

func aboutOf(subver string) string {
	r, _ := ruleOf(subver)
	return r.text
}

// noAgentLine is what a peer that sent its version but left its name empty
// did next, in a few words. Seen 21. to 26.10.2026, 429 such sessions: 113
// asked for headers and addresses like a full node, 43 asked for addresses
// and nothing else, the rest said nothing or only answered pings.
func noAgentLine(recv map[string]int64, connTime int64, now time.Time) (line, short string) {
	// The pill says it has no name; the sentence says what it did.
	switch {
	case recv["getheaders"] > 0 && recv["getaddr"] > 0:
		return "Asks for headers and addresses, like a full node does", "Acts like a full node"
	case recv["getheaders"] > 0 && recv["sendcmpct"] > 0:
		return "Asks for headers and compact blocks, like a full node does", "Acts like a full node"
	case recv["getaddr"] > 0 && recv["getheaders"] == 0 && recv["getdata"] == 0:
		return "Asks for addresses, not for headers or blocks", "Asks only for addresses"
	case quietAfterHandshake(recv, connTime, now):
		return "Introduced itself, then asked for nothing", "Asked for nothing"
	}
	return "Introduced itself without a name; nothing it did tells what it is", "Nothing it did tells"
}

// hemiUnnamed is how Hemi's Bitcoin code (tbc) names itself when the program
// built on it sets no name: "/%v:%v/" with the component empty, the version
// Major.Minor.Patch-dev+ the first nine characters of the commit, and Core
// drops the "+" (rawpeer.go, version.go in github.com/hemilabs/heminetwork).
// version.go takes the commit from the build of the main program: two of the
// three seen are commits of Hemi's op-geth (539f4e33c of 01.09.2026,
// c1771608d of 05.10.2026 in hemilabs/op-geth), the hVM node that builds in
// heminetwork/service/tbc (core/blockchain_hvm_migration.go) and sets no
// component; 923cba97a is in neither repository (checked 23.11.2026). Any
// other program on tbc without a name looks the same, so it is "probably".
var hemiUnnamed = regexp.MustCompile(`^/:[0-9]+\.[0-9]+\.[0-9]+-dev\+?[0-9a-f]{9}/$`)

const hemiUnnamedAbout = "Probably Hemi's hVM node: its op-geth with Hemi's Bitcoin code (tbc) built in, which leaves the name empty"

// noAgentAbout is noAgentLine's sentence alone.
func noAgentAbout(recv map[string]int64, connTime int64, now time.Time) string {
	line, _ := noAgentLine(recv, connTime, now)
	return line
}
