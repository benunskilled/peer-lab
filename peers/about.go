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
var aboutRules = aboutRulesOf(clients)

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
