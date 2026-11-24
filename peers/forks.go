package main

import (
	"slices"
	"strings"
	"time"
)

// Libre Relay (github.com/petertodd/bitcoin, branches libre-relay-v28.0 to
// v31.1) is Core under Core's own name with service bit 29 added
// (NODE_LIBRE_RELAY in protocol.h). Core lists that bit as "UNKNOWN[2^29]".
// Libre Relay opens extra connections to nodes advertising it and drops them
// at once when the bit is missing ("does not offer libre relay as expected").
// Garbageman (github.com/chrisguida/bitcoin, branches garbageman-v28,
// garbageman-v29, garbageman-29.3) is a Knots fork with the Knots part of
// its name commented out, so it says "/Satoshi:29.1.0/" or 29.3.0, and it
// advertises the same bit to draw Libre Relay nodes in and drop what they
// send. Libre Relay has no 29.3 and Garbageman nothing past 29, so the
// version tells them apart except up to 29.1. On the node this was written
// for, 21. to 24.10.2026: 21 connections with the bit from 12 addresses,
// 18 of them as 30.2.0, all gone straight after the version message.
const libreRelayService = "UNKNOWN[2^29]"

// libreRelay is the kind and about for a Core name offering bit 29, or "", "".
func libreRelay(subver string, services []string) (kind, about, label string) {
	if !slices.Contains(services, libreRelayService) {
		return "", "", ""
	}
	maj, mnr, ok := claimedNumbers(subver)
	if !ok {
		return "", "", ""
	}
	garbageman := "Claims: Libre Relay\nProbably: Garbageman - a Knots fork that poses as Libre Relay and drops what it gets"
	// Libre Relay and Garbageman nodes start from NETWORK_LIMITED | WITNESS
	// | bit 29 (g_local_services in init.cpp, libre-relay-v31.2 and
	// garbageman-v29), so the bit without NETWORK_LIMITED is a tool looking
	// for them. Two send exactly NETWORK, WITNESS and bit 29 under a fixed
	// name: tx-pigeon pushes transactions to such nodes as
	// "/Satoshi:27.0.0/" (github.com/stutxo/tx-pigeon, src/main.rs,
	// build_version_msg), Garbageman Nodes Manager looks for them as
	// "/Satoshi:29.1.0/" (github.com/paulscode/garbageman-nm,
	// webui/api/src/services/peer-discovery.ts, SERVICES_GARBAGEMAN). The
	// missing service itself is told by anachronism, as an "Also:".
	if !slices.Contains(services, "NETWORK_LIMITED") {
		exact := len(services) == 3 && slices.Contains(services, "NETWORK") && slices.Contains(services, "WITNESS")
		tool := "a tool looking for Libre Relay nodes"
		switch {
		case exact && strings.HasPrefix(subver, "/Satoshi:27.0.0/"):
			tool = "tx-pigeon, a tool that pushes transactions to Libre Relay nodes"
		case exact && strings.HasPrefix(subver, "/Satoshi:29.1.0/"):
			tool = "Garbageman Nodes Manager looking for Libre Relay nodes"
		}
		return fakeWalletKind, "Claims: Libre Relay\nProbably: " + tool, ""
	}
	switch {
	case strings.Contains(subver, "Knots:"):
		// Libre Relay is Core; only a Garbageman built with its Knots name left in says Knots.
		return fakeWalletKind, garbageman, ""
	case maj >= 30:
		return "Node (Libre Relay)", "Libre Relay, a fork of Core that also relays transactions Core and Knots filter out", "Libre Relay (Core fork)"
	case maj == 29 && mnr >= 2:
		return fakeWalletKind, garbageman, ""
	}
	return "Node (Libre Relay)", "Libre Relay - or Garbageman, a Knots fork that poses as it", "Libre Relay or Garbageman"
}

// Service bits Knots gives to rules other than Bitcoin's (src/protocol.h;
// bit 28 in its releases since 29.4.1.knots20260508, 02.09.2026, whose notes
// say "knots#359 Hardfork: New BLAKE2b proof-of-work algorithm" - every
// Knots since follows that chain; Oracle Knots sets bit 28 too, MarcanoFilms/oracle-knots): bit 27 NODE_REDUCED_DATA, a node enforcing BIP110's
// ReducedData rules, bit 28 NODE_BLAKE2B, a node enforcing a hard fork to
// BLAKE2b proof of work (and ReducedData). Core lists them as UNKNOWN[2^n].
// Source: bitcoinknots/bitcoin src/protocol.h L351-355 @v29.5.1.knots20260508 (checked 2026-11-24)
// In peer-observer's record of the node this was written for, 21. to
// 25.10.2026, with that node at height 973,026: 898 sessions offering bit 28
// from 406 addresses reported start heights around 978,000, a chain of their
// own; 2,719 offering bit 27 from 921 addresses reported around 961,000, the
// BIP110 chain, which went its own way. Neither can deliver a block of this
// node's chain, so both count as never first. The
// node had also dialled 33 and 18 of them itself, each holding an outbound
// slot for 40 to 65 seconds at the median, up to 22 minutes. Crawlers repeat
// bits they saw elsewhere (/bitcoin-seeder:0.01/ sent both), so only nodes and
// unknown software count.
// forkTipSlack is how many blocks behind the tip a peer's headers may be and
// still count as this chain: one block found while the answer was on its way,
// with room to spare.
const forkTipSlack = 6

const (
	reducedDataService = "UNKNOWN[2^27]"
	blake2bService     = "UNKNOWN[2^28]"
	purityService      = "UNKNOWN[2^25]"
)

// shownServices is the peer's services for the panel, with the bits Core only
// knows as "UNKNOWN[2^n]" named after the software that defines them (checked
// 23.11.2026): 12 and 13 BIP 183's Utreexo bits (bip183/protocol.go in
// utreexo/utreexod; Castor sets 12 too), 25 Bitcoin Purity's, 27 BIP110's,
// 28 Knots' BLAKE2b - or, from satd, NODE_INTACT_BLOCKS (node/src/net/peer.rs
// in epochbtc/satd) - 29 Libre Relay's, 30 satd's NODE_NAP (nap/p2p.rs).
// Only for showing: forkOf and the disguise rules read Core's names.
func shownServices(subver string, services []string) []string {
	if services == nil {
		return nil
	}
	satd := strings.HasPrefix(strings.ToLower(subver), "/satd:")
	names := map[string]string{
		"UNKNOWN[2^12]": "UTREEXO", "UNKNOWN[2^13]": "UTREEXO_ARCHIVE",
		purityService: "PURITY_ASERT", reducedDataService: "REDUCED_DATA",
		blake2bService: "BLAKE2B", libreRelayService: "LIBRE_RELAY",
	}
	if satd {
		names[blake2bService] = "INTACT_BLOCKS"
		names["UNKNOWN[2^30]"] = "NAP"
	}
	out := make([]string, len(services))
	for i, s := range services {
		if n, ok := names[s]; ok {
			s = n
		}
		out[i] = s
	}
	return out
}

// Bitcoin Purity offers bit 25 (NODE_PURITY_ASERT) beside bit 27 by default
// (g_local_services in src/init.cpp, github.com/saltduck/bitcoinpurity at
// 833e2f6), and only it defines bit 25 (src/protocol.h). With -uaspoof=1 its
// name shrinks to "/Satoshi:29.4/" and the bits are all that is left of it.
func purityByBits(subver string, services []string) bool {
	return strings.HasPrefix(subver, "/Satoshi:") && !strings.Contains(subver, "/Purity:") &&
		slices.Contains(services, purityService) && slices.Contains(services, reducedDataService)
}

// Checked 23.11.2026: -uaspoof=1 calls FormatSubVersion with base_name_only
// (src/init.cpp L1895-1899), which leaves "Purity:" out (src/clientversion.cpp
// L77-79); the bits stay (g_local_services, src/init.cpp L1033; bit 25 and 27
// in src/protocol.h L351, L356).
const purityByBitsAbout = "Bitcoin Purity under a bare Core name (its -uaspoof=1): only Purity offers service bit 25, here beside BIP110's bit 27"

// forkOf is the flag and its hover text for a peer offering one of those
// bits, or "", "". A peer whose headers reach this node's tip follows this
// chain whatever it offers, and is not flagged: in the measured record of the
// node this was written for, 3,945 of 3,947 sessions with bit 27 shared no
// header past about 961,000 while the node was thousands of blocks further;
// the other two, a home-built node, were at the tip. synced is the peer's
// synced_headers (nil or -1 while unknown), tip the highest of any peer.
//
// satd gives bit 28 a meaning of its own, NODE_INTACT_BLOCKS: a pruned node
// that still serves every intact block (-pruneopreturnpeers=intact;
// node/src/net/peer.rs L7-13 in epochbtc/satd, release notes 0.7.0, checked
// 23.11.2026). From a peer saying "/satd:" the bit is that, not BLAKE2b.
func forkOf(subver string, services []string, kind string, synced *int64, tip int64) (string, string) {
	if kind != "Other" && !strings.HasPrefix(kind, "Node") {
		return "", ""
	}
	if synced != nil && *synced >= 0 && tip > 0 && tip-*synced <= forkTipSlack {
		return "", ""
	}
	has := map[string]bool{}
	for _, s := range services {
		has[s] = true
	}
	switch {
	case has[blake2bService] && !strings.HasPrefix(strings.ToLower(subver), "/satd:"):
		return "BLAKE2b fork", "Offers Knots' BLAKE2b bit (service bit 28): it follows a hard fork to another proof of work, so it is probably on a chain of its own, not yours"
	case has[reducedDataService]:
		return "BIP110 fork", "Offers Knots' ReducedData bit (service bit 27): it enforces BIP110's rules, which probably put it on a chain of its own, not yours"
	}
	return "", ""
}

// Syncing: a peer that takes whole blocks from this node faster than the chain
// grows, and has not caught up, is a node still syncing - one just set up, or
// one started again. A block every ten minutes is 0.16 MB a minute. On one
// node, 21. to 26.10.2026, 418 inbound peers took 20 MB or more; 365 of them at
// 1 MB a minute or faster over their whole stay (median 5). The average over
// the whole stay is all a single look at getpeerinfo gives, so a peer that has
// reached the tip is left out by what Core says it has: the last block it
// shares with this node is within syncBehind blocks (a day) of the tip. Seen
// 26.10.2026: one peer 6 blocks behind that had taken 11.9 GB in 2.5 days at
// 3 MB a minute, another 27 behind with 3.8 GB - nodes at the tip that fetch
// blocks all the time - and three genuine ones with no common block yet.
const (
	syncMinBytes  = 20_000_000
	syncMinPerMin = 1_000_000.0
	syncBehind    = 144
)

// syncingFrom is how many megabytes of blocks an inbound peer has taken, in this session, from
// this node while it syncs, or 0. sent is bytessent_per_msg, synced is
// synced_blocks (nil or below zero: Core does not know), tip the highest
// header any peer shares.
func syncingFrom(inbound bool, sent map[string]int64, synced *int64, tip, connTime int64, now time.Time) int64 {
	b := sent["block"]
	if !inbound || b < syncMinBytes || connTime <= 0 {
		return 0
	}
	if synced != nil && *synced >= 0 && tip > 0 && tip-*synced < syncBehind {
		return 0
	}
	secs := now.Unix() - connTime
	if secs < 1 || float64(b)/(float64(secs)/60) < syncMinPerMin {
		return 0
	}
	return b / 1_000_000
}
