package format

import (
	"strings"
	"testing"
)

// @covers AC-TASKS-PLAN-CARD-001.1
func TestParseDate(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		want    string
		invalid bool
	}{
		{"absent", "", "", false},
		{"plain", "date: 2026-10-12", "2026-10-12", false},
		{"quoted", "date: '2026-10-12'", "2026-10-12", false},
		{"padded", "date: '  2026-10-12 '", "2026-10-12", false},
		{"leap day", "date: 2028-02-29", "2028-02-29", false},
		{"null", "date:", "", false},
		{"empty string", "date: ''", "", false},
		{"impossible day", "date: 2026-02-30", "", true},
		{"not a leap year", "date: 2026-02-29", "", true},
		{"month 13", "date: 2026-13-01", "", true},
		{"unpadded", "date: 2026-1-5", "", true},
		{"slashes", "date: 2026/10/12", "", true},
		{"day first", "date: 12.10.2026", "", true},
		{"with time", "date: 2026-10-12T10:00:00Z", "", true},
		{"word", "date: tomorrow", "", true},
		{"list", "date: [2026-10-12]", "", true},
		{"mapping", "date: {a: b}", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pf, ok := Parse("p.md", []byte("---\nboard: queued\n"+c.line+"\n---\n"))
			if !ok {
				t.Fatal("expected plan file")
			}
			if pf.Date != c.want {
				t.Fatalf("date = %q, want %q", pf.Date, c.want)
			}
			got := len(pf.ParseErrors) == 1 && strings.Contains(pf.ParseErrors[0], "invalid_date")
			if got != c.invalid || (!c.invalid && len(pf.ParseErrors) != 0) {
				t.Fatalf("errors = %v, invalid = %v", pf.ParseErrors, c.invalid)
			}
		})
	}
}
