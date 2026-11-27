package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// What kind of software a peer runs, read off the user agent it sent in its
// version message.
//
// This is what the peer SAYS it is, and a user agent is a free-text field the
// other side chooses. On the node this was written for, four connections
// announced themselves as "/Sat0shi:31.0.0/" - Bitcoin Core with a zero in
// place of the o - from rented machines in five regions. So the kind is a
// label, not evidence.
//
// The counterweight is in the three observed flags on Peer, which nobody can
// write into a string: what the peer offers, whether it wants transactions,
// and whether Core has ever learned which chain it is on. A peer calling
// itself Core while offering nothing and having no known chain is the
// interesting case, and it is visible only because the two are shown side by
// side.
//
// The name answers the question somebody actually has, which is not "what
// software is this" but "is this a node that could hand me a block". So the
// class comes first and the software after it: "Node (Core)", not "Core".
//
// Order matters: the first match wins, so a more specific pattern goes above
// a more general one (Knots before Core - a Knots user agent contains
// "Satoshi" too).
//
// relays says whether this kind of software passes blocks on at all. It is the
// same distinction the dashboard draws for "never first" in its ranking, and
// the two have to agree: a mark that means two different things on one page
// is worse than no mark.
//
// A wallet has nothing to relay, a crawler is there to look, an indexer serves
// somebody else's wallet. Measured across 2,270 peers on one node: 688 of them
// ran software like this, 790 observations between them, and not one block
// delivered first. Unknown software counts as relaying - not knowing is not the
// same as knowing it cannot.
var kindRules = kindRulesOf(clients)

var coreNameWithSuffix = regexp.MustCompile(`^/(Satoshi[0-9X]+):`)

// How each spelling misspeltCore catches begins its reason. tell.go gives
// every one of them the same short form, from this list.
const (
	spelledZero    = "it writes Sat0shi with a zero"
	spelledSuffix  = "Core calls itself /Satoshi:"
	spelledSlashes = "Core writes its name between slashes"
	spelledOpenEnd = "Core always ends its name with"
)

var misspellings = []string{spelledZero, spelledSuffix, spelledSlashes, spelledOpenEnd}

// misspeltCore says how a name imitating Core's differs from it, for the
// four spellings the Disguised rule catches, or "".
func misspeltCore(subver string) string {
	switch {
	case strings.HasPrefix(subver, "/Sat0shi:"):
		return spelledZero
	case coreNameWithSuffix.MatchString(subver):
		return spelledSuffix + ", this one /" + coreNameWithSuffix.FindStringSubmatch(subver)[1] + ":"
	case strings.HasPrefix(subver, "Satoshi:"):
		return spelledSlashes + ": /" + strings.Trim(subver, "/") + "/"
	case strings.HasPrefix(subver, "/Satoshi:") && !strings.Contains(subver[1:], "/"):
		return spelledOpenEnd + " \"/\": " + subver + "/"
	}
	return ""
}

var coreVersion = regexp.MustCompile(`[Ss]at[o0]shi[0-9X]*:([0-9.]+)`)

var claimedCore = regexp.MustCompile(`^/Satoshi:([0-9]+)\.([0-9]+)`)

// coreRelease reads a user agent that names a real Core release number:
// 0.x up to 0.21, then 22 and later. "/Satoshi:1.14.5(2x)/" (btc1) and other
// numbers Core never used return ok == false and are left alone.
func coreRelease(subver string) (maj, mnr int, ok bool) {
	maj, mnr, ok = claimedNumbers(subver)
	return maj, mnr, ok && (maj == 0 || maj >= 22)
}

// claimedNumbers is the major and minor number after "/Satoshi:", whether or
// not Core ever had that version.
func claimedNumbers(subver string) (maj, mnr int, ok bool) {
	m := claimedCore.FindStringSubmatch(subver)
	if m == nil {
		return 0, 0, false
	}
	maj, _ = strconv.Atoi(m[1])
	mnr, _ = strconv.Atoi(m[2])
	return maj, mnr, true
}

func coreAtLeast(subver string, major, minor int) bool {
	maj, mnr, ok := coreRelease(subver)
	return ok && (maj > major || maj == major && mnr >= minor)
}

// devBuild is a Core development build: 0.20.99 (0.21 under way) or 31.99.0.
var devBuild = regexp.MustCompile(`^/Satoshi:[0-9]+\.([0-9]+\.)?99[./]`)

// noAgentKind is a peer that has not told its name. On one node that was
// 21,830 sessions in a fortnight, from 2,516 addresses, typically gone after
// 16 seconds: connections that only checked the port, or that are still
// being set up when the page asks.
const noAgentKind = "No name"

// kindOf returns the family name, noAgentKind for a peer that sent no user
// agent at all, and "Other" for one whose agent matches nothing here - which
// is a normal answer, not a failure: the network runs plenty of software this
// list has never heard of.
func kindOf(subver string) string {
	name, _ := classify(subver)
	return name
}

// relaysBlocks is false only for software known not to pass blocks on. Anything
// unrecognised, and anything that sent no user agent at all, counts as relaying.
func relaysBlocks(subver string) bool {
	_, relays := classify(subver)
	return relays
}

// mempoolWord is removed before the kind rules look for "pool", so that a
// "(mempool.guide)" label does not make a node a pool node.
var mempoolWord = regexp.MustCompile(`(?i)mempool`)

var bracketed = regexp.MustCompile(`\([^)]*\)`)

// The kinds that read only the agent outside the brackets. The dashboard's
// test reads this list, so both services agree on it.
var readsBareName = map[string]bool{"Research scanner": true, "Crawler": true, "Indexer": true, "Wallet": true, "Lightning wallet": true, "Other light client": true}

// emptyName is a user agent with nothing in it but slashes, or a version
// without a name before it: "//" came 523 times in three days from two
// addresses, "/:1.1.0-dev+c1771608d/" ten times - nothing said, as with none.
var emptyName = regexp.MustCompile(`^/*$|^/:[^/]*/$`)

func classify(subver string) (string, bool) {
	if subver == "" || emptyName.MatchString(subver) {
		return noAgentKind, true
	}
	// The bracketed comment is the operator's own label. It may say a node
	// belongs to a pool, but a word like "monitor" or "electrs" in it says
	// what the operator runs beside it, not what the peer is: those kinds
	// read only the name outside the brackets.
	bare := bracketed.ReplaceAllString(subver, "")
	for _, r := range kindRules {
		s := subver
		if r.name == "Pool node" {
			s = mempoolWord.ReplaceAllString(subver, "")
		} else if readsBareName[r.name] && !strings.Contains(r.re.String(), `\(`) {
			s = bare
		}
		if r.re.MatchString(s) {
			return r.name, r.relays
		}
	}
	return "Other", true
}

// coreMajor is the release number people say: 0.21 for 0.x, 31 after.
func coreMajor(maj, mnr int) string {
	if maj == 0 {
		return fmt.Sprintf("0.%d", mnr)
	}
	return fmt.Sprint(maj)
}
