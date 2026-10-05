package format

import (
	"reflect"
	"strings"
	"testing"
)

// @covers AC-TASKS-PLAN-BOARD-OPS-005.1
func TestParseTracks(t *testing.T) {
	many := "---\nboard: queued\ntracks:\n" + strings.Repeat("  - a.md\n", MaxTracks+1) + "---\n"
	atLimit := "---\nboard: queued\ntracks:\n" + strings.Repeat("  - a.md\n", MaxTracks) + "---\n"
	cases := []struct {
		name    string
		content string
		want    []string
		invalid bool
	}{
		{"absent", "---\nboard: queued\n---\n", nil, false},
		{"list", "---\nboard: queued\ntracks:\n  - docs/a.md\n  - ' docs/plans '\n---\n", []string{"docs/a.md", "docs/plans"}, false},
		{"at limit", atLimit, strings.Split(strings.TrimSuffix(strings.Repeat("a.md,", MaxTracks), ","), ","), false},
		{"over limit", many, nil, true},
		{"scalar", "---\nboard: queued\ntracks: docs/a.md\n---\n", nil, true},
		{"empty entry", "---\nboard: queued\ntracks:\n  - a.md\n  - ''\n---\n", nil, true},
		{"nested entry", "---\nboard: queued\ntracks:\n  - [a]\n---\n", nil, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pf, ok := Parse("p.md", []byte(c.content))
			if !ok {
				t.Fatal("expected plan file")
			}
			if !reflect.DeepEqual(pf.Tracks, c.want) {
				t.Fatalf("tracks = %v, want %v", pf.Tracks, c.want)
			}
			if c.invalid != (len(pf.ParseErrors) == 1 && strings.Contains(pf.ParseErrors[0], "invalid_tracks")) {
				t.Fatalf("errors = %v, invalid = %v", pf.ParseErrors, c.invalid)
			}
		})
	}
}
