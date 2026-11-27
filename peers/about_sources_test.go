package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// A plain about sentence - no "Probably", no "Named like" - says what the
// software is, so the place that proves it has to sit right above the rule,
// in one form:
//
//	// Source: owner/repo path Lnn @commit-or-tag (checked YYYY-MM-DD)
//
// The monthly check reads these lines back; see README's "How sure a label is".
func TestEveryPlainAboutRuleHasASource(t *testing.T) {
	b, err := os.ReadFile("clients.go")
	if err != nil {
		t.Fatal(err)
	}
	rule := regexp.MustCompile("^\\s*\\{re: regexp\\.MustCompile\\(`(.*?)`\\)(?:, kind: \"[^\"]*\")?, text: \"((?:[^\"\\\\]|\\\\.)*)\"")
	source := regexp.MustCompile(`^\s*// Source: \S+/\S+ \S+ L\d+(-\d+)? @\S+ \(checked \d{4}-\d{2}-\d{2}\)$`)
	lines := strings.Split(string(b), "\n")
	plain, missing := 0, 0
	for i, l := range lines {
		m := rule.FindStringSubmatch(l)
		if m == nil || strings.HasPrefix(m[2], "Probably") || strings.HasPrefix(m[2], "Named like") {
			continue
		}
		plain++
		found := false
		for j := i - 1; j >= 0 && strings.HasPrefix(strings.TrimSpace(lines[j]), "//"); j-- {
			if source.MatchString(lines[j]) {
				found = true
				break
			}
		}
		if !found {
			missing++
			t.Errorf("clients.go:%d: %q says plainly what it is but has no // Source: line above it", i+1, m[1])
		}
	}
	if plain < 150 {
		t.Fatalf("only %d plain rules found - is the rule pattern in this test out of date?", plain)
	}
	// Malformed source lines are as good as none.
	for i, l := range lines {
		if strings.Contains(l, "// Source:") && !source.MatchString(l) {
			t.Errorf("clients.go:%d: source line not in the agreed form: %s", i+1, strings.TrimSpace(l))
		}
	}
}
