package format

import (
	"strings"
	"testing"
)

// @covers AC-TASKS-PLAN-BOARD-OPS-007.3
func TestNewPlanRoundTrip(t *testing.T) {
	titles := []string{
		"Simple", "Fix: colons # and hashes", `He said "hi" and \ left`, "Ünïcödé 计划 日本語",
		"tab\there", "- dash", "'single'", "true", "123", "null", "multi\nline", "emoji \U0001F600",
	}
	bodies := []string{"", "Some body.\n", "no trailing newline", "- [ ] item\n\n## Section\ntext\n"}
	for _, title := range titles {
		for _, body := range bodies {
			out := NewPlan(NewPlanInput{Title: title, Priority: "high", Executor: `co"dex: #1`, Body: body})
			pf, ok := Parse("x.md", out)
			if !ok {
				t.Fatalf("not a plan file: %q", out)
			}
			if len(pf.ParseErrors) != 0 {
				t.Fatalf("parse errors %v for %q", pf.ParseErrors, out)
			}
			if pf.Board != BoardQueued || pf.Title != title || pf.Priority != "high" || pf.Executor != `co"dex: #1` {
				t.Fatalf("round trip mismatch for %q: %+v", title, pf)
			}
			wantHeading := "# " + strings.ReplaceAll(title, "\n", " ") + "\n"
			if !strings.HasPrefix(pf.Body, wantHeading) {
				t.Fatalf("body %q lacks heading %q", pf.Body, wantHeading)
			}
			rest := strings.TrimPrefix(pf.Body, wantHeading)
			if body == "" && rest != "" {
				t.Fatalf("unexpected body %q", rest)
			}
			if body != "" && strings.TrimPrefix(rest, "\n") != strings.TrimSuffix(body, "\n")+"\n" {
				t.Fatalf("body %q, want %q", rest, body)
			}
		}
	}
}

func TestNewPlanDefaultsAndOmissions(t *testing.T) {
	out := string(NewPlan(NewPlanInput{Title: "  T  "}))
	want := "---\nboard: queued\ntitle: \"T\"\npriority: medium\n---\n# T\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
	if strings.Contains(out, "executor") {
		t.Fatal("empty executor must be omitted")
	}
}

// @covers AC-TASKS-PLAN-BOARD-OPS-005.2
func TestWorkOrderDone(t *testing.T) {
	cases := map[string]bool{
		"---\nstatus: done\n---\n":            true,
		"---\nstatus: \"done\"\n---\n":        true,
		"---\nstatus: in_progress\n---\n":     false,
		"---\nstatus: [done]\n---\n":          false,
		"---\ntitle: x\n---\n":                false,
		"status: done\n":                      false,
		"---\nstatus: done\n":                 false,
		"---\nstatus: Done\n---\n":            false,
		"---\r\nstatus: done\r\n---\r\nb\r\n": true,
	}
	for content, want := range cases {
		if got := WorkOrderDone([]byte(content)); got != want {
			t.Errorf("WorkOrderDone(%q) = %v, want %v", content, got, want)
		}
	}
}
