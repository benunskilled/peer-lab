package main

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"
)

// TestClassifyFile tells, for user agents in a file, how Peer Lab sees each
// by its name alone: its kind, and whether a rule names the software. A tool
// for the research list on the maintainer's machine, kept in a test so it is
// never part of the app: it runs only when CLASSIFY_IN names a file (one user
// agent per line) and writes JSON to CLASSIFY_OUT.
//
//	CLASSIFY_IN=names.txt CLASSIFY_OUT=out.json go test -run TestClassifyFile .
func TestClassifyFile(t *testing.T) {
	in, out := os.Getenv("CLASSIFY_IN"), os.Getenv("CLASSIFY_OUT")
	if in == "" || out == "" {
		t.Skip("CLASSIFY_IN and CLASSIFY_OUT not set")
	}
	f, err := os.Open(in)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	type seen struct {
		Kind  string `json:"kind"`
		Named bool   `json:"named"`
	}
	res := map[string]seen{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		ua := sc.Text()
		_, named := ruleOf(ua)
		res[ua] = seen{Kind: kindOf(ua), Named: named}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
