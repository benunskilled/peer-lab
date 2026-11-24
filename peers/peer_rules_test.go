package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The dashboard (app/) judges peers the peers service has no word on - not
// connected, or the service down - by the same rules: the kinds in clients
// and the fork bits in forks.go, read from app/src/data/peer-rules.json. This
// test writes that file from them with UPDATE_PEER_RULES=1 and otherwise fails
// when it is out of date, so the two cannot drift apart.
//
// A pattern goes over as JavaScript source with its flags; only (?i) at the
// start is used here, and anything else of Go's own syntax fails the test
// rather than meaning something else in a browser. Each example carries the
// kind Go gives it, so the dashboard's test can check it reads them the same.

type peerRulesKind struct {
	Kind    string `json:"kind"`
	Pattern string `json:"pattern"`
	Flags   string `json:"flags"`
	// What the pattern is matched against: "whole" the user agent, "bare" the
	// name with the bracketed comment taken out, "no-mempool" with the word
	// "mempool" taken out (pool nodes).
	Reads  string `json:"reads"`
	Relays bool   `json:"relays"`
}

type peerRulesFork struct {
	Service     string `json:"service"`
	Fork        string `json:"fork"`
	ExceptAgent string `json:"exceptAgent,omitempty"`
}

type peerRules struct {
	Note      string            `json:"note"`
	EmptyName string            `json:"emptyName"`
	Bracketed string            `json:"bracketed"`
	Kinds     []peerRulesKind   `json:"kinds"`
	ForkKinds []string          `json:"forkKinds"`
	ForkSlack int               `json:"forkTipSlack"`
	ForkBits  []peerRulesFork   `json:"forkBits"`
	Examples  map[string]string `json:"examples"`
}

func jsPattern(t *testing.T, re string) (string, string) {
	flags := ""
	if strings.HasPrefix(re, "(?i)") {
		flags, re = "i", re[4:]
	}
	if strings.Contains(re, "(?") || strings.Contains(re, `\z`) || strings.Contains(re, `\A`) || strings.Contains(re, "[[:") {
		t.Fatalf("pattern %q uses Go syntax the dashboard cannot read", re)
	}
	return re, flags
}

// kindAgents are awkward user agents on top of aboutExamples, carried over
// from the dashboard's old comparison with kinds.go: a word in an operator's
// brackets, "mempool" in a pool rule, misspelt Core names, names with no
// source. Their kinds go into the file as examples too.
var kindAgents = []string{
	"/ckpool/", "/ckp2p:1.0/", "/Satoshi:29.1.0(PyBLOCK-POOL)/Knots:20250903/",
	"/Satoshi:28.0.0(mempool.space)/", "/mempool/", "/mempool.space:1.0/", "/Bitcoin ABC:28.0.0/",
	"/BUCash:1.9.0/", "/Bitcoin SV:1.1.0/", "/BCHUnlimited:1.10.0/", "/Bitcoin XT:0.11.0/",
	"/bitcoin@dsn.tm.kit.edu/", "/dsn.kastel:1/", "/node.cs.ac.uk:1.0/", "/uni-goettingen.de/",
	"/Satoshi:25.0.0(node.cs.ac.uk)/", "/Satoshi:29.0.0(my-monitor)/",
	"/Satoshi:29.0.0(electrum-backend)/", "/Satoshi:28.0.0(neutrino)/", "/Satoshi:29.0.0(lit:v1)/",
	"/bitnodes.io:0.1/", "/metrika:1.0/", "/nodemap/", "/some-crawler:2/", "/portscanner/",
	"/electrs:0.10.0/", "/electrumx:1.16/", "/esplora/", "/bitcoinj:0.15/", "/breadwallet:1.0/",
	"/bither/", "/MultiBit:0.5/", "/wasabi:2.0/", "/Bitcoin Wallet:9.0/", "/WalletKit:0.1.0/",
	"/bread:2.1/", "/btcwire:0.5.0/spvwallet:0.1.0/", "/btcwire:0.4.1/lit:v0.1/", "/nakamoto:0.3.0/",
	"/btcwire:0.4.1/Satoshi:0.19/", "/neutrino:0.2.0/", "/neutrino-checker:0.1/",
	"/btcwire:0.5.0/btcpost-neutrino:0.17.1/", "/Rust BIP-157:0.6.0/rust-bitcoin:0.32.8/",
	"/bitcoin-seeder:0.01/", "/btc-range-scan:0.1.0/", "/BTC-Nodes:2026-09-24/Sonar/",
	"/census:0.1.7/", "/btc-node-observatory:0.1.0/", "/go-bitnode-monitor:firstseen/",
	"/atlas-scout:0.1/", "nebula/2.4.1-85b3e43", "/Sat0shi:31.0.0/", "/Satoshi2:0.18.3/",
	"/SatoshiX:0.18.0/", "Satoshi:22.0.0", "/Classic:1.3.4(EB8)/",
	"/BitcoinUnlimited:1.0.3(EB16; AD12)/", "/SuperBitcoin:0.16.0.2/", "/iriumd:1.9.66/",
	"vds_0.9.53", "VDS 0.9.9.1", "/your-fork:0.1/", "/btcwire:0.5.0/CaribouSPV:0.0.1-beta/",
	"/NodeWatch-BTC:0.2/", "/aver-btc-listener:0.1/", "bitdash.io", "/BitArgus:0.1/",
	"/btcwire:0.5.0/btcd:0.25.0/", "/libbitcoin:4.0.0/", "/bcoin:v1.0.0-beta.14/",
	"/Gocoin:1.10.0pre/", "/Floresta:0.9.1/", "/Floresta:0.9.1/mandacaru:0.15.2/",
	"/BitcoinRocks:31.1.1/", "/bip110-observer:0.1/", "/EMPORION:0.3/", "/SYMMARTYS:0.3/",
	"/Bitflash:btc-light/", "/21datalab-crawler:0.1/", "nebula/2.4.1-65580d4",
	"/nebula:v3.0.0-20261019105927-e59406bbe71e/", "/cryptexplorer:0.1.0/", "/BlockSail:0.1/",
	"/trinetra:0.1/", "/multiseed:0.1/", "/topology-probe:0.1/", "/p2p-probe:0.1/", "/probe:1.0/",
	"/btc-peers:0.1/", "/:crawler:1/", "/:monitor/", "/:electrs/", "/BTC-Nodes:1.0/Probe/",
	"/bitcoin-peers:0.1.0/", "/logos-peer-list:0.1/", "/nodes.mom.market:0.1/",
	"/btcwire:0.5.0/bitcoinstats.com:0.1/", "/p2p-extractor:0.1/", "/bip157-fetch:0.1/",
	"/dexcash-nodes:0.1.0/", "/Satoshi:29.0.0/", "/Satoshi:28.1.0/Knots:20250903/", "/btcwire:0.5.0/",
	"/Satoshi:27.0.0(mempool.guide)/electrs:0.10.0/", "/btc-mempool:1/", "/Sat0shi:31.0.0/crawler:1/",
	"/CKCoinD:1.0/",
}

func buildPeerRules(t *testing.T) peerRules {
	out := peerRules{
		Note:      "Written by peers/peer_rules_test.go (UPDATE_PEER_RULES=1 go test -run TestPeerRulesFile) from peers/clients.go and peers/forks.go. Do not edit.",
		EmptyName: emptyName.String(),
		Bracketed: bracketed.String(),
		ForkSlack: forkTipSlack,
		Examples:  map[string]string{},
	}
	seen := map[string]bool{"Other": true}
	out.ForkKinds = []string{"Other"}
	for _, r := range kindRules {
		p, f := jsPattern(t, r.re.String())
		reads := "whole"
		switch {
		case r.name == "Pool node":
			reads = "no-mempool"
		case readsBareName[r.name] && !strings.Contains(r.re.String(), `\(`):
			reads = "bare"
		}
		out.Kinds = append(out.Kinds, peerRulesKind{r.name, p, f, reads, r.relays})
		if !seen[r.name] && forkApplies(r.name) {
			out.ForkKinds = append(out.ForkKinds, r.name)
		}
		seen[r.name] = true
	}
	for _, b := range forkBits {
		out.ForkBits = append(out.ForkBits, peerRulesFork{b.service, b.fork, b.exceptAgent})
	}
	for _, e := range aboutExamples {
		out.Examples[e.ua] = kindOf(e.ua)
	}
	for _, ua := range kindAgents {
		out.Examples[ua] = kindOf(ua)
	}
	return out
}

func TestPeerRulesFile(t *testing.T) {
	path := filepath.Join("..", "app", "src", "data", "peer-rules.json")
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(buildPeerRules(t)); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("UPDATE_PEER_RULES") == "1" {
		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	have, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Skip("no app/ next to peers/ (peers built on its own)")
	}
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(have, buf.Bytes()) {
		t.Fatal("app/src/data/peer-rules.json is out of date: run UPDATE_PEER_RULES=1 go test -run TestPeerRulesFile in peers/")
	}
}
