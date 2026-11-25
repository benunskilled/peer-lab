package main

import (
	"regexp"
	"strings"
)

// client is one line of clients.
type client struct {
	re                *regexp.Regexp
	kind              string // "" when the line gives no kind
	text, label, name string // text "" when the line gives no sentence
}

type kindRule struct {
	name   string
	re     *regexp.Regexp
	relays bool
}

// relays is whether software of a kind can pass a block on: nodes of any
// make, pool nodes, and the disguised ones, which could be either. Unknown
// software counts as relaying too (classify).
func relays(kind string) bool {
	return strings.HasPrefix(kind, "Node") || kind == "Pool node" || kind == "Disguised"
}

func kindRulesOf(lines []client) []kindRule {
	var out []kindRule
	for _, l := range lines {
		if l.kind != "" {
			out = append(out, kindRule{l.kind, l.re, relays(l.kind)})
		}
	}
	return out
}

func aboutRulesOf(lines []client) []aboutRule {
	var out []aboutRule
	for _, l := range lines {
		if l.text != "" {
			out = append(out, aboutRule{l.re, l.text, l.label, l.name})
		}
	}
	return out
}

// clients is every piece of software Peer Lab knows by its user agent, one
// line each: the pattern, the kind it gives (the ⊘ and the fork check read
// that), and the sentence, short label and name the panel shows. A line can
// carry only a kind - the generic words a crawler uses - or only a sentence.
//
// Read twice, from the top: the kind is the first line with a kind that
// matches (classify in kinds.go), the sentence the first line with a
// sentence that matches (ruleOf in about.go). Order matters within each: a
// misspelt Core name and a pool are asked first, and a line shares kind and
// sentence only when its pattern is tied to the start of the name - the
// broad words ("wasabi", "crawl") give their kind at the end, in their old
// order.
//
// A plain sentence says what the software is and has its Source line right
// above (about_sources_test.go); "Probably" and "Named like" say how
// sure the rest is - see README's "How sure a label is".
var clients = []client{
	// Asked first, before anything else reads the name: Core's name misspelt,
	// and pool software or a pool named in the brackets.
	// Bitcoin Core's name, misspelt: "/Sat0shi:" with a zero, "/Satoshi2:",
	// "/SatoshiX:", or "Satoshi:22.0.0" without the slashes Core always sends.
	// Real Core writes exactly "/Satoshi:x.y.z/". On the node this was written
	// for, "/Sat0shi:31.0.0/" alone came 475 times in a fortnight. And
	// "/Satoshi:0.21.0" without the closing slash: Core always adds it
	// (FormatSubVersion in src/clientversion.cpp, ss << "/"). Once in 114,020
	// "/Satoshi:" sessions on that node, 16.10.-19.11.2026, inbound, no
	// services; every real Core and Knots there ended in "/". First, so
	// nothing below reads it as Core. What it does is unknown, so it is not
	// marked as unable to relay.
	{re: regexp.MustCompile(`^/Sat0shi:`), kind: "Disguised"},
	{re: regexp.MustCompile(`^/Satoshi[0-9X]+:`), kind: "Disguised"},
	{re: regexp.MustCompile(`^Satoshi:`), kind: "Disguised"},
	{re: regexp.MustCompile(`^/Satoshi:[^/]*$`), kind: "Disguised"},
	// Mining pool software speaking the p2p protocol, or a node whose operator
	// says in the comment part of the agent - the bit in brackets - that it
	// belongs to a pool, e.g. "/Satoshi:29.1.0(PyBLOCK-POOL)/Knots:20250903/".
	// Only the brackets count: outside them "pool" is also in "mempool".
	// "mempool" inside the brackets does not count: "(mempool.guide)" is a
	// node's own label, not a pool (seen on the node this was written for).
	// /CKCoinD:/ is ckcoind, ckolivas' mining fork of Core (UA_NAME in
	// src/clientversion.cpp in ckolivas/ckcoind). /ckp2p:/ is ckpool's own block relay: ckp2p.c on
	// the p2pcrawler and nodrop branches of github.com/ckolivas/ckpool (not
	// on master, which is why it was once taken for someone else's).
	// P2Pool's bitcoind connection says /P2Pool:x/ (p2pool/bitcoin/p2p.py), and
	// a name that is itself a pool - /pydroid-pool:3/ - is one too.
	// miningcore-btc-solo is a solo pool built on Miningcore: AppInfo.cs in
	// bugoao/miningcore-btc-solo sets P2pUserAgent "/miningcore-btc-solo:<v>/",
	// P2p/P2pFastPeer.cs sends it (17.11.2026).
	{re: regexp.MustCompile(`(?i)ckpool`), kind: "Pool node"},
	{re: regexp.MustCompile(`(?i)^/ckp2p:`), kind: "Pool node"},
	{re: regexp.MustCompile(`(?i)^/CKCoinD:`), kind: "Pool node"},
	{re: regexp.MustCompile(`(?i)\([^)]*pool[^)]*\)`), kind: "Pool node"},
	{re: regexp.MustCompile(`(?i)^/P2Pool:`), kind: "Pool node"},
	{re: regexp.MustCompile(`(?i)^/[^:/(]*pool[^:/(]*:`), kind: "Pool node"},
	{re: regexp.MustCompile(`(?i)^/miningcore-btc-solo:`), kind: "Pool node"},
	// sipa's seeder sends "/bitcoin-seeder:0.01/" (bitcoin.cpp in
	// sipa/bitcoin-seeder, L97 at 92d2b46). Any other name with "seeder" in it
	// only says so.
	// Source: sipa/bitcoin-seeder bitcoin.cpp L97 @91e021f (checked 2026-11-24)
	{re: regexp.MustCompile(`^/bitcoin-seeder:`), text: "A DNS seeder: collects addresses of reachable nodes for new nodes to start from", label: "DNS seeder"},
	// Source: LunnaBoo/Bitcoin-Seeder-in-C src/protocol.h L14 @c334247 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/boo:`), kind: "Crawler", text: "Bitcoin Seeder in C, a learning project rebuilding sipa's seeder: collects addresses of reachable nodes", label: "Seeder in C (boo)"},
	// generic-seeder: sAppName "generic-seeder" (main.cpp L23), sent as
	// "/generic-seeder/" (bitcoin.cpp L80) in team-exor/generic-seeder, a seeder
	// for Bitcoin and coins cloned from it (checked 23.11.2026).
	// Source: team-exor/generic-seeder main.cpp L23 @6cf84cd (checked 2026-11-24)
	{re: regexp.MustCompile(`^/generic-seeder/`), text: "generic-seeder, a DNS seeder for Bitcoin and coins cloned from it", label: "generic-seeder"},
	// ua = b"\x10/mrm-census:0.2/" (crawler.py L40 in toomanymao/mrm-node-census,
	// commit 230796b of 26.07.2026, checked 23.11.2026).
	// Source: toomanymao/mrm-node-census crawler.py L40 @230796b (checked 2026-11-24)
	{re: regexp.MustCompile(`^/mrm-census:`), text: "mrm-node-census, a small crawler that maps reachable nodes, onion ones too", label: "mrm-node-census crawler"},
	{re: regexp.MustCompile(`(?i)seeder|dnsseed`), text: "Named like a DNS seeder, which collects addresses of reachable nodes for new nodes to start from", label: "Named like a DNS seeder"},
	// BTC Nodes (btcnodes.io), Rodrigo Martínez's successor to Bitnodes
	// (bitcointalk.org topic 5582226). Its public crawler config still says
	// /bitnodes.io:0.3/; that "/BTC-Nodes:<date>/Sonar/", ".../Probe/" and
	// ".../Delegator/" are its own was not found in a source, so only the name.
	// Before the census rule, which would take Sonar.
	{re: regexp.MustCompile(`^/BTC-Nodes:`), text: "Named like BTC Nodes (btcnodes.io), the successor to Bitnodes that maps reachable nodes", label: "Named like BTC Nodes"},
	// 21 Data Lab's own crawler speaks Bitcoin P2P and asks nodes for their
	// peer lists (21datalab.com/de/methodik); its user agent is not given
	// there, so only the name (11.11.2026).
	{re: regexp.MustCompile(`^/21datalab-crawler:`), text: "Named like 21 Data Lab's crawler, which counts reachable Bitcoin nodes (21datalab.com)", label: "Named like 21 Data Lab"},
	// Nebula, Dennis Trautwein's network crawler (ProbeLab), crawls Bitcoin
	// since 2.4.0 (github.com/dennis-tra/nebula/releases). Who runs a given
	// crawl it does not say. "nebula/2.4.1-…" and "/nebula:v3.0.0-…/".
	// Source: dennis-tra/nebula bitcoin/crawler.go L345 @2ace7f4 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/?nebula[/:]`), text: "Nebula, a network crawler by Dennis Trautwein (ProbeLab) that also maps Bitcoin nodes", label: "Nebula crawler"},
	// A Knots node under its operator's own name, offering service bits 27
	// and 28. Bit 27 is BIP110's (src/protocol.h at knots20260508), which
	// Bitcoin Purity sets too; bit 28, NODE_BLAKE2B, is only on Knots'
	// 29.x-knots branch (src/protocol.h at 5811235), not in knots20260508.
	{re: regexp.MustCompile(`(?i)^/GrayersDad:`), kind: "Node (Knots)", text: "Probably Bitcoin Knots under a name its operator chose", label: "Probably Knots"},
	// Before the census rule, which would take it for "monitor" (koyn-sync).
	// bitcoinj's own example sends the same: peerGroup.setUserAgent(
	// "PeerMonitor", "1.0") (examples/.../PeerMonitor.java L69 at v0.14.4).
	// Source: elkrem/koyn-sync src/main/java/io/elkrem/koynsync/Main.java L257 @c9b9ef1 (checked 2026-11-24)
	{re: regexp.MustCompile(`/PeerMonitor:`), text: "Koyn's header tool, which collects block headers for the Koyn Arduino library, or bitcoinj's PeerMonitor example of the same name", label: "Koyn/bitcoinj PeerMonitor"},
	// Census and seed monitors whose names hold one of the census words, each
	// with its source (14.11.2026): USER_AGENT in lib/observatory.rb
	// (azuchi/btc-node-observatory), OUR_USER_AGENT in dns_monitor/handshake.py
	// (ndungudedan/dns-monitor, and willcl-ark's fork), user_agent in
	// config.toml (echennells/dns-seed-monitor).
	// Source: azuchi/btc-node-observatory lib/observatory.rb L12 @36b19bf (checked 2026-11-24)
	{re: regexp.MustCompile(`^/btc-node-observatory:`), text: "Bitcoin Node Observatory, Shigeyuki Azuchi's daily census of reachable nodes", label: "Node Observatory census"},
	// Source: ndungudedan/dns-monitor dns_monitor/handshake.py L35 @e293a73 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/dns-monitor:`), text: "dns-monitor, a tool that checks the nodes Bitcoin's DNS seeds hand out", label: "DNS seed monitor"},
	// user_agent = "/DCTRL-YVR-seed-monitor:0.1.0/" (config.toml L14 in
	// echennells/dns-seed-monitor, 23.11.2026).
	// Source: echennells/dns-seed-monitor config.toml L14 @28bc2e7 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/DCTRL-YVR-seed-monitor:`), text: "DCTRL YVR's DNS seed monitor: checks the nodes Bitcoin's DNS seeds hand out", label: "DCTRL seed monitor"},
	// Only the word: no source for any of these names.
	{re: regexp.MustCompile(`(?i)census|survey|observatory|monitor|sonar`), text: "Named like a node census, which counts and watches reachable nodes", label: "Named like a node census"},
	// One address, 142 sessions in four days, each gone after its version
	// message. Elastum was an exchange; elastum.io no longer resolves.
	{re: regexp.MustCompile(`(?i)/Elastum:`), text: "Named like Elastum, a former exchange; talks to nodes through bitcoinj", label: "Named like Elastum"},
	// 26-27.10.2026: one address, five sessions in 40 minutes under protocol
	// 70001 with no services, saying our node is 127.0.0.1:8332. Two ended after
	// the version message; three took 7 to 16 MB of blocks each.
	// That is btc-rpc-proxy fetching a block the node lacks from a peer: it
	// sends "BTC RPC Proxy v<version>" (src/fetch_blocks.rs L50 in
	// Start9Labs/btc-rpc-proxy, 0.5 to 0.8.x, the fork StartOS runs; L49 in
	// Martin Habovštiak's Kixunil/btc-rpc-proxy up to 0.4.1; 15.11.2026).
	// Any other form was found nowhere and only names itself.
	// Source: Kixunil/btc-rpc-proxy src/fetch_blocks.rs L49 @dc95a40 (checked 2026-11-24)
	{re: regexp.MustCompile(`^BTC RPC Proxy v[0-9]`), text: "BTC RPC Proxy, which guards Bitcoin Core's RPC and fetches blocks the node lacks from peers", label: "BTC RPC Proxy"},
	{re: regexp.MustCompile(`^BTC RPC Proxy `), text: "Named like an RPC proxy for Bitcoin; no public source", label: "Named like an RPC proxy"},
	// Bisq's userAgent option defaults to "Bisq" (common/.../config/Config.java
	// in bisq-network/bisq), handed to bitcoinj in WalletsSetup.java.
	// Source: bisq-network/bisq common/src/main/java/bisq/common/config/Config.java L753 @bdb202b (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)/Bisq:`), text: "Bisq, a decentralised exchange; its wallet runs on bitcoinj", label: "Bisq exchange wallet"},
	// Source: forrestv/p2pool p2pool/bitcoin/p2p.py L35 @53c438b (checked 2026-11-24)
	{re: regexp.MustCompile(`^/P2Pool:`), text: "P2Pool, a peer-to-peer mining pool: miners share the work without a pool operator", label: "P2Pool mining pool"},
	// node-stratum-pool (zone117x) listens for new blocks over P2P, offering
	// NODE_NETWORK alone, and hands found blocks to its node by RPC.
	// Source: zone117x/node-stratum-pool lib/peer.js L64 @d1e71da (checked 2026-11-24)
	{re: regexp.MustCompile(`^/node-stratum/`), kind: "Pool node", text: "node-stratum-pool, Node.js mining pool software listening for new blocks", label: "node-stratum-pool"},
	// BitcoinRocks uses RocksDB and compressed block storage, retaining
	// Bitcoin consensus (UA_NAME and CMakeLists.txt in BitcoinRocks-Core/BitcoinRocks).

	// Source: BitcoinRocks-Core/BitcoinRocks CMakeLists.txt L29 @94ad5e5 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/BitcoinRocks:`), kind: "Node (BitcoinRocks)", text: "BitcoinRocks, an experimental fork of Core using RocksDB and compressed block storage", label: "BitcoinRocks, a Core fork"},
	// alexar76/symmartys (formerly emporion): symmartys/bitcoin_headers.py
	// sends "/SYMMARTYS:0.3/"; EMPORION is the older name, still in
	// emporion/bitcoin_headers.py at commit 922e4bc (17.10.2026).
	// Source: alexar76/emporion emporion/bitcoin_headers.py L46 @v0.4.0 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/EMPORION:`), kind: "Other light client", text: "EMPORION, a Bitcoin header client for verifying OpenTimestamps proofs; later renamed SYMMARTYS", label: "EMPORION header client"},
	// const ua = Buffer.from('/wuji-headers:0.1/') in indexer/bitcoin-p2p.mjs
	// (MM-sheng/wuji, 15.11.2026).
	// Source: MM-sheng/wuji indexer/bitcoin-p2p.mjs L317 @7ebbb42 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/wuji-headers:`), kind: "Other light client", text: "WUJI's indexer: follows Bitcoin's block headers to build an index usable on Ethereum", label: "WUJI header client"},
	// const USER_AGENT in sdk/rust/pcoc-gossip/src/wire.rs in groundinglab/pcoc.
	// Source: groundinglab/pcoc sdk/rust/pcoc-gossip/src/wire.rs L17 @9c07a7f (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/pcoc-bari:`), kind: "Other light client", text: "pcoc Bari, a timestamping service that follows Bitcoin's block headers", label: "pcoc Bari header client"},
	// UA_NAME("Prometheus") in src/clientversion.cpp, version 30.4.1 in
	// CMakeLists.txt (CryptoIceMLH/bitcoin-prometheus, 15.11.2026).
	// Source: CryptoIceMLH/bitcoin-prometheus src/clientversion.cpp L23 @d1b302b (checked 2026-11-24)
	{re: regexp.MustCompile(`^/Prometheus:`), text: "BTC-Prometheus, a full node forked from Bitcoin Core", label: "BTC-Prometheus node"},
	// Source: alexar76/symmartys symmartys/bitcoin_headers.py L46 @b70b1eb (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/SYMMARTYS:`), kind: "Other light client", text: "SYMMARTYS, formerly EMPORION: follows Bitcoin headers to verify OpenTimestamps proofs", label: "SYMMARTYS header client"},
	// Source: nyonson/bitcoin-peers connection/src/connection/configuration.rs L15 @ce17e9f (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/bitcoin-peers:`), kind: "Crawler", text: "bitcoin-peers, a Rust toolkit to connect to and survey Bitcoin peers", label: "bitcoin-peers crawler"},
	// Source: logos-co/logos-bitcoin-peer-list crawler/src/p2p.rs L21 @29e73f8 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/logos-peer-list:`), kind: "Crawler", text: "Logos peer list, a crawler that checks onion peers for the Logos wallet", label: "Logos peer list"},
	// Source: chjj/mako include/mako/net.h L116 @41ef104 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/mako:`), text: "mako, a Bitcoin full node written in C", label: "mako node (C)"},
	// let ua = b"/bip110-crawler:0.1.0/" (src/p2p.rs L421 in
	// donaldevine/bip110-crawler at f412989, checked 23.11.2026).
	// Source: donaldevine/bip110-crawler src/p2p.rs L421 @f412989 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/bip110-crawler:`), text: "bip110-crawler, a crawler that checks which peers follow BIP-110", label: "bip110-crawler"},
	// Castor, Floresta's Utreexo bridge, answers with "/bridge:0.1.3/" (src/node.rs
	// L1541 in getfloresta/castor at 6713d5b, checked 23.11.2026).
	// Source: getfloresta/castor src/node.rs L1541 @eba7b22 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/bridge:`), text: "Castor, Floresta's Utreexo bridge: serves Utreexo proofs to other nodes", label: "Castor Utreexo bridge"},
	// Source: decentralizedb/bip110-observer app/crawler.py L134 @588e0a8 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/bip110-observer:`), kind: "Crawler", text: "BIP-110 Observer, a crawler that collects peer user agents and advertised services", label: "BIP-110 Observer crawler"},
	// Only the name points to favillakey.io; the site documents no P2P
	// connections, so the kind stays Other.
	{re: regexp.MustCompile(`^/FavillaKEY:`), text: "Named like Favilla KEY, a Nostr and Bitcoin signing device (favillakey.io); what it connects for is not documented", label: "Named like Favilla KEY"},
	// Bitcoin Ghost (github.com/bitcoin-ghost/ghost): ghostd, a fork of Core
	// v30, UA_NAME "Ghost" in clientversion.cpp.

	// Source: bitcoin-ghost/ghost ghost-core/src/clientversion.cpp L24 @8248dbf (checked 2026-11-24)
	{re: regexp.MustCompile(`^/Ghost:`), kind: "Node (Ghost)", text: "Bitcoin Ghost, a fork of Core v30 that filters data-carrying transactions; its operators can join a decentralised mining pool", label: "Bitcoin Ghost (Core fork)"},
	// Source: hashhog/camlcoin lib/peer.ml L1019 @482c676 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/CamlCoin:`), text: "CamlCoin, a Bitcoin full node written in OCaml (hashhog)", label: "CamlCoin node (OCaml)"},
	// Source: hashhog/rustoshi crates/network/src/peer.rs L1815 @f9969a1 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/rustoshi:`), text: "rustoshi, a Bitcoin full node written in Rust (hashhog)", label: "rustoshi node (Rust)"},
	// Source: hashhog/lunarblock src/peer.lua L877 @84f5cfd (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/LunarBlock:`), text: "LunarBlock, a Bitcoin full node written in Lua (hashhog)", label: "LunarBlock node (Lua)"},
	// Source: hashhog/haskoin src/Haskoin/Network.hs L893 @e4f6b9e (checked 2026-11-24)
	{re: regexp.MustCompile(`^/Haskoin:`), text: "A Bitcoin full node written in Haskell (hashhog)", label: "Haskoin node (Haskell)"},
	// Source: hashhog/beamchain src/beamchain_peer.erl L1745 @18729c8 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/beamchain:`), text: "beamchain, a Bitcoin full node written in Erlang (hashhog)", label: "beamchain node (Erlang)"},
	// Source: hashhog/blockbrew cmd/blockbrew/main.go L1692 @a76b27f (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/blockbrew:`), text: "blockbrew, a Bitcoin full node written in Go (hashhog)", label: "blockbrew node (Go)"},
	// Source: hashhog/clearbit src/p2p.zig L46 @6510c80 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/clearbit:`), text: "clearbit, a Bitcoin full node written in Zig (hashhog)", label: "clearbit node (Zig)"},
	// Source: hashhog/nimrod src/network/messages.nim L23 @240a00d (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/nimrod:`), text: "nimrod, a Bitcoin full node written in Nim (hashhog)", label: "nimrod node (Nim)"},
	// Source: hashhog/hotbuns src/consensus/params.ts L549 @0cfeed4 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/hotbuns:`), text: "hotbuns, a Bitcoin full node written in TypeScript on Bun (hashhog)", label: "hotbuns node (TypeScript)"},
	// Source: hashhog/ouroboros src/ouroboros/p2p_messages.py L270 @1264ed2 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/(bitcoin-hybrid|ouroboros):`), text: "ouroboros, a Bitcoin full node written in Rust and Python (hashhog)", label: "ouroboros node (Rust)"},
	// crates/rbitcoin-primitives/src/lib.rs in rbitcoin-org/rbitcoin (moved
	// from reardencode).
	// Source: rbitcoin-org/rbitcoin crates/rbitcoin-primitives/src/lib.rs L82-89 @5d2ec82 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/rbitcoin:`), text: "rbitcoin, a Bitcoin full node written in Rust (reardencode)", label: "rbitcoin node (Rust)"},
	// Source: AmazingAng/rbtc crates/rbtc-net/src/message.rs L282 @4f30f67 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/rbtc:`), text: "rbtc, Bitcoin Core rewritten in Rust", label: "rbtc, Core in Rust"},
	// The original repository, bitcoinecho/bitcoin-echo, is gone (404,
	// 14.11.2026); a kept copy sets the name.
	// Source: jesterhodl/bitcoin-echo src/protocol/peer.c L31 @9196919 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/BitcoinEcho:`), text: "Bitcoin Echo, a full node in plain C, meant to be frozen as it is", label: "Bitcoin Echo node (C)"},
	// Source: BobClawblaw/bitcoinmachinecode asm/version.inc L33 @98d1276 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/BitcoinMachineCode:`), text: "Bitcoin Machine Code, a node written in x86 assembly, entirely AI-generated", label: "Bitcoin Machine Code node"},
	// Source: brunoerg/exnode lib/exnode/node.ex L306 @241d057 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/exnode:`), text: "exnode, a Bitcoin full node written in Elixir that checks scripts with Bitcoin Core's kernel library", label: "exnode (Elixir node)"},
	// Comb Downloader NG, the Haircomb wallet's block fetcher, says
	// "/Natasha:<its version>/" only with -natasha ("Useful for servers",
	// main.go L63); without it, "/Satoshi:29.0.0/" (util.go L8-9). Its source
	// is on Codeberg, read here from the Go module proxy's copy.
	// Source: codeberg.org/watashi564/combdownloader-ng main.go L81-82 @698cc05 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/Natasha:`), text: "Comb Downloader NG in server mode: fetches Bitcoin blocks and headers for the Haircomb wallet and passes them on", label: "Comb Downloader (Natasha)"},
	// Source: kernel-node/kernel-node src/net/peer.rs L470 @ac5ad5c (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/kernel-node:`), text: "kernel-node, a node built on libbitcoinkernel, Core's validation engine as a library", label: "kernel-node (Core kernel)"},
	// Source: yuvicc/bitcoinkernel-node src/node/p2p.cpp L104 @0d3d474 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/bitcoinkernel-node:`), text: "bitcoinkernel-node, a small full node built on libbitcoinkernel, Core's validation engine as a library", label: "bitcoinkernel-node"},
	// satd in crawler mode says so only in brackets: a rule that asks
	// for a bracket itself reads it (classify).

	// satd's crawl sessions say "/satd:<version>(crawler)/" (user_agent in
	// node/src/net/crawl.rs in epochbtc/satd at c3be00a, checked 23.11.2026).
	// Source: epochbtc/satd node/src/net/crawl.rs L226 @e0e8c28 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/satd:[^/(]*\(crawler`), kind: "Crawler", text: "satd in crawler mode: maps reachable nodes", label: "satd crawler"},
	// Source: epochbtc/satd node/src/lib.rs L38 @e0e8c28 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/satd:`), text: "satd, a Bitcoin full node written in Rust that works like Core", label: "satd node (Rust)"},
	// Source: hemilabs/heminetwork cmd/tbcd/tbcd.go L283 @c76e7e7 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/tbcd:`), text: "tbcd, the Bitcoin daemon of the Hemi network", label: "tbcd, Hemi's daemon"},
	// "soak" is nowhere in hemilabs/heminetwork. Hemi's developers describe a
	// "/btcwire:0.5.0/hemi-soak:1.0/ fleet peer" that served a partner's node a
	// fork of ~2,100 headers with no work (PR #1174 in hemilabs/heminetwork,
	// merged 16.09.2026; bnoc.xyz/t/170), but do not say who runs it.
	{re: regexp.MustCompile(`(?i)hemi-soak`), text: "Named like a Hemi soak test; Hemi's developers saw one serve their node a fork without work", label: "Named like hemi-soak"},
	// paulscode's release of 28.10.2026, Knots carried on after it hard-forked
	// to BLAKE2b: src/clientversion.cpp at tag v31.1.knots-sha256.20261018 in
	// paulscode/knots-sha256 (not on master).
	// Oracle Knots (MarcanoFilms/oracle-knots, src/clientversion.cpp at
	// 5486108): "OracleKnots:" after its own name, on the BLAKE2b chain whose
	// first block is 961640 (src/kernel/chainparams.cpp). With -uaprivacy it
	// sends Knots' plain string; then only service bit 28 tells (forks.go).
	// Source: MarcanoFilms/oracle-knots src/clientversion.cpp L74 @ae2f88d (checked 2026-11-24)
	{re: regexp.MustCompile(`OracleKnots:`), text: "Oracle Knots, a Knots fork on the BLAKE2b chain (hard fork at block 961,640)", label: "Oracle Knots (BLAKE2b)"},
	// The standalone BIP-110 client (dathonohm/bitcoin, src/clientversion.cpp at
	// 78dfa44): ua += "UASF-BIP110:0.4/". Its rules split off at 961632.
	// Source: dathonohm/bitcoin src/clientversion.cpp L77 @78dfa44 (checked 2026-11-24)
	{re: regexp.MustCompile(`UASF-BIP110:`), text: "Knots with the BIP-110 UASF, on the chain that split off at block 961,632", label: "BIP-110 UASF chain"},
	// Two forks of 2017 on Core's name: the BIP148 UASF client, "UASF-Segwit:1.0
	// (BIP148)/" (src/clientversion.cpp in UASF/bitcoin, branch 0.14-BIP148),
	// and btc1, SegWit2x, with the comment "2x" (src/init.cpp in btc1/bitcoin).
	// Source: UASF/bitcoin src/clientversion.cpp L106 @v0.14.2-uasfsegwit1.0 (checked 2026-11-24)
	{re: regexp.MustCompile(`UASF-Segwit:`), text: "The BIP148 UASF client of 2017, which enforced SegWit", label: "BIP148 UASF (2017)"},
	// Source: btc1/bitcoin src/init.cpp L1239 @v1.14.5 (checked 2026-11-24)
	{re: regexp.MustCompile(`\(2x\)`), text: "btc1, the SegWit2x client of 2017; its fork was called off", label: "btc1 SegWit2x (2017)"},
	// Source: paulscode/knots-sha256 src/clientversion.cpp L74 @e9c534b (checked 2026-11-24)
	{re: regexp.MustCompile(`/KnotsSHA256:`), text: "Knots (SHA256), paulscode's build of Knots that stays on the SHA256 chain", label: "Knots (SHA256), paulscode"},
	// Plumb (plumb-node/plumb): one Knots release plus the spam filters its
	// maintainers reviewed; policy only, so it follows that release's chain
	// (README). "/Plumb:15/" after Knots' own part; its kind stays Knots.
	// Source: plumb-node/plumb src/clientversion.cpp L76 @a583168 (checked 2026-11-24)
	{re: regexp.MustCompile(`/Plumb:`), text: "Plumb, Bitcoin Knots with extra spam-policy filters on top", label: "Plumb (Knots fork)"},
	// Knots Legacy (DimiH2025/KnotsLegacy) keeps Knots 29.3.0 and its own date:
	// CLIENT_VERSION_SUFFIX ".knots20260920" (tag V29.3.0.20260920) and
	// ".knots20261124" (tag V29.3.0.20261124). Knots itself tagged 29.3 only
	// with 20260210, 20260507 and 20260508. Anyone can build that string, so
	// "Probably". Its kind stays Knots.
	// Source: DimiH2025/KnotsLegacy CMakeLists.txt L29-37 @2a38019 (checked 2026-11-25)
	{re: regexp.MustCompile(`^/Satoshi:29\.3\.0(\([^)]*\))?/Knots:(20260920|20261124)/`), text: "Probably Knots Legacy, a fork that stays on Knots 29.3: Knots itself never released 29.3 with this date", label: "Probably Knots Legacy"},
	// spruned's copy of pycoinnet: subversion=b"/pycoinnet/" in
	// spruned/dependencies/pycoinnet/version.py L81, used unchanged by
	// spruned/daemon/bitcoin_p2p/p2p_connection.py L114 (mempoolco/spruned,
	// checked 23.11.2026). pycoinnet itself says "/Notoshi/" (below).
	// Source: mempoolco/spruned spruned/dependencies/pycoinnet/version.py L81 @630aad3 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/pycoinnet/`), text: "spruned, a Bitcoin client without a blockchain of its own: it fetches any block from peers when asked", label: "spruned (no own chain)"},
	// APP_NAME "blueberry", APP_VERSION "2026.08.17" (peers/.../net/UserAgent.kt
	// in BlueWallet/Blueberry.kmp), sent as "/${name}:${version}/"
	// (session/VersionHandshake.kt in GladosBlueWallet/bip324.kmp).
	// Source: BlueWallet/Blueberry.kmp peers/src/commonMain/kotlin/io/bluewallet/blueberry/peers/net/UserAgent.kt L3 @c8071fc (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/blueberry:`), kind: "Wallet", text: "Blueberry, BlueWallet's new wallet that talks to nodes directly", label: "Blueberry (BlueWallet)"},
	// Labels a node's setup puts in brackets after Core's name. Apollo's own
	// bitcoin.conf sets it (jstefanop/apolloapi-v2, backend/default-configs).
	// Source: jstefanop/apolloapi-v2 backend/default-configs/bitcoin.conf L5 @c33a0be (checked 2026-11-24)
	{re: regexp.MustCompile(`\(FutureBit-Apollo-Node\)`), text: "A node on FutureBit Apollo, a home Bitcoin miner with a built-in full node", label: "FutureBit Apollo node"},
	// docker/my-dojo/bitcoin/restart.sh: -uacomment="Samourai Dojo $DOJO_VERSION_TAG"
	// (the original at code.samourai.io is offline; Umbrel's copy,
	// getumbrel/umbrel-samourai-dojo, has it).
	// Source: getumbrel/umbrel-samourai-dojo docker/my-dojo/bitcoin/restart.sh L16 @f4a417e (checked 2026-11-24)
	{re: regexp.MustCompile(`\(Samourai Dojo `), text: "A node run by Samourai Dojo, the self-hosted backend for Samourai Wallet and other light wallets", label: "Samourai Dojo node"},
	// docker/my-dojo/bitcoin/restart.sh in git.ronindojo.io/Ronin/dojo:
	// -uacomment="RoninDojo $RONINDOJO_VERSION_TAG" (v2.4.1; older tags use
	// $DOJO_VERSION_TAG, and an empty tag leaves "(RoninDojo )").
	// Source: git.ronindojo.io/Ronin/dojo docker/my-dojo/bitcoin/restart.sh L19 @69136f3 (checked 2026-11-24)
	{re: regexp.MustCompile(`\(RoninDojo `), text: "A node run by RoninDojo, an installer and UI for Samourai Wallet's Dojo", label: "RoninDojo node"},
	// The bitcoin.conf in MiniBolt's Core and Knots pages: uacomment=MiniBolt node.
	// Source: minibolt-guide/minibolt bitcoin/bitcoin/bitcoin-client.md L311 @fd980f0 (checked 2026-11-24)
	// Source: minibolt-guide/minibolt bitcoin/bitcoin/btc-rpc-explorer.md L412 @53f972e (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)\((Official )?MiniBolt[ _]node\)|; MiniBolt node\)`), text: "A node set up with MiniBolt, a step-by-step guide to building a Bitcoin node", label: "MiniBolt guide node"},
	// mempool's own node config carries "#uacomment=@wiz" (production/
	// bitcoin.conf L19 at v3.3.2 in mempool/mempool; wiz co-founded
	// mempool.space). Anyone can copy it, and mempool.space lists no nodes of
	// its own, so only "Probably".
	{re: regexp.MustCompile(`(?i)^/Satoshi:[^/]*\(@wiz\)/(Knots:[^/]*/)?$`), text: "Probably a mempool.space node: '@wiz' is the comment in mempool's own node config", label: "Probably mempool.space"},
	// Both only on a bare Core or Knots name: anything after it (electrs)
	// says more. "(mempool.guide)": mempool.guide is Retropex's block explorer
	// (Retropex/mempool's homepage); no source sets this comment.
	{re: regexp.MustCompile(`(?i)^/Satoshi:[^/]*\(mempool\.guide\)/(Knots:[^/]*/)?$`), text: "Named like mempool.guide, Retropex's block explorer", label: "Named like mempool.guide"},
	// A 2017 Docker image of Bitcore set uacomment=bitcore (m00re/bitcore-docker);
	// these run newer Core, so the text goes by the name only.
	{re: regexp.MustCompile(`\(bitcore\)`), text: "Named like BitPay's Bitcore, an index server that runs on a node", label: "Named like Bitcore"},
	// userAgent = "/winnow:0.1/" (Sources/WalletCore/Network/Transport/
	// PeerConnection.swift in winnowwallet/winnow).
	// Source: winnowwallet/winnow Sources/WalletCore/Network/Transport/PeerConnection.swift L82 @162abb7 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/winnow:`), kind: "Wallet", text: "Winnow, an iPhone wallet that talks to nodes directly", label: "Winnow iPhone wallet"},
	// Source: horizontalsystems/bitcoin-kit-android bitcoincore/src/main/kotlin/io/horizontalsystems/bitcoincore/network/messages/VersionMessage.kt L18 @1f581d1 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/BitcoinKit:`), kind: "Wallet", text: "BitcoinKit, a wallet library (Unstoppable Wallet on Android and others)", label: "BitcoinKit wallet library"},
	// Source: horizontalsystems/BitcoinCore.Swift Sources/BitcoinCore/Classes/Network/Peer/Peer.swift L71 @5b49f42 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/WalletKit:`), kind: "Wallet", text: "Unstoppable Wallet on iPhone", label: "Unstoppable Wallet iOS"},
	// CLIENT_NAME("Ladybug") (src/clientversion.cpp in rkarpuzov/Ladybug-pupa):
	// the node of a coin of its own on Bitcoin Core, with Bitcoin's network
	// magic. Its Android wallet says "Ladybug Wallet" on bitcoinj instead.
	// Source: rkarpuzov/Ladybug-pupa src/clientversion.cpp L16 @8b8be89 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/Ladybug:`), kind: "Other chain", text: "Ladybug, an altcoin built on Bitcoin Core", label: "Ladybug altcoin"},
	// HODL Wallet puts its library's name first: "/Liviano:<version>/hodlwallet:2.0/"
	// (HodlWallet/Core/Services/WalletService.cs in hodlwallet/hodlwallet,
	// Liviano/Version.cs in hodlwallet/liviano).
	// Source: hodlwallet/hodlwallet HodlWallet/Core/Services/WalletService.cs L64 @f1f7284 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)/hodlwallet:`), text: "HODL Wallet, a phone wallet", label: "HODL Wallet"},
	// "Yuki / {YUKI_VERSION}" (src/network/outbound_messages.rs in imperviousinc/yuki).
	// Source: imperviousinc/yuki src/network/outbound_messages.rs L49 @d077223 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^Yuki /`), kind: "Other light client", text: "Yuki, a light client by Impervious built on Kyoto", label: "Yuki light client"},
	// Source: VerifiedXBlockchain/VerifiedX-Core VerifiedXCore/Bitcoin/Attestation/Net/BtcSources.cs L100 @e814c14 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/VerifiedX:`), kind: "Other light client", text: "A VerifiedX validator: a sidechain that holds bitcoin and follows Bitcoin's chain for it", label: "VerifiedX validator"},
	// format!("/LionSeed:{}/", ...) (src/crawl.rs in LionThunderFingers/LionSeed),
	// "serves only verified fork nodes" by its description.
	// Source: LionThunderFingers/LionSeed src/crawl.rs L38 @9d75a7b (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/LionSeed:`), text: "LionSeed, a DNS seeder that lists only the nodes of one fork", label: "LionSeed DNS seeder"},
	// gnoban connects to peers itself, but under python-bitcoinlib's name: its
	// BitcoinMsgver() keeps the library's subversion (gnoban.py L640-669 in
	// caesrcd/gnoban, 23.11.2026), so a peer saying "gnoban" only borrows it.
	{re: regexp.MustCompile(`(?i)^/gnoban:`), text: "Named like gnoban, a tool that scores nodes and bans the ones its operator does not want", label: "Named like gnoban"},
	// "libbtc 0.1", no slashes (src/net.c in libbtc/libbtc).
	// Source: libbtc/libbtc src/net.c L293 @bd386ee (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^libbtc `), kind: "Other light client", text: "libbtc, a small C library for Bitcoin wallets and tools", label: "libbtc C library"},
	// "/haskoin-btc:<version>/" (src/Haskoin/Network/Constants.hs in
	// jprupp/haskoin-core, formerly haskoin/haskoin-core); haskoin-node, which Haskoin Store runs on, sends it.
	// Source: jprupp/haskoin-core src/Haskoin/Network/Constants.hs L91 @240283f (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/haskoin-btc:`), kind: "Other light client", text: "Haskoin, a Haskell library for Bitcoin wallets and tools", label: "Haskoin library (Haskell)"},
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
	{re: regexp.MustCompile(`(?i)^/btcwire:[^/]*/bitfyn:`), kind: "Wallet", text: "bitfyn, a desktop wallet that follows the chain by compact block filters", label: "bitfyn wallet"},
	// Source: pin2t/twotbit-macos Core/Sources/TwotbitKit/P2P/PeerConnection.swift L32 @19e2e00 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/twotbit:`), kind: "Wallet", text: "Twotbit, a desktop wallet that follows the chain by compact block filters", label: "Twotbit wallet"},
	// Source: mx4/bitc core/bitc-defs.h L9 @b8f9d4f (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/bitc:`), kind: "Wallet", text: "bitc, a small Bitcoin wallet written in C", label: "bitc wallet (C)"},
	// Source: bitcoindevkit/bdk src/blockchain/compact_filters/peer.rs L114 @77c95b9 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^MagicalBitcoinWallet$`), kind: "Wallet", text: "BDK, the Bitcoin Dev Kit wallet library, in its old compact filter backend", label: "BDK (old filter backend)"},
	// Source: ciphrex/mSIGNA src/networksync.cpp L358 @b79c55e (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^Wallet v0\.1$`), kind: "Wallet", text: "mSIGNA, a multisig desktop wallet", label: "mSIGNA wallet"},
	// Source: keeshux/bitcoinspv BitcoinSPV/Sources/Global/WSConfig.m L30 @53ec533 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/BitcoinSPV:`), kind: "Wallet", text: "BitcoinSPV, a light wallet library for iPhone and Mac", label: "BitcoinSPV library"},
	// Source: richardkiss/pycoinnet pycoinnet/helpers/standards.py L64 @7c354cb (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/Notoshi/$`), kind: "Other light client", text: "pycoinnet, Richard Kiss's Python library for the Bitcoin protocol", label: "pycoinnet library"},
	// Source: sarchar/pyspv pyspv/__init__.py L29-32 @33395ae (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/pyspv:`), kind: "Other light client", text: "pyspv, a Python library for light wallets", label: "pyspv library"},
	// Source: brain-zhang/chainhorn chainhorn/contrib/core/__init__.py L23 @4db18fe (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/chainhorn:`), kind: "Other light client", text: "chainhorn, a Python light client", label: "chainhorn light client"},
	// Source: rust-bitcoin/murmel src/constructor.rs L61 @f8902dd (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/Murmel:`), kind: "Other light client", text: "Murmel, a Rust light client", label: "Murmel light client"},
	// Source: Cattens-Roses/cryptogate crates/networks/bitcoin/src/peer.rs L203 @b07eefd (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/enginecore:`), kind: "Wallet", text: "EngineCore, a terminal wallet for Bitcoin, Litecoin and Ethereum that follows Bitcoin by compact block filters", label: "EngineCore wallet"},
	// Source: mappum/bitcoin-net src/peer.js L66 @622a4eb (checked 2026-11-24)
	{re: regexp.MustCompile(`/bitcoin-net:`), text: "bitcoin-net, the P2P library of Webcoin, a JavaScript light client", label: "bitcoin-net (Webcoin)"},
	// Source: shadders/BitcoinWallet src/BitcoinWallet/Parameters.java L47 @99d423c (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/BitcoinWallet:`), kind: "Wallet", text: "BitcoinWallet, a Java desktop wallet by ScripterRon", label: "BitcoinWallet (Java)"},
	// Source: TheBlueMatt/rust-bitcoin-spv src/bin/main.rs L78-81 @a5bc31f (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/rust-spv:`), kind: "Other light client", text: "rust-bitcoin-spv, a Rust light client", label: "rust-bitcoin-spv"},
	// Source: DoubleSha/BitcoinSwift BitcoinSwift/PeerController.swift L70 @8756a45 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/BitcoinSwift:`), kind: "Other light client", text: "BitcoinSwift, a Swift library for the Bitcoin protocol", label: "BitcoinSwift library"},
	// josibake/sashimi exists ("a Bitcoin node as Unix tools", Zig), but no
	// "/sashimi:" is in its code (14.11.2026).
	{re: regexp.MustCompile(`(?i)^/sashimi:`), text: "Named like sashimi, small Unix tools for Bitcoin by josibake", label: "Named like sashimi"},
	// src/lib.rs in gnostr-org/gnostr-bitcoin: user_agent = "/rust-sha2-p2p-client/".
	// Source: gnostr-org/gnostr-bitcoin src/lib.rs L434 @71effa0 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/rust-sha2-p2p-client/`), text: "gnostr-bitcoin, a terminal monitor of the Bitcoin network from the gnostr project", label: "gnostr-bitcoin monitor"},
	{re: regexp.MustCompile(`(?i)^/Gnostr:`), text: "Named like gnostr, the git+nostr tools", label: "Named like gnostr"},
	// CLIENT_NAME("Aurum") (src/clientversion.cpp in aurumcoin-au/Aurumcoin-AU),
	// with Bitcoin's network magic.
	// Source: aurumcoin-au/Aurumcoin-AU src/clientversion.cpp L22 @96bad6d (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/Aurum:`), kind: "Other chain", text: "Aurumcoin, an altcoin built on Bitcoin Core 0.12", label: "Aurumcoin altcoin"},
	// Source: saltduck/bitcoinpurity src/clientversion.cpp L79 @124fbc8 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)/Purity:`), text: "Bitcoin Purity, a fork of Knots that made BIP110 permanent in a hard fork at block 961637", label: "Bitcoin Purity hard fork"},
	// Source: Flopcoin/Flopcoin src/clientversion.cpp L16 @3ad241b (checked 2026-11-25)
	{re: regexp.MustCompile(`(?i)^/Floptoshi:`), kind: "Other chain", text: "Flopcoin Core, the node of the Flopcoin altcoin", label: "Flopcoin altcoin"},
	// BTokenWPF talks to Bitcoin on port 8333 and to its own BToken chain.
	// Source: BusinessChain/BTokenWPF BTokenLib/Token/Network/NetworkToken.Connector.cs L21 @3b3f28e (checked 2026-11-25)
	{re: regexp.MustCompile(`(?i)^/BTokenCore:`), kind: "Wallet", text: "BTokenWPF by BusinessChain, a desktop wallet that also runs its own BToken chain", label: "BTokenWPF wallet"},
	// Schildbach's wallet was already at 4.57 on bitcoinj 0.14.3 and went
	// up from there, so "Bitcoin Wallet:1.0.x" on bitcoinj 0.14+ is not it.
	// Source: bitcoin-wallet/bitcoin-wallet wallet/build.gradle L13 @8cbe9f9 (checked 2026-11-25)
	{re: regexp.MustCompile(`^/bitcoinj:0\.(1[4-9]|[2-9][0-9])[^/]*/(Schildbach )?Bitcoin Wallet:1\.0[./]`), text: "Named like Schildbach's Bitcoin Wallet, but its version doesn't match", label: "Named like Bitcoin Wallet"},
	// BTCX's fork keeps the name "Bitcoin Wallet" and marks its version:
	// android:versionName="4.58.1-btcx", package se.btcx.wallet.
	// Source: BTCX/bitcoin-wallet wallet/AndroidManifest.xml L4-7 @5dc1697 (checked 2026-11-25)
	{re: regexp.MustCompile(`Bitcoin Wallet:[^/]*-btcx/`), text: "BTCX's Android wallet, a fork of Schildbach's Bitcoin Wallet", label: "BTCX wallet (fork)"},
	// Schildbach's own BlackBerry build, branch blackberry-prod:
	// android:versionName="5.21-blackberry".
	// Source: schildbach/bitcoin-wallet wallet/AndroidManifest.xml L4-7 @bc173d3 (checked 2026-11-25)
	{re: regexp.MustCompile(`Bitcoin Wallet:[^/]*-blackberry`), text: "Bitcoin Wallet by Schildbach, its BlackBerry build", label: "BlackBerry Bitcoin Wallet"},
	// USER_AGENT = "Bitcoin Wallet" (wallet/src/de/schildbach/wallet/Constants.java
	// in bitcoin-wallet/bitcoin-wallet), on bitcoinj.
	// Source: bitcoin-wallet/bitcoin-wallet wallet/src/de/schildbach/wallet/Constants.java L127 @e1c558c (checked 2026-11-24)
	{re: regexp.MustCompile(`Bitcoin Wallet:`), text: "Bitcoin Wallet by Schildbach, an Android app", label: "Bitcoin Wallet (Android)"},
	// breadwallet-core sends "/bread:" with its own version, 2.1 since 2019
	// (BRPeer.h), also in BRD's later WalletKit (blockset-corp/walletkit).
	// Source: breadwallet/breadwallet-legacy BreadWallet/BRPeer.h L39 @951bb47 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/bread:`), text: "BRD's wallet core: the BRD app, shut down in 2022, or a wallet built on its code", label: "BRD wallet core"},
	// "/breadwallet:%@/" (BreadWallet/BRPeer.h in breadwallet/breadwallet).
	// Source: breadwallet/breadwallet-legacy BreadWallet/BRPeer.h L39 @951bb47 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)breadwallet`), text: "BRD (breadwallet), a phone wallet shut down in 2022", label: "BRD wallet, closed 2022"},
	// peerGroup.setUserAgent("MultiBit", ...) (MultiBitService.java in
	// Multibit-Legacy/multibit).
	// Source: Multibit-Legacy/multibit src/main/java/org/multibit/network/MultiBitService.java L310 @21e2e9d (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)multibit`), text: "MultiBit, a desktop wallet discontinued in 2017", label: "MultiBit, ended 2017"},
	// "/Bither" + version + "/", no colon (message/VersionMessage.java in
	// Bither/bitherj).
	// Source: bither/bitherj bitherj/src/main/java/net/bither/bitherj/message/VersionMessage.java L124 @1dff0be (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)bither`), text: "Bither, a wallet for phone and desktop", label: "Bither wallet"},
	// Wasabi itself picks a Core name at random, "/Satoshi:30.2.0/" and others
	// (UserAgents in WalletWasabi/Helpers/Constants.cs): a peer saying "wasabi"
	// only borrows the name.
	{re: regexp.MustCompile(`(?i)wasabi`), text: "Named like Wasabi Wallet; Wasabi itself introduces itself as Bitcoin Core", label: "Named like Wasabi"},
	// On bitcoinj, so before its rule: HTTPSeed (crawler.kt in bitcoinj/httpseed)
	// and CoinView (Main.java in vs-uulm/CoinView), see the sources further down.
	// Source: bitcoinj/httpseed src/main/kotlin/crawler.kt L158 @40d2e88 (checked 2026-11-24)
	{re: regexp.MustCompile(`/HTTPSeed:`), text: "Cartographer (HTTPSeed), bitcoinj's crawler behind an HTTP seed, archived", label: "HTTPSeed crawler"},
	// Simple Bitcoin Wallet (btcontract/wallet, package com.btcontract.wallet):
	// appName = "Bitcoin" (Utils.scala), peerGroup.setUserAgent(appName,
	// "1.075") in WalletApp.scala; 1.072 and 1.074 in earlier commits.
	// Source: btcontract/wallet app/src/main/java/com/btcontract/wallet/WalletApp.scala L121 @7ec01b1 (checked 2026-11-25)
	{re: regexp.MustCompile(`^/bitcoinj:[^/]*/Bitcoin:[0-9]`), text: "Simple Bitcoin Wallet by btcontract, an Android wallet", label: "Simple Bitcoin Wallet"},
	// bitcoinj's own programs: peers.setUserAgent("WalletTool", "1.0") in its
	// command-line wallet tool, and APP_NAME = "WalletTemplate" with "1.0" in
	// its desktop wallet template (wallettemplate/.../Main.java L46, L152).
	// Source: bitcoinj/bitcoinj tools/src/main/java/org/bitcoinj/tools/WalletTool.java L1214 @v0.14.4 (checked 2026-11-25)
	{re: regexp.MustCompile(`/WalletTool:`), text: "WalletTool, bitcoinj's command-line wallet", label: "bitcoinj WalletTool"},
	// Source: bitcoinj/bitcoinj wallettemplate/src/main/java/wallettemplate/Main.java L152 @v0.14.4 (checked 2026-11-25)
	{re: regexp.MustCompile(`/WalletTemplate:`), text: "bitcoinj's wallet template, a sample desktop wallet to build on", label: "bitcoinj WalletTemplate"},
	// BlockchainAnalyzer (catena2w/BlockchainAnalyzer, IOHK's Scorex):
	// peerGroup.setUserAgent("SmartContract.com", "0.1") on mainnet.
	// Source: catena2w/BlockchainAnalyzer src/main/scala/iohk/scorex/bitcoin/Analyzer.scala L63 @5e20e76 (checked 2026-11-25)
	{re: regexp.MustCompile(`/SmartContract\.com:`), text: "BlockchainAnalyzer, a research tool that reads the chain through bitcoinj", label: "BlockchainAnalyzer"},
	// You've Got Bitcoin (duncannevin/youvegotbitcoin), sending bitcoin by
	// email: peerGroup.setUserAgent("email bitcoin", "0.1").
	// Source: duncannevin/youvegotbitcoin src/main/scala/com/duncannevin/youvegotbitcoin/bitcoin/BitcoinClient.scala L152 @1804b0f (checked 2026-11-25)
	{re: regexp.MustCompile(`/email bitcoin:`), text: "You've Got Bitcoin, an app that sends bitcoin by email, on bitcoinj", label: "You've Got Bitcoin"},
	// Source: vs-uulm/CoinView collection/application/research/Main.java L24 @d3a9f95 (checked 2026-11-24)
	{re: regexp.MustCompile(`/UU-Network-Research:`), text: "CoinView, Ulm University's collector of transaction announcements", label: "CoinView (Ulm University)"},
	// The library alone, without a wallet app's name after it. After the
	// wallet rules above, which also start with bitcoinj.
	// Source: bitcoinj/bitcoinj core/src/main/java/org/bitcoinj/core/VersionMessage.java L53 @1c84f21 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/bitcoinj:`), text: "bitcoinj, a Java library that wallets and other tools use to talk to nodes", label: "bitcoinj Java library"},
	// neutrino sends a version fixed in its own code, not lnd's: 0.12.0-beta
	// from neutrino v0.12.0 to v0.16.2 (lnd 0.13 to 0.20), then 0.17.0, and
	// 0.17.1 from v0.17.1 on, v0.18.0 too (lnd 0.21, June 2026;
	// UserAgentVersion in neutrino.go in lightninglabs/neutrino).
	// Zeus 13.2 ships lnd 0.21.4, Blixt 0.9 lnd 0.20, and neither changes the
	// name; lnd on a server in neutrino mode says the same, so an app is only an example.
	// Source: lightningnetwork/lnd config.go L665 @v0.21.0-beta (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)/neutrino:0\.17\.`), text: "lnd 0.21 or newer, without a node of its own (neutrino), e.g. Zeus", label: "lnd 0.21+ (neutrino)", name: "lnd 0.21+"},
	// Source: lightninglabs/neutrino neutrino.go L49 @v0.12.0 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)/neutrino:0\.12\.0-beta/`), text: "Older lnd, without a node of its own (neutrino), e.g. Blixt or an older Zeus", label: "Older lnd (neutrino)", name: "older lnd"},
	// Source: lightninglabs/neutrino neutrino.go L52 @55dc1ec (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)/neutrino:`), text: "Neutrino: Lightning wallets on lnd, such as Blixt or Zeus", label: "lnd wallet (neutrino)"},
	// neutrino's own start and version under a name of its own
	// ("/btcwire:0.5.0/btcpost-neutrino:0.17.1/", 14.11.2026): its code,
	// renamed, by someone who has not published it.
	{re: regexp.MustCompile(`(?i)^/btcwire:[^/]*/[^/]+-neutrino:`), text: "Probably neutrino, lnd's light client, under another name: its btcwire start and its version", label: "Probably renamed neutrino"},
	// neutrino under a name its program sets, on btcd's wire library
	// (/btcwire:<v>/ in front): Bisoncraft's wallets set UserAgentName in their
	// copy of neutrino, Zap hands lnd --neutrino.useragentname.
	// Source: bisoncraft/caribou wallet/utxowallet/spv/neutrino.go L49 @fccbc64 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/btcwire:[^/]*/CaribouSPV:`), text: "Caribou, a wallet by Bisoncraft that follows Bitcoin with its own build of neutrino", label: "Caribou wallet"},
	// Source: bisoncraft/utxowallet spv/neutrino.go L49 @422fc40 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/btcwire:[^/]*/BisonSPV:`), text: "Bisoncraft's utxowallet, a wallet module on its own build of neutrino", label: "Bisoncraft utxowallet"},
	// Source: LN-Zap/zap-desktop services/neutrino/neutrino.js L371-372 @d89c6c0 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/btcwire:[^/]*/zap-desktop:`), text: "Zap, a Lightning wallet for the desktop, on lnd without a node of its own (neutrino)", label: "Zap desktop (neutrino)"},
	// Source: LN-Zap/zap-iOS SwiftLnd/Lnd/LocalLndConfiguration.swift L77 @a1af8bf (checked 2026-11-24)
	{re: regexp.MustCompile(`^/btcwire:[^/]*/zap-ios:`), text: "Zap for iPhone, a Lightning wallet on lnd without a node of its own (neutrino)", label: "Zap iPhone (neutrino)"},
	// Only the word in the name ("/neutrino-checker:0.1/", 14.11.2026): no
	// source, so no wallet.
	{re: regexp.MustCompile(`(?i)neutrino`), text: "Named like Neutrino, lnd's light client", label: "Named like Neutrino"},
	// "/Rust BIP-157:<version>/rust-bitcoin:<version>/" (src/network/outbound.rs
	// in 2140-dev/kyoto).
	// Source: 2140-dev/kyoto src/network/outbound.rs L106 @4fa8cc7 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)BIP-157|kyoto`), text: "Kyoto, a light client for wallets built with BDK", label: "Kyoto light client (BDK)"},
	// n.agent = "/Bitflash:btc-light/" (src/btclight.cpp L424 in
	// bitflash-network/bitflash, 15.11.2026).
	// Source: bitflash-network/bitflash src/btclight.cpp L424 @134dc25 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/Bitflash:btc-light/`), kind: "Other light client", text: "Bitflash's wallet, which follows the chain by headers and compact filters (docs.bitflash.network)", label: "Bitflash wallet"},
	// "/Floresta:<version>/" (bin/florestad/build.rs in getfloresta/Floresta).
	// Source: getfloresta/Floresta bin/florestad/build.rs L22 @64238a2 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)floresta`), text: "Floresta, a lightweight Utreexo node", label: "Floresta Utreexo node"},
	// format!("/electrs:{}/", ELECTRS_VERSION) (src/p2p.rs in romanz/electrs,
	// up to v0.10.10; later versions have no P2P code).
	// Source: romanz/electrs src/p2p.rs L339 @v0.10.8 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)electrs`), text: "electrs, the address index behind Electrum-style wallets", label: "electrs address index"},
	// UA_NAME("CKCoinD") (src/clientversion.cpp in ckolivas/ckcoind, branch ck29).
	// Source: ckolivas/ckcoind src/clientversion.cpp L23 @4a0a9a2 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/CKCoinD:`), text: "ckcoind, a mining fork of Bitcoin Core used with ckpool", label: "ckcoind (ckpool fork)"},
	// "/rust-bitcoin:0.18/bluematt-tokio-client:0.1/" (src/peer.rs in
	// dongcarl/dnsseed-rust; TheBlueMatt/dnsseed-rust is gone).
	// Source: dongcarl/dnsseed-rust src/peer.rs L124 @2cf23b8 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)bluematt-tokio-client`), text: "dnsseed-rust, the crawler behind a DNS seeder", label: "dnsseed-rust crawler"},
	// Source: ckolivas/ckpool src/ckp2p.c L537 @e1da45f (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)ckp2p`), text: "ckp2p, ckpool's block relay: passes on blocks only, never transactions", label: "ckp2p block relay"},
	// AppInfo.cs and P2p/P2pFastPeer.cs in bugoao/miningcore-btc-solo.
	// Source: bugoao/miningcore-btc-solo src/MiningcoreBtcSolo/Util/AppInfo.cs L5-9 @8749652 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/miningcore-btc-solo:`), text: "miningcore-btc-solo, a solo mining pool built on Miningcore", label: "miningcore-btc-solo pool"},
	// internal/bitcoin/p2p_client.go in Freakoverse/dnn-node: UserAgentName
	// "DNN-Node" on btcd's peer package, which puts /btcwire:<v>/ before it.
	// Source: Freakoverse/dnn-node internal/bitcoin/p2p_client.go L185 @c8c4775 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/btcwire:[^/]*/DNN-Node:`), text: "DNN Node, a Nostr relay and naming service anchored to Bitcoin", label: "DNN Node"},
	// Relay networks with a name of their own: bloXroute's gateway builds
	// "bloXroute Gateway <version>" (bxgateway, 2022); Matt Corallo's
	// RelayNode adapters send /RelayNetworkClient:42/, /RelayNetworkServer:42/
	// and /RelayNetworkTerminator:42/0000000/ (c++/client.cpp and
	// bitcoindterminator.cpp in TheBlueMatt/RelayNode, 2013-2017); Falcon was
	// seen as /Cornell-Falcon-Network:0.1.0/ in a bips discussion (March 2017),
	// no code, so only the name.
	// Source: bloXroute-Labs/bxgateway src/bxgateway/gateway_constants.py L64 @4d09fdc (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^bloXroute Gateway`), text: "bloXroute Gateway, a relay network's gateway: passes on blocks and transactions", label: "bloXroute gateway"},
	// Source: TheBlueMatt/RelayNode c++/client.cpp L23 @c50d812 (checked 2026-11-24)
	// Source: TheBlueMatt/RelayNode c++/server.cpp L16 @c50d812 (checked 2026-11-24)
	// Source: TheBlueMatt/RelayNode c++/bitcoindterminator.cpp L17 @c50d812 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/RelayNetwork(Client|Server|Terminator):`), text: "Matt Corallo's old Relay Network, which ran until 2017", label: "Old Relay Network"},
	{re: regexp.MustCompile(`^/Cornell-Falcon-Network:`), text: "Named like Falcon, Cornell's block relay network from 2016", label: "Named like Falcon"},
	// Before NBitcoin, whose name is inside "OpenBitcoin".
	// USER_AGENT = '/OpenBitcoin-crawler:0.1/' (crawl.mjs in openbitcoincom/node-crawler).
	// Source: openbitcoincom/node-crawler crawl.mjs L32 @384b00c (checked 2026-11-24)
	{re: regexp.MustCompile(`^/OpenBitcoin-crawler:`), text: "OpenBitcoin's crawler, a survey of reachable nodes for openbitcoin.com", label: "OpenBitcoin crawler"},
	// Source: MetacoSA/NBitcoin NBitcoin/Protocol/Payloads/VersionPayload.cs L69 @2d0fe93 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)NBitcoin`), text: "NBitcoin, a .NET Bitcoin library (behind BTCPay Server, among others)", label: "NBitcoin .NET library"},
	// "/bitcore:<version>/" is bitcore-p2p's default, the version its own
	// (lib/messages/commands/version.js in bitpay/bitcore-p2p), so any program on it says so: BitPay's node and
	// index server, or bitcore-spv's light client (throughnothing/bitcore-spv).
	// Source: bitpay/bitcore-p2p lib/messages/commands/version.js L40 @e99803e (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)/bitcore:`), text: "Bitcore, BitPay's Bitcoin code: its node and index server, or another program on its P2P library", label: "Bitcore (BitPay)"},
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
	{re: regexp.MustCompile(`^/pynode:0\.0\.1/$`), text: "Bitcoin Core handing over a transaction privately, then leaving; the name is that of bitcoin-submittx and pynode, which send it too", label: "Core broadcast or pynode"},
	// Any other pynode version (0.0.1 is Core's private broadcast, above): Laban2122's experimental Python node, which
	// syncs headers and keeps blocks from a set height (NETWORK_LIMITED only,
	// p2p/peer_link.py L136).
	// Source: Laban2122/pynode config.json L31 @22a64ff (checked 2026-11-24)
	{re: regexp.MustCompile(`^/pynode:([1-9]|0\.[1-9]|0\.0\.([02-9]|1[0-9]))`), kind: "Node (other)", text: "pynode, an experimental Python node that keeps blocks from a set height", label: "pynode (Python node)"},
	// m4ycon on bnoc (#198): it sweeps the P2A anchors of a fleet that minted
	// rune 1:0 in chains of transactions; on the node this was written for it
	// pushed unannounced 66-byte transactions, the size of such a sweep. The
	// name is in m4ycon's measurement setup, "user_agent": "/p2aburn:1.0/"
	// (data/experiment/final/params.py in
	// m4ycon/cmpct-block-reconstruction-degradation).
	// Source: m4ycon/cmpct-block-reconstruction-degradation data/experiment/final/params.py L98 @1a0ec59 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/p2aburn:`), text: "p2aburn, a bot that sweeps P2A anchors (outputs anyone may spend) behind a Runes minting fleet", label: "p2aburn anchor sweeper"},
	// strSubVer = b'/python-bitcoinlib:' + version (bitcoin/messages.py L123 in
	// petertodd/python-bitcoinlib, 23.11.2026).
	// Source: petertodd/python-bitcoinlib bitcoin/messages.py L123 @fbbe924 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)python-bitcoinlib`), text: "python-bitcoinlib, a Python library scripts use to talk to nodes", label: "python-bitcoinlib"},
	// PyBLOCK's install scripts set uaappend=PyBLOCK and uacomment=PyBLOCK Crew
	// (install-full-node.sh in curly60e/pyblock at 3d8be3a, deleted in June
	// 2026; still in PyBLOCK-Bitcoin/pyblock). "(PyBLOCK-POOL)" is in no
	// source: a label its operator chose, and its kind is Pool node.
	// Only those two are PyBLOCK's own (23.11.2026): "(PyBLOCK Crew)" and
	// PyBLOCK appended at the end. Any other bracket, "(PyBLOCK-BIP110)" too,
	// an operator may write by hand.
	// Source: PyBLOCK-Bitcoin/pyblock install-full-node.sh L458 @010f211 (checked 2026-11-24)
	// Source: PyBLOCK-Bitcoin/pyblock install-full-node.sh L455 @010f211 (checked 2026-11-24)
	{re: regexp.MustCompile(`\(PyBLOCK Crew\)|PyBLOCK/?$`), text: "PyBLOCK, a node dashboard", label: "PyBLOCK dashboard"},
	{re: regexp.MustCompile(`(?i)\(PyBLOCK|pyblock`), text: "Named like PyBLOCK, a node dashboard", label: "Named like PyBLOCK"},
	// Projects whose name matches; none of them publishes the exact string,
	// so "Named like".
	// Urbit reaches Bitcoin through its %btc-provider (github.com/urbit/urbit);
	// Symbiosis Finance runs a Bitcoin bridge (symbiosis.finance/bridge-btc);
	// DexCash Network is a blockchain of its own (dexcash.org). curl is the
	// download tool and does not speak Bitcoin's protocol, so a peer naming it
	// borrows the name.
	{re: regexp.MustCompile(`^urbit$`), text: "Named like Urbit's Bitcoin software (%btc-provider)", label: "Named like Urbit"},
	{re: regexp.MustCompile(`/Symbiosis Bitcoin Bridge:`), text: "Named like the Bitcoin bridge of Symbiosis Finance, on btcd's Go library", label: "Named like Symbiosis"},
	{re: regexp.MustCompile(`^/curl:`), text: "Named like curl, the download tool; curl itself does not speak Bitcoin's protocol", label: "Named like curl"},
	{re: regexp.MustCompile(`(?i)^/dexcash-nodes:`), kind: "Other chain", text: "Named like DexCash Network, a blockchain of its own", label: "Named like DexCash"},
	// Bitnodes' crawler sends "/bitnodes.io:0.3/" (protocol.py in
	// ayeowch/bitnodes); any other name with Bitnodes in it, /alt-bitnodes:/
	// or /Metrika-Bitnodes:/, is only named like it. BTC Nodes (btcnodes.io),
	// built from Bitnodes' code, keeps the same default.
	// Source: ayeowch/bitnodes conf/crawl.conf.default L53 @036dc5c (checked 2026-11-24)
	// Source: brunneis/btcnodes conf/crawl.conf.default L55 @f73ab93 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/bitnodes\.io:`), text: "Bitnodes' crawler, or one built from its code such as BTC Nodes: maps reachable nodes", label: "Bitnodes crawler"},
	// alt-bitnodes: USER_AGENT in deploy/install.sh (ifuensan/alt-bitnodes),
	// HackNodes' public node count. /bitnodes.earn.com:0.1/: the crawler of a
	// LORIA/Inria study (conf/crawl.conf.default in jpeisenbarth/bitnodes,
	// branch add_statistics; CONCORDIA, last run 2020).
	// Source: ifuensan/alt-bitnodes deploy/install.sh L27 @511a4de (checked 2026-11-24)
	{re: regexp.MustCompile(`^/alt-bitnodes:`), text: "alt-bitnodes, HackNodes' crawler built on Bitnodes, counting reachable nodes", label: "alt-bitnodes crawler"},
	// Source: ayeowch/bitnodes conf/crawl.conf.default L44 @1035ba3 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/bitnodes\.earn\.com:`), text: "A Bitnodes crawler run for a LORIA/Inria research study", label: "Bitnodes (Inria study)"},
	{re: regexp.MustCompile(`(?i)bitnodes`), text: "Named like Bitnodes, the public node map", label: "Named like Bitnodes"},
	// Seen from one netcup address, offering nothing and gone after a second;
	// the name fits a node map, but nothing public says whose.
	{re: regexp.MustCompile(`(?i)GlobalNodeMap`), text: "Named like a node map; nothing public says whose", label: "Named like a node map"},
	// illuminodes/mantle: the node says "/mantle:0.1.0/", its replay tool,
	// which fetches mainnet blocks from peers to validate them, says
	// "/mantle-replay-iouring:0.1.0/" (crates/fetcher/src/lib.rs); beside it
	// "/mantle-discover:0.1.0/" collects addresses with getaddr
	// (replay/src/discover_peers.rs) and "/mantle-rank-peers:0.1.0/" times
	// how fast a node answers its version (replay/src/rank_peers.rs).
	// Source: illuminodes/mantle crates/mantle-node/src/main.rs L33 @bc1522a (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/mantle:`), text: "Mantle, a Bitcoin full node written from scratch in Rust (illuminodes)", label: "Mantle node (Rust)"},
	// Source: illuminodes/mantle crates/replay/src/fetcher.rs L494 @18bd7fa (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/mantle-replay`), text: "Mantle's replay tool: fetches blocks from nodes to check them with Mantle, a Bitcoin node written from scratch in Rust", label: "Mantle replay tool"},
	// Source: illuminodes/mantle replay/src/discover_peers.rs L173 @c3a4fe4 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/mantle-discover`), text: "Mantle's peer finder: asks nodes for addresses of other nodes", label: "Mantle peer finder"},
	// Source: illuminodes/mantle replay/src/rank_peers.rs L534 @c3a4fe4 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/mantle-rank-peers`), text: "Mantle's speed test: times how fast a node answers", label: "Mantle speed test"},
	// btcd's wire package sends "/btcwire:0.5.0/" when a program sets no name
	// of its own (wire/msgversion.go), and its peer package's example sets
	// "peer" 1.0.0 (peer/example_test.go). On the node this was written for,
	// 21. to 25.10.2026, the bare name asked for addresses and headers and
	// wanted no transactions.
	// Source: btcsuite/btcd wire/msgversion.go L20 @3cb9f60 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/btcwire:[^/]*/$`), text: "A program on btcd's Go library that set no name of its own", label: "btcd library, no own name"},
	// btcd's example and wire default together (15.11.2026): exactly this.
	// Source: btcsuite/btcd peer/example_test.go L24 @2c46702 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/btcwire:0\.5\.0/peer:1\.0\.0/$`), text: "A program built from the example code in btcd's Go library, keeping its name", label: "btcd sample code"},
	{re: regexp.MustCompile(`^/btcwire:[^/]*/peer:1\.0\.0/$`), text: "Probably a program built from the example code in btcd's Go library, keeping its name", label: "Probably btcd sample code"},
	// RESET (Trac-Systems/reset) is a coin clocked by Bitcoin's blocks; a
	// share of its reward goes to whoever brings in a new block first. Its
	// agents fetch from mempool.space unless told otherwise and its code has
	// no P2P client, so these names are a guess from the name, the behaviour
	// and the timing: its repository was created on 15.10.2026, and the
	// measured record of the node this was written for, from 13.9., shows the
	// first reset-* name on 20.10.; by 25.10. there were seven, asking only
	// for headers, most wanting no transactions. That is timing and behaviour,
	// not a source: "Named like" (13.11.2026).
	{re: regexp.MustCompile(`(?i)^/reset-`), text: "Named like RESET, a coin clocked by Bitcoin's blocks that pays a small reward to whoever brings in a new block first", label: "Named like RESET", name: "RESET"},
	// The Real Bitcoin (TRB), a bitcoind kept on the old 0.5.4 line: its
	// client name and version 99999, written as "/therealbitcoin.org:0.9.99.99/"
	// by FormatSubVersion (src/util.cpp L899-905).
	// Source: irnb/therealbitcoin src/knobs.h L4-5 @ec4f7f9 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/therealbitcoin\.org:`), text: "The Real Bitcoin (TRB), a bitcoind kept on the old 0.5.4 line", label: "The Real Bitcoin (0.5.4)"},
	// JWRD Computing (jwrd.net) maintains its own bitcoind of the old 0.5
	// line. Its version 0.9.99.99 is TRB's 99999, so it is likely a TRB
	// descendant; nothing public says it sends this name.
	{re: regexp.MustCompile(`^/jwrd\.net:`), text: "Named like JWRD Computing's own bitcoind, a fork of an early version", label: "Named like JWRD"},
	// btcd's wire library with "Satoshi" put after it: two monitors do that,
	// each with its own version, so only the version tells them apart.
	// Source: mit-dci/pooldetective blockobserver/coinparam/bitcoin.go L96-97 @232317e (checked 2026-11-24)
	{re: regexp.MustCompile(`^/btcwire:[^/]*/Satoshi:0\.19/$`), text: "Pool Detective, by MIT's Digital Currency Initiative: watches what mining pools do", label: "MIT's Pool Detective", name: "Pool Detective"},
	// Source: CIRCL/pbtc peer/peer.go L52-53 @05437a4 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/btcwire:[^/]*/Satoshi:0\.9\.3/$`), text: "pbtc, CIRCL's passive Bitcoin network monitor", label: "CIRCL's pbtc monitor", name: "pbtc"},
	{re: regexp.MustCompile(`(?i)^/btcwire:[^/]*/Satoshi:`), kind: "Research scanner", text: "Probably a network monitor on btcd's library that adds Core's name, like Pool Detective or CIRCL's pbtc", label: "Probably a btcd monitor"},
	// Source: mit-dci/lit uspv/init.go L130 @8c3d3b4 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)/lit:v`), text: "lit, a Lightning node by MIT's Digital Currency Initiative, no longer developed", label: "lit Lightning node (MIT)"},
	// observer/src/protocol.rs in 0xB10C/p2p-observer, work in progress in
	// October 2026; extractors/p2p in peer-observer/peer-observer, which a
	// node connects to itself when told to (addnode).
	// src/sync/stages/pull/peer.rs in maestro-org/maestro-symphony;
	// src/crawl.rs in achow101/dnsseedrs; Satscraft/bip324-probe.
	// /your-fork:/ has no sentence of its own (see TestNoKindFromOneNode). The measured
	// record of the node this was written for, 13.9. to 25.10.2026: 241 sessions from 187 addresses, offering nothing, no
	// transactions, gone after 18 seconds at the median; the same handshake
	// as Bitnodes' crawler (version, sendaddrv2, verack, then getaddr). A
	// session that stays, like Bitnodes' own, asks for nothing: getpeerinfo
	// on 31.10.2026, one of 22 hours, no getaddr.
	// Source: maestro-org/maestro-symphony src/sync/stages/pull/peer.rs L36 @ef66a53 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/symphony:`), kind: "Indexer", text: "Maestro Symphony, a Bitcoin indexer and API server", label: "Maestro Symphony indexer"},
	// Source: achow101/dnsseedrs src/crawl.rs L176 @127da83 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/crawlrs:`), text: "dnsseedrs, a DNS seeder by Ava Chow, a Bitcoin Core maintainer", label: "dnsseedrs (Ava Chow)"},
	// Source: Satscraft/bip324-probe src/v1.rs L43 @2cb7d7f (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/bip324-probe:`), kind: "Crawler", text: "bip324-probe, a tool that checks whether nodes offer encrypted v2 connections", label: "bip324-probe v2 checker"},
	// Source: 0xB10C/p2p-observer observer/src/protocol.rs L111 @e265bb9 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/p2p-observer:`), kind: "Crawler", text: "p2p-observer, 0xB10C's tool that connects to nodes to watch the network", label: "0xB10C's p2p-observer"},
	// Source: peer-observer/peer-observer extractors/p2p/src/lib.rs L39 @8889014 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/p2p-extractor:`), kind: "Crawler", text: "peer-observer's p2p extractor: nodes are told to connect to it, and it records what they send", label: "peer-observer's extractor"},
	// Sources, 14.11.2026: AgenticBitcoin's USER_AGENT
	// (crates/abtc-adapters/src/network/mod.rs in casey-bowman/agentic-bitcoin),
	// SatoshiSharp's UserAgent (src/ConsoleApp/MainBlockDownload.cs in
	// tqfoley/BtcSatoshiSharp), NBXplorer's "NBXplorer-" + a random number
	// (NBXplorer/Backend/Indexer.cs in btcpayserver/NBXplorer).
	// Source: casey-bowman/agentic-bitcoin crates/abtc-adapters/src/network/mod.rs L42 @c192131 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/AgenticBitcoin:`), text: "AgenticBitcoin, Bitcoin rewritten in Rust by AI agents", label: "AgenticBitcoin (AI-made)"},
	// Source: tqfoley/BtcSatoshiSharp src/ConsoleApp/MainBlockDownload.cs L110 @3ce6490 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/SatoshiSharp:`), text: "SatoshiSharp, parts of Bitcoin written in C#", label: "SatoshiSharp (C#)"},
	// Source: btcpayserver/NBXplorer NBXplorer/Backend/Indexer.cs L284 @b3ab010 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^NBXplorer-`), kind: "Indexer", text: "NBXplorer, the indexer behind BTCPay Server", label: "NBXplorer (BTCPay)"},
	// NodeConstants.scala in bitcoin-s/bitcoin-s; indexer.rs in stacks-core.
	// Source: bitcoin-s/bitcoin-s core/src/main/scala/org/bitcoins/core/api/node/constant/NodeConstants.scala L4 @a928f38 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/bitcoin-s:`), kind: "Other light client", text: "bitcoin-s, a Bitcoin toolkit in Scala with a light client on compact block filters", label: "bitcoin-s toolkit (Scala)"},
	// Source: stacks-network/stacks-core stackslib/src/burnchains/bitcoin/indexer.rs L54 @7c17b1d (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^Stacks/`), kind: "Other light client", text: "A Stacks node: follows Bitcoin's blocks for the Stacks chain built on it", label: "Stacks node"},
	// userAgentName "utreexod" (server.go in utreexo/utreexod); spvwallet's
	// UserAgent (config.go in OpenBazaar/spvwallet).
	// Source: utreexo/utreexod server.go L73 @ccd1afc (checked 2026-11-24)
	{re: regexp.MustCompile(`/utreexod:`), text: "utreexod, a Utreexo full node built on btcd", label: "utreexod Utreexo node"},
	// Source: cloudhead/nakamoto p2p/src/fsm.rs L70 @76ab7a3 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/nakamoto:`), kind: "Other light client", text: "nakamoto, a light client on compact block filters", label: "nakamoto light client"},
	// Source: OpenBazaar/spvwallet config.go L73 @225c519 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)/spvwallet:`), text: "spvwallet, the wallet of OpenBazaar, a marketplace closed in 2021", label: "OpenBazaar's spvwallet"},
	// Autarkysoft's C# toolkit (Configuration.cs); it pushes transactions
	// under "/Satoshi:0.22.0/" instead, so only its own name lands here.
	// Source: Autarkysoft/Denovo Src/Denovo/Models/Configuration.cs L26 @858aff3 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/Denovo:`), text: "Denovo, a Bitcoin toolkit in C#: pushes transactions, mines on testnet", label: "Denovo toolkit (C#)"},
	// network.py in jimmysong/programmingbitcoin, and session7/network.py in
	// jimmysong/pb-exercises for the course.
	// Source: jimmysong/programmingbitcoin code-ch10/network.py L111 @1aa1a63 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/programming(bitcoin|blockchain):`), text: "Code from Jimmy Song's book or course Programming Bitcoin", label: "Programming Bitcoin code"},
	// The example code of GladosBlueWallet/bip324 (examples/signet-getaddr.ts),
	// the TypeScript library Blueberry's transport was ported from.
	// Source: GladosBlueWallet/bip324 examples/signet-getaddr.ts L32 @8abbc6d (checked 2026-11-24)
	{re: regexp.MustCompile(`^/bip324-ts:`), text: "bip324-ts, a TypeScript library for encrypted v2 connections (its example code)", label: "bip324-ts library"},
	// dsn.kastel.kit.edu/bitcoin, the DSN group at KIT: the page says it
	// measures the network for research, but names no user agent. The domain
	// in the name points there.
	{re: regexp.MustCompile(`(?i)^/dsn\.[a-z]+\.kit\.edu/`), text: "Probably KIT Karlsruhe's Bitcoin network monitoring", label: "Probably KIT monitoring", name: "KIT"},
	// app/network/scanner.py in DogukanBaysal/DaeConn. The other two only from
	// 0xB10C's list of 15.05.2023, what he saw, not their code (entities.toml
	// in bitcoin-noc/network-entities, formerly banlist): "Names".
	// Source: DogukanBaysal/DaeConn app/network/scanner.py L90 @698f77c (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/connector/$`), kind: "Research scanner", text: "DaeConn, a research tool that measures the networks of Bitcoin and its forks", label: "DaeConn research tool"},
	// nodes.mom.market: MOM.market is Multiven's marketplace, and Multiven said
	// in March 2018 it monitored Bitcoin nodes (via nodes.multiven.io); nothing
	// names this subdomain or its crawler. bitcoinstats.com: Christian Decker
	// runs its DNS seed crawler (delvingbitcoin.org, 01.03.2024), but no source
	// sets this name (GPT research, 23.11.2026).
	{re: regexp.MustCompile(`(?i)^/nodes\.mom\.market:`), kind: "Crawler", text: "Named like nodes.mom.market, a site that monitors Bitcoin nodes", label: "Named like mom.market"},
	{re: regexp.MustCompile(`(?i)^/btcwire:[^/]*/bitcoinstats\.com:`), kind: "Crawler", text: "Named like bitcoinstats.com, a site with statistics on the Bitcoin network", label: "Named like bitcoinstats"},
	// btcd's wire library with the app name after it left empty.
	// Source: btcsuite/btcd wire/msgversion.go L257 @3cb9f60 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/btcwire:[^/]*/:/$`), text: "A program on btcd's Go library that left its own name empty", label: "btcd library, empty name"},
	// Source: btcsuite/btcd server.go L67 @2c46702 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)btcd:`), text: "btcd, a full node written in Go", label: "btcd node (Go)"},
	// Sources, 14.11.2026: BC_USER_AGENT (include/bitcoin/system/preprocessor.hpp
	// in libbitcoin/libbitcoin-system), USER_AGENT (lib/net/common.js in
	// bcoin-org/bcoin), UserAgent (client/common/config.go in piotrnar/gocoin).
	// Source: libbitcoin/libbitcoin-network include/bitcoin/network/settings.hpp L32 @c66f70c (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)libbitcoin`), text: "libbitcoin, a full node written in C++, independent of Core", label: "libbitcoin node"},
	// Source: bcoin-org/bcoin lib/net/common.js L144 @a8dee6d (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)/bcoin:`), text: "bcoin, a full node written in JavaScript", label: "bcoin node (JS)"},
	// Source: piotrnar/gocoin client/common/config.go L504 @ca8789c (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)gocoin`), text: "Gocoin, a full node and wallet written in Go", label: "Gocoin node and wallet"},
	// gosuda/bitcoin-rs, crates/consensus/src/primitives/version.rs: "/bitcoin-rs:<version>/".
	// Source: gosuda/bitcoin-rs crates/consensus/src/primitives/version.rs L3 @9acd366 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/bitcoin-rs:`), text: "bitcoin-rs, a full node written in Rust, independent of Core", label: "bitcoin-rs node (Rust)"},
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
	{re: regexp.MustCompile(`^/bitcoinpr:`), text: "BitcoinPR, a Bitcoin full node written in Rust", label: "BitcoinPR node (Rust)"},
	// Source: btclib-org/btclib-node src/btclib_node/constants.py L136 @d74075c (checked 2026-11-24)
	{re: regexp.MustCompile(`^/btclib:`), text: "btclib-node, a Bitcoin full node written in Python", label: "btclib node (Python)"},
	// Source: BTCDecoded/blvm-node src/network/network_manager.rs L2502 @65bf5dc (checked 2026-11-24)
	{re: regexp.MustCompile(`^/Bitcoin Commons:`), text: "Bitcoin Commons (BLVM), a Bitcoin full node written in Rust", label: "Bitcoin Commons (Rust)"},
	// Source: coreyphillips/bitcoin-spec-node crates/node/src/node.rs L536 @2396240 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/bitcoin-spec-node:`), text: "bitcoin-spec-node, a research node in Rust that checks every block but serves none", label: "bitcoin-spec-node"},
	// Source: Galaxoid-Labs/forseti wire/types.odin L20 @069d3f6 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/Forseti:`), text: "Forseti, an experimental Bitcoin full node written in Odin", label: "Forseti node (Odin)"},
	// Source: swift-bitcoin/swift-bitcoin src/bitcoin-transport/messages/VersionMessage.swift L5 @4a87c60 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/SwiftBitcoin:`), text: "Swift Bitcoin, an experimental Bitcoin full node written in Swift", label: "Swift Bitcoin node"},
	// Source: refined-element/BitcoinWolfe crates/wolfe-types/src/lib.rs L22 @e419cec (checked 2026-11-24)
	{re: regexp.MustCompile(`^/BitcoinWolfe:`), text: "BitcoinWolfe, an experimental node in Rust on libbitcoinkernel, Core's validation engine", label: "BitcoinWolfe node"},
	// Source: subcoin-project/subcoin crates/subcoin-network/src/peer_manager.rs L77 @9999efc (checked 2026-11-24)
	{re: regexp.MustCompile(`^/Subcoin:`), text: "Subcoin, an experimental Bitcoin node written in Rust", label: "Subcoin node (Rust)"},
	// Source: jrawsthorne/rust-bitcoin-node src/net/new_peer.rs L206 @9c7d4f5 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/rust_bitcoin_node:`), text: "rust-bitcoin-node, a Rust full node, no longer developed", label: "rust-bitcoin-node"},
	// Source: stratisproject/StratisBitcoinFullNode src/Stratis.Bitcoin/Configuration/NodeSettings.cs L135 @f52659f (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^StratisNode:`), kind: "Node (other)", text: "Stratis Bitcoin Full Node, a node in C#, archived in 2021", label: "Stratis node (C#)"},
	// Source: mhanne/bitcoin-ruby lib/bitcoin/protocol/version.rb L21 @0977b05 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/bitcoin-ruby:`), text: "bitcoin-ruby's node, a Ruby node from before SegWit", label: "bitcoin-ruby node"},
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
	{re: regexp.MustCompile(`(?i)^/open-nodes:`), kind: "Crawler", text: "Open Nodes, a crawler that maps reachable nodes", label: "Open Nodes crawler"},
	// Source: wslcb/btz src/version.zig L38 @3fedab9 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/btz:`), kind: "Crawler", text: "btz, a crawler in Zig that asks nodes for addresses", label: "btz crawler"},
	// Source: crytic/fluxture fluxture/bitcoin.py L467 @9c384d4 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^fluxture$`), kind: "Crawler", text: "Fluxture, Trail of Bits' crawler for peer-to-peer networks", label: "Fluxture (Trail of Bits)"},
	// Source: justinmoon/crawler net.py L129 @bc1f0b0 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/buidl-bootcamp/`), kind: "Crawler", text: "The crawler of Justin Moon's BUIDL Bootcamp, a teaching DNS seed from 2019", label: "BUIDL Bootcamp crawler"},
	// Source: shazow/btc-crawl cmd.go L32 @601a21d (checked 2026-11-24)
	{re: regexp.MustCompile(`^/btc-crawl:`), text: "btc-crawl, a crawler by Andrey Petrov (shazow)", label: "btc-crawl crawler"},
	// Source: bitcoinanatomy/bitcoinanatomy explorer/scripts/crawler.py L45 @3d9c5d3 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/btc-crawler:`), text: "Anatomy of Bitcoin's crawler, for its map of reachable nodes", label: "Anatomy of Bitcoin"},
	// Source: jochemin/crawly backend/src/p2p/p2p.rs L538 @797a536 (checked 2026-11-24)
	{re: regexp.MustCompile(`^Crawly$`), text: "Crawly, a crawler behind a node-map app by jochemin", label: "Crawly crawler"},
	// Source: danielabrozzoni/new-p2p-crawler src/protocol.rs L21 @811a35b (checked 2026-11-24)
	{re: regexp.MustCompile(`^/new-p2p-crawler:`), text: "new-p2p-crawler, Daniela Brozzoni's crawler of reachable nodes", label: "new-p2p-crawler"},
	// Source: OriginalEveres/bitcoin-node-scanner src/handshake/handshake.ts L157 @bd99850 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/bitcoin-node-scanner:`), text: "bitcoin-node-scanner, a crawler that asks nodes for addresses", label: "bitcoin-node-scanner"},
	// Source: dutenrapha/bitcoin-topology-discovery bitcoin_probe.py L13 @ff1cfe0 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/topology-probe:`), kind: "Crawler", text: "A topology crawler (bitcoin-topology-discovery) that asks nodes for addresses", label: "Topology crawler"},
	// Source: p2pobservatory/bitcoin-crawler crawl_btcd.go L35 @11dca5a (checked 2026-11-24)
	{re: regexp.MustCompile(`^/crawler:0\.1/$`), text: "The crawler of P2P Observatory (IMDEA Networks)", label: "P2P Observatory crawler"},
	// Source: jameslitton/coinscope netmine.cfg L40 @7edb2a9 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/Coinscope-GH:`), kind: "Research scanner", text: "CoinScope, a University of Maryland study of Bitcoin's network", label: "CoinScope (UMD study)"},
	// Source: Crepepepe/JBS setting.properties L3 @664d28c (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/JBS:`), kind: "Research scanner", text: "JBS, a Bitcoin node scanner from university studies", label: "JBS research scanner"},
	// Source: thickzero/TxEcho txecho.go L748 @06ab7c2 (checked 2026-11-24)
	{re: regexp.MustCompile(`^/txecho:`), text: "TxEcho, a payment monitor that watches nodes for transactions and blocks", label: "TxEcho payment monitor"},
	// No public source. On the node this was written for, 28.10. to
	// 01.11.2026: 23 sessions from 16 addresses, all in Cloudflare's network
	// (AS13335), none sending BIP 331's sendpackages, gone after 11 s (median).
	// Only the name is said: what one node saw is no rule for all of them.
	{re: regexp.MustCompile(`^/cf-package-relay:`), text: "Named like package relay, which passes related transactions on together (BIP 331)", label: "Named like package relay"},
	// Bitcoin Cash clients carry its block size as "(EB8.0)" too, so they
	// come before Classic.
	// Source: Bitcoin-ABC/bitcoin-abc src/clientversion.cpp L19 @012f530 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)Bitcoin ABC`), text: "Bitcoin ABC, a Bitcoin Cash client, eCash's node since 2020", label: "Bitcoin ABC (BCH)"},
	// Clients of other chains that still dial Bitcoin's port.
	// Bitcoin Classic's "(EB8)" is Bitcoin Cash's block size; SuperBitcoin,
	// Irium and VDS are forks and altcoins of their own. Bitcoin Purity
	// (github.com/saltduck/bitcoinpurity), a fork of Knots, made BIP110
	// permanent in a hard fork at block 961637 with a difficulty of its own;
	// it says "/Satoshi:29.4/Purity:1.1.0/" (FormatSubVersion in
	// src/clientversion.cpp). On the node this was written for, 21. to
	// 29.10.2026: 269 sessions, all offering its bit 25 and BIP110's bit 27.
	// Bitcoin Unlimited's Bitcoin edition (2015 to 2018) was a big-block
	// client for Bitcoin itself, not another chain; it sends "(EB16)" too, so
	// its sentence comes before the "(EB" one.

	// src/clientversion.cpp in BitcoinUnlimited/BitcoinUnlimited:
	// "BitcoinUnlimited" from 2015 (b126b10), a big-block client for Bitcoin,
	// whose Bitcoin edition ended in January 2018 (85f5bd6); "BUCash" for the
	// Bitcoin Cash edition from 25.07.2017 (849199f), "BCH Unlimited" with a
	// space from 02.04.2020 (4ce5b9e). Before the "(EB" rule: they send it too.
	// Source: BitcoinUnlimited/BitcoinUnlimited src/clientversion.cpp L16 @b126b10 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/BitcoinUnlimited:`), kind: "Node (other)", text: "Bitcoin Unlimited, a big-block client for Bitcoin; this edition ended in 2018", label: "Bitcoin Unlimited (old)"},
	// Source: BitcoinUnlimited/BitcoinUnlimited src/clientversion.cpp L20 @bucash1.1.1.1 (checked 2026-11-24)
	// Source: BitcoinUnlimited/BitcoinUnlimited src/clientversion.cpp L19 @05de381 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)BUCash|BCH ?Unlimited`), text: "Bitcoin Unlimited's Bitcoin Cash edition (BUCash, now BCH Unlimited)", label: "Bitcoin Unlimited (BCH)"},
	// Source: bitcoin-sv/bitcoin-sv src/clientversion.cpp L17 @6504a3a (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)Bitcoin SV`), text: "Bitcoin SV, the node of BSV, which split from Bitcoin Cash in 2018", label: "Bitcoin SV (BSV)"},
	// Source: bitcoinxt/bitcoinxt src/clientversion.cpp L23 @48bbdd4 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)Bitcoin XT`), text: "Bitcoin XT, a fork of Core for bigger blocks that went on to Bitcoin Cash", label: "Bitcoin XT"},
	// Source: SuperBitcoin/SuperBitcoin src/sbtccore/clientversion.cpp L17 @90b776e (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)SuperBitcoin`), text: "Super Bitcoin, a fork of Bitcoin from December 2017 with a chain of its own", label: "Super Bitcoin fork"},
	// USER_AGENT "/iriumd:1.9.66/" (src/p2p_wire.rs in iriumlabs/irium), the
	// one wire code it uses for Bitcoin's network too: version, getheaders,
	// headers (15.11.2026).
	// Source: iriumlabs/irium src/p2p_wire.rs L35 @85c2c0e (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)iriumd`), text: "Irium, an altcoin with a chain of its own; its node also reads Bitcoin's block headers", label: "Irium altcoin"},
	// v-dimension/vds-core sends "/MagicBean:x/" (CLIENT_NAME in
	// src/clientversion.cpp), so "VDS <version>" is only the name.
	// Bitcoin Finance's own announcement (bitcointalk topic 5500641, June
	// 2024) prints "Satoshi-BTF(BitcoinFinance)-Main-DCR:0.15.1" as its node's
	// version; no code of it is public, hence "Probably".
	{re: regexp.MustCompile(`^/Satoshi-BTF\(BitcoinFinance\)`), text: "Probably Bitcoin Finance (BTF), a fork of Bitcoin with a chain of its own", label: "Probably BTF fork"},
	{re: regexp.MustCompile(`(?i)vds[_ ]`), text: "Named like VDS, an altcoin with a chain of its own", label: "Named like VDS"},
	// CLIENT_NAME("Bitcoin Interest") and CLIENT_NAME("Bitcoin Clashic ABCD")
	// (src/clientversion.cpp in BitcoinInterestOfficial/BitcoinInterest and
	// Bitcoin-Clashic/bitcoin-clashic).
	// Source: BitcoinInterestOfficial/BitcoinInterest src/clientversion.cpp L16 @671384e (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/Bitcoin Interest:`), kind: "Other chain", text: "Bitcoin Interest, a fork of Bitcoin with a chain of its own", label: "Bitcoin Interest fork"},
	// Source: Bitcoin-Clashic/bitcoin-clashic src/clientversion.cpp L16 @3d18496 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)^/Bitcoin Clashic`), kind: "Other chain", text: "Bitcoin Clashic, a fork of Bitcoin Cash from 2017", label: "Bitcoin Clashic (BCH)"},
	// Source: bitcoinclassic/bitcoinclassic src/Application.cpp L172 @4c27454 (checked 2026-11-24)
	{re: regexp.MustCompile(`(?i)Classic:`), text: "Bitcoin Classic: a Bitcoin Cash client", label: "Bitcoin Classic (BCH)"},
	// "(EB8)" is the block size a big-block client accepts: Classic, Unlimited
	// and others all send it, so alone it says only that much.
	{re: regexp.MustCompile(`\(EB[0-9]`), text: "Probably a big-block client: \"EB\" is the block size it accepts", label: "Probably big-block client"},

	// Kind only: the words a whole family of software uses, and kinds no
	// sentence goes with. Asked after every line above.
	{re: regexp.MustCompile(`(?i)Bitcoin ABC`), kind: "Other chain"},
	{re: regexp.MustCompile(`(?i)BUCash`), kind: "Other chain"},
	{re: regexp.MustCompile(`(?i)Bitcoin SV`), kind: "Other chain"},
	{re: regexp.MustCompile(`(?i)BCH ?Unlimited`), kind: "Other chain"},
	{re: regexp.MustCompile(`(?i)Bitcoin XT`), kind: "Other chain"},
	{re: regexp.MustCompile(`(?i)OracleKnots:`), kind: "Other chain"},
	{re: regexp.MustCompile(`(?i)UASF-BIP110:`), kind: "Other chain"},
	{re: regexp.MustCompile(`(?i)Classic:`), kind: "Other chain"},
	{re: regexp.MustCompile(`(?i)\(EB[0-9]`), kind: "Other chain"},
	{re: regexp.MustCompile(`(?i)SuperBitcoin`), kind: "Other chain"},
	{re: regexp.MustCompile(`(?i)iriumd`), kind: "Other chain"},
	{re: regexp.MustCompile(`(?i)vds[_ ]`), kind: "Other chain"},
	{re: regexp.MustCompile(`(?i)/Purity:`), kind: "Other chain"},
	{re: regexp.MustCompile(`(?i)^/Satoshi-BTF\(`), kind: "Other chain"},
	// Network scanners, in two flavours: the ones run as a service and the
	// ones run by universities. Both announce themselves honestly.
	// Pool Detective, MIT's Digital Currency Initiative watching what mining
	// pools do: its block observer speaks btcd's old wire library and adds
	// "Satoshi:0.19" to the library's name, "/btcwire:0.4.1/Satoshi:0.19/"
	// (blockobserver/coinparam/bitcoin.go in mit-dci/pooldetective).
	// CIRCL's pbtc does the same with "Satoshi:0.9.3" (peer/peer.go in
	// CIRCL/pbtc); their sentences tell them apart by the version.
	// /trinetra:/ has no public source and stays "Other" (/p2p-probe:/, no
	// source either, is a crawler by what its name says):
	// in peer-observer's record of the node this was written for (21. to
	// 29.10.2026) each only did the handshake and left within a second, every
	// time (39 and 3 sessions), but one node is no rule. /rske-research:/ and /acsrg-research:/ say
	// what they are; the second took 81,563 transactions in one session.
	// /dataforge-observer:/ only listened to what was announced.
	// DaeConn, a measurement tool for Bitcoin-derived networks, says
	// "/connector/" (app/network/scanner.py in DogukanBaysal/DaeConn).
	{re: regexp.MustCompile(`(?i)kit\.edu`), kind: "Research scanner"},
	{re: regexp.MustCompile(`(?i)dsn\.tm`), kind: "Research scanner"},
	{re: regexp.MustCompile(`(?i)dsn\.kastel`), kind: "Research scanner"},
	{re: regexp.MustCompile(`(?i)\.ac\.`), kind: "Research scanner"},
	{re: regexp.MustCompile(`(?i)uni-`), kind: "Research scanner"},
	{re: regexp.MustCompile(`(?i)^/rske-research:`), kind: "Research scanner"},
	{re: regexp.MustCompile(`(?i)^/acsrg-research:`), kind: "Research scanner"},
	{re: regexp.MustCompile(`(?i)^/dataforge-observer:`), kind: "Research scanner"},
	{re: regexp.MustCompile(`(?i)/UU-Network-Research:`), kind: "Research scanner"},
	// Crawlers, seeders and node counters: they collect addresses and look,
	// and several of them say so only as "scan", "seeder", "census" or
	// "monitor" - on one node 953 sessions of /bitcoin-seeder/ and 372 of
	// /btc-range-scan/ used to land in "Other". /cryptexplorer:/,
	// /BlockSail:/ and /your-fork:/ have no public source and stay "Other",
	// although on that node they asked for addresses and nothing else
	// (peer-observer, 21. to 29.10.2026); /topology-probe:/ is a crawler by its
	// source (bitcoin_probe.py in dutenrapha/bitcoin-topology-discovery), the
	// other probes by what their names say. BIP-110 Observer collects peer
	// user agents and services (app/crawler.py in decentralizedb/bip110-observer).
	// nodes.mom.market and bitcoinstats.com listened to what was announced and
	// asked for nothing in 0xB10C's list of 15.05.2023 (entities.toml on the
	// 2023-05-inv-eaters branch of bitcoin-noc/banlist, now network-entities). bitcoin-peers, a
	// toolkit with a crawler, says "/bitcoin-peers:<version>/"
	// (connection/src/connection/configuration.rs in nyonson/bitcoin-peers);
	// the Logos peer list checks onion peers for the Logos wallet
	// (crawler/src/p2p.rs in logos-co/logos-bitcoin-peer-list).
	// Koyn's header tool says "/bitcoinj:0.14.7/PeerMonitor:1.0/" (Main.java in
	// elkrem/koyn-sync): a light client, not a crawler, whatever "monitor" says.
	{re: regexp.MustCompile(`/PeerMonitor:`), kind: "Other light client"},
	{re: regexp.MustCompile(`(?i)bitnodes`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)bluematt-tokio-client`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)metrika`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)nodemap`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)crawl`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)scan`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)seeder`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)census`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)monitor`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)observatory`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)sonar`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)scout`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)nebula`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)watch`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)listener`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)argus`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)readonly`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)bitdash`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)logosnaut`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)dnsseed`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)survey`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)lionseed`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)gnoban`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)^/p2p-probe:`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)^/probe:`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)^/BTC-Nodes:[^/]*/Probe/`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)^/btc-peers:`), kind: "Crawler"},
	{re: regexp.MustCompile(`(?i)/HTTPSeed:`), kind: "Crawler"},
	// Address indexers for wallets: they follow the chain but relay nothing.
	// "/mempool" at the start of a part of the agent: "/mempool/" and
	// "/mempool.space:1.0/" are the indexer, "(mempool.guide)" is a node label.
	// NBXplorer, BTCPay Server's indexer, names itself "NBXplorer-" and a
	// random number (NBXplorer/Backend/Indexer.cs in btcpayserver/NBXplorer).
	{re: regexp.MustCompile(`(?i)electrs`), kind: "Indexer"},
	{re: regexp.MustCompile(`(?i)electrum`), kind: "Indexer"},
	{re: regexp.MustCompile(`(?i)esplora`), kind: "Indexer"},
	{re: regexp.MustCompile(`(?i)/mempool`), kind: "Indexer"},
	// SPV and mobile wallets. bitcoinj is the Android wallet's library. Most
	// of them ask a node for their transactions through a bloom filter
	// (BIP37), which Core leaves off by default: at such a node they leave
	// within a second (peer-observer, 21. to 24.10.2026: Bitcoin Wallet a
	// median 0.1 s, BRD 0.3 s), at one that offers it they stay. /WalletKit:/
	// is Unstoppable Wallet on iPhone (Peer.swift in
	// horizontalsystems/BitcoinCore.Swift), /BitcoinKit:/ the same wallet on
	// Android (VersionMessage.kt in horizontalsystems/bitcoin-kit-android).
	{re: regexp.MustCompile(`(?i)bitcoinj`), kind: "Wallet"},
	{re: regexp.MustCompile(`(?i)breadwallet`), kind: "Wallet"},
	{re: regexp.MustCompile(`(?i)/bread:`), kind: "Wallet"},
	{re: regexp.MustCompile(`(?i)bither`), kind: "Wallet"},
	{re: regexp.MustCompile(`(?i)multibit`), kind: "Wallet"},
	{re: regexp.MustCompile(`(?i)wasabi`), kind: "Wallet"},
	{re: regexp.MustCompile(`(?i)Bitcoin Wallet`), kind: "Wallet"},
	{re: regexp.MustCompile(`(?i)/hodlwallet:`), kind: "Wallet"},
	{re: regexp.MustCompile(`(?i)/spvwallet:`), kind: "Wallet"},
	{re: regexp.MustCompile(`(?i)^/btcwire:[^/]*/(CaribouSPV|BisonSPV):`), kind: "Wallet"},
	// Lightning wallets on lnd that follow the chain through neutrino, its
	// BIP157 client: Zeus and Blixt among them. Named apart from the other
	// light clients so the wallets among the peers show as wallets. In
	// peer-observer's record of the node this was written for, 21. to
	// 24.10.2026: 403 sessions from 78 addresses, more than any other wallet
	// besides Wasabi.
	// lit, the Lightning node MIT's Digital Currency Initiative built with an
	// SPV client of its own, says "/btcwire:0.4.1/lit:v0.1/" (uspv/init.go
	// in mit-dci/lit, last changed in 2022).
	// Only neutrino's own name: one that only has the word in it (a checker,
	// a renamed copy) is no wallet anyone has shown (14.11.2026).
	{re: regexp.MustCompile(`(?i)/neutrino:`), kind: "Lightning wallet"},
	{re: regexp.MustCompile(`(?i)/lit:v`), kind: "Lightning wallet"},
	{re: regexp.MustCompile(`(?i)^/btcwire:[^/]*/zap-(desktop|ios):`), kind: "Lightning wallet"},
	// Other BIP157 light clients: Kyoto, which announces itself as
	// "Rust BIP-157". btcwire alone is only
	// the Go p2p library, which says nothing about the program using it, so it
	// stays out here and is caught by "Other" below.
	// VerifiedX, a sidechain that holds bitcoin: each of its validators keeps
	// eight Bitcoin peers to follow the chain itself, asking for headers and
	// now and then a block, never a transaction (BtcSources.cs and
	// BtcLightClient.cs in VerifiedXBlockchain/VerifiedX-Core, from 9.0 of
	// 24.10.2026). On the node this was written for, 12 of 238 peers that
	// day, all inbound, from 12 addresses at six hosting companies.
	// nakamoto, cloudhead's light client on compact block filters, says
	// "/nakamoto:0.3.0/" (p2p/src/fsm.rs). /multiseed:/ has no public source and
	// stays "Other"; on that node it stayed 52 minutes asking for headers and loading a bloom
	// filter, and took compact filter headers (peer-observer, 27.10.2026).
	// EMPORION, renamed SYMMARTYS, fetches headers to verify OpenTimestamps
	// proofs (bitcoin_headers.py in alexar76/emporion, under the old and new
	// package names). Both versions send their own name with version 0.3.
	// Bitflash's wallet speaks Bitcoin P2P itself and loads only headers and
	// compact filter hashes from peers that offer NODE_COMPACT_FILTERS
	// (docs.bitflash.network/atomic-swaps), as "/Bitflash:btc-light/"
	// (src/btclight.cpp in bitflash-network/bitflash). WUJI's indexer follows
	// Bitcoin headers for an index on Ethereum, as "/wuji-headers:0.1/"
	// (indexer/bitcoin-p2p.mjs in MM-sheng/wuji).
	// pcoc's Bari timestamping service follows mainnet headers only, never a
	// block body: USER_AGENT "/pcoc-bari:<v>/" in sdk/rust/pcoc-gossip/src/wire.rs
	// in groundinglab/pcoc (18.11.2026).
	{re: regexp.MustCompile(`(?i)BIP-157`), kind: "Other light client"},
	{re: regexp.MustCompile(`(?i)^/bip157-fetch:`), kind: "Other light client"},
	{re: regexp.MustCompile(`(?i)kyoto`), kind: "Other light client"},
	{re: regexp.MustCompile(`(?i)^/pycoinnet/$`), kind: "Other light client"},
	{re: regexp.MustCompile(`(?i)/bitcoin-net:`), kind: "Other light client"},
	// A small Utreexo node. Named so it is not lost in "Other", and counted as
	// relaying like any node - the dashboard has no rule for it either.
	{re: regexp.MustCompile(`(?i)floresta`), kind: "Node (Floresta)"},
	// Other full-node software, named rather than left in "Other".
	{re: regexp.MustCompile(`(?i)btcd:`), kind: "Node (btcd)"},
	{re: regexp.MustCompile(`(?i)libbitcoin`), kind: "Node (libbitcoin)"},
	{re: regexp.MustCompile(`(?i)/bcoin:`), kind: "Node (bcoin)"},
	{re: regexp.MustCompile(`(?i)gocoin`), kind: "Node (Gocoin)"},
	// Full nodes written from scratch, each sending its own name (checked in
	// its source): the hashhog fleet of ten languages (github.com/hashhog),
	// reardencode/rbitcoin, AmazingAng/rbtc, bitcoinecho/bitcoin-echo,
	// BobClawblaw/bitcoinmachinecode, kernel-node/kernel-node, tbcd, the
	// Hemi network's Bitcoin daemon (hemilabs/heminetwork),
	// illuminodes/mantle, gosuda/bitcoin-rs, epochbtc/satd and
	// yuvicc/bitcoinkernel-node. hashhog's ouroboros now says
	// "/ouroboros:0.1.0/" (src/ouroboros/peer.py), and chjj/mako, a full node in
	// C, "/mako:0.0.0/" (include/mako/net.h); BTC-Prometheus, a fork of Core,
	// UA_NAME("Prometheus") in src/clientversion.cpp (CryptoIceMLH/
	// bitcoin-prometheus, 15.11.2026). Two more have no public source and stay
	// "Other", although they acted as full nodes on the node this was written for (peer-observer, 21. to
	// 29.10.2026): /BTC-Nodes:/ was among its own outbound
	// peers, offering blocks (NETWORK) and relaying transactions for hours.
	// /Natasha:/, which offered the same and sent headers, has a source since
	// 24.11.2026: Comb Downloader NG, which serves headers and blocks on. BTC-Nodes' other
	// names, "/BTC-Nodes:<date>/Sonar/" and ".../Probe/", are caught as a
	// crawler first.
	{re: regexp.MustCompile(`(?i)^/(CamlCoin|rustoshi|LunarBlock|Haskoin|beamchain|blockbrew|clearbit|nimrod|hotbuns|bitcoin-hybrid|ouroboros|mako|rbitcoin|rbtc|BitcoinEcho|BitcoinMachineCode|kernel-node|bitcoinkernel-node|tbcd|mantle|bitcoin-rs|satd|bitcoinpr|btclib|Bitcoin Commons|bitcoin-spec-node|Forseti|SwiftBitcoin|BitcoinWolfe|Subcoin|rust_bitcoin_node|bitcoin-ruby|Prometheus|bridge|exnode|therealbitcoin\.org|Natasha):`), kind: "Node (other)"},
	{re: regexp.MustCompile(`(?i)/utreexod:`), kind: "Node (other)"},
	{re: regexp.MustCompile(`(?i)Knots`), kind: "Node (Knots)"},
	{re: regexp.MustCompile(`^/Satoshi:`), kind: "Node (Core)"},
}
