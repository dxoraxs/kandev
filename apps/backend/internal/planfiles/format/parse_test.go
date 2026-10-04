package format

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseNotPlanFile(t *testing.T) {
	cases := map[string]string{
		"no frontmatter":        "# Title\nbody\n",
		"empty":                 "",
		"no closing fence":      "---\nboard: queued\n# Title\n",
		"no board key":          "---\ntitle: x\n---\nbody\n",
		"invalid yaml":          "---\nboard: [unclosed\n---\nbody\n",
		"not a mapping":         "---\n- a\n- b\n---\n",
		"fence not on line one": "\n---\nboard: queued\n---\n",
		"empty frontmatter":     "---\n---\nbody\n",
		"duplicate key":         "---\nboard: done\nboard: queued\n---\n",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			if _, ok := Parse("p.md", []byte(content)); ok {
				t.Fatalf("expected ok=false for %q", content)
			}
		})
	}
}

// @covers AC-TASKS-PLAN-FILES-001.1
// @covers AC-TASKS-PLAN-FILES-001.2
func TestParseValidFile(t *testing.T) {
	content := "---\nboard: In_Progress \ntitle: My plan\npriority: HIGH\norder: 12.5\n" +
		"executor: ' codex '\ndepends_on:\n  - a.md\n  - b.md\nexternal_id: ' ext-1 '\nunknown: 3\n---\n# Heading\nbody\n"
	pf, ok := Parse("plan.md", []byte(content))
	if !ok {
		t.Fatal("expected plan file")
	}
	if pf.Board != BoardInProgress || pf.Title != "My plan" || pf.Priority != "high" ||
		pf.Executor != "codex" || pf.ExternalID != "ext-1" {
		t.Fatalf("unexpected fields: %+v", pf)
	}
	if pf.Order == nil || *pf.Order != 12.5 {
		t.Fatalf("order = %v", pf.Order)
	}
	if !reflect.DeepEqual(pf.DependsOn, []string{"a.md", "b.md"}) {
		t.Fatalf("depends_on = %v", pf.DependsOn)
	}
	if pf.Body != "# Heading\nbody\n" {
		t.Fatalf("body = %q", pf.Body)
	}
	if len(pf.ParseErrors) != 0 {
		t.Fatalf("unexpected errors: %v", pf.ParseErrors)
	}
	if pf.Hash != ContentHash([]byte(content)) || len(pf.Hash) != 64 {
		t.Fatalf("hash = %q", pf.Hash)
	}
}

func TestParseDefaults(t *testing.T) {
	pf, ok := Parse("plan.md", []byte("---\nboard: queued\n---\n"))
	if !ok || pf.Priority != "medium" || pf.Order != nil || pf.DependsOn != nil || pf.Body != "" {
		t.Fatalf("ok=%v pf=%+v", ok, pf)
	}
}

// @covers AC-TASKS-PLAN-FILES-001.3
func TestParseTitleFallback(t *testing.T) {
	cases := []struct {
		name, content, want string
	}{
		{"frontmatter wins", "---\nboard: queued\ntitle: FM\n---\n# H\n", "FM"},
		{"heading", "---\nboard: queued\n---\n\nintro\n# First\n# Second\n", "First"},
		{"heading CRLF", "---\r\nboard: queued\r\n---\r\n# First\r\n", "First"},
		{"heading in code fence skipped", "---\nboard: queued\n---\n```\n# no\n```\n# yes\n", "yes"},
		{"level two ignored", "---\nboard: queued\n---\n## Two\n", "my-plan"},
		{"stem", "---\nboard: queued\n---\nbody\n", "my-plan"},
		{"empty title falls back", "---\nboard: queued\ntitle: ''\n---\n# H\n", "H"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pf, ok := Parse("dir/my-plan.md", []byte(c.content))
			if !ok || pf.Title != c.want {
				t.Fatalf("ok=%v title=%q want %q", ok, pf.Title, c.want)
			}
		})
	}
}

// @covers AC-TASKS-PLAN-FILES-001.4
func TestParseInvalidValues(t *testing.T) {
	long := strings.Repeat("x", 41)
	cases := []struct {
		name, fm, errKey string
		check            func(PlanFile) bool
	}{
		{"board unknown", "board: nope", "board", func(p PlanFile) bool { return p.Board == "" }},
		{"board list", "board: [a]", "board", func(p PlanFile) bool { return p.Board == "" }},
		{"board empty", "board:", "board", func(p PlanFile) bool { return p.Board == "" }},
		{"priority", "board: done\npriority: urgent", "priority", func(p PlanFile) bool { return p.Priority == "medium" }},
		{"order text", "board: done\norder: abc", "order", func(p PlanFile) bool { return p.Order == nil }},
		{"order nan", "board: done\norder: .nan", "order", func(p PlanFile) bool { return p.Order == nil }},
		{"order list", "board: done\norder: [1]", "order", func(p PlanFile) bool { return p.Order == nil }},
		{"executor long", "board: done\nexecutor: " + long, "executor", func(p PlanFile) bool { return p.Executor == "" }},
		{"depends scalar", "board: done\ndepends_on: a.md", "depends_on", func(p PlanFile) bool { return p.DependsOn == nil }},
		{"depends empty entry", "board: done\ndepends_on: ['']", "depends_on", func(p PlanFile) bool { return p.DependsOn == nil }},
		{"external control", "board: done\nexternal_id: \"a\\tb\"", "external_id", func(p PlanFile) bool { return p.ExternalID == "" }},
		{"external long", "board: done\nexternal_id: " + strings.Repeat("e", 256), "external_id", func(p PlanFile) bool { return p.ExternalID == "" }},
		{"title map", "board: done\ntitle: {a: b}", "title", func(p PlanFile) bool { return true }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pf, ok := Parse("p.md", []byte("---\n"+c.fm+"\n---\n"))
			if !ok {
				t.Fatal("expected ok=true")
			}
			if len(pf.ParseErrors) != 1 || !strings.HasPrefix(pf.ParseErrors[0], c.errKey+":") {
				t.Fatalf("errors = %v", pf.ParseErrors)
			}
			if !c.check(pf) {
				t.Fatalf("invalid field not zero: %+v", pf)
			}
		})
	}
}

func TestParseValidSiblingsSurviveInvalidField(t *testing.T) {
	pf, ok := Parse("p.md", []byte("---\nboard: done\npriority: nope\norder: 2\nexecutor: x\n---\n"))
	if !ok || pf.Board != BoardDone || pf.Order == nil || *pf.Order != 2 || pf.Executor != "x" {
		t.Fatalf("ok=%v pf=%+v", ok, pf)
	}
}

func TestParseEncodingVariants(t *testing.T) {
	cases := map[string]string{
		"CRLF":         "---\r\nboard: done\r\ntitle: T\r\n---\r\nbody\r\n",
		"BOM":          "\xef\xbb\xbf---\nboard: done\ntitle: T\n---\nbody\n",
		"BOM CRLF":     "\xef\xbb\xbf---\r\nboard: done\r\ntitle: T\r\n---\r\nbody\r\n",
		"no final EOL": "---\nboard: done\ntitle: T\n---",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			pf, ok := Parse("p.md", []byte(content))
			if !ok || pf.Board != BoardDone || pf.Title != "T" || len(pf.ParseErrors) != 0 {
				t.Fatalf("ok=%v pf=%+v", ok, pf)
			}
		})
	}
}

func TestParseAllBoardStatuses(t *testing.T) {
	for _, s := range []string{"queued", "in_progress", "waiting_owner", "waiting_external", "deferred", "done", "hidden"} {
		pf, ok := Parse("p.md", []byte("---\nboard: "+strings.ToUpper(s)+"\n---\n"))
		if !ok || string(pf.Board) != s || len(pf.ParseErrors) != 0 {
			t.Fatalf("%s: ok=%v pf=%+v", s, ok, pf)
		}
	}
}
