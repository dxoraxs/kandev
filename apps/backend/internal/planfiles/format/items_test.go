package format

import "testing"

// @covers AC-TASKS-PLAN-BOARD-OPS-005.2
func TestCountItems(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		done, total int
	}{
		{"empty", "", 0, 0},
		{"markers", "- [ ] a\n- [x] b\n* [X] c\n* [ ] d\n", 2, 4},
		{"indentation", "  - [ ] a\n\t- [x] b\n      * [x] c\n", 2, 3},
		{"crlf", "- [x] a\r\n- [ ] b\r\n", 1, 2},
		{"fenced backticks", "- [ ] a\n```\n- [x] b\n- [ ] c\n```\n- [x] d\n", 1, 2},
		{"fenced tildes", "~~~md\n- [x] b\n~~~\n- [ ] a\n", 0, 1},
		{"unclosed fence", "- [ ] a\n```\n- [x] b\n", 0, 1},
		{"not items", "- a\n- [] b\n- [y] c\n-[ ] d\n- [x]e\n+ [ ] f\n1. [ ] g\n[ ] h\n", 0, 0},
		{"end of line marker", "- [ ]\n- [x]", 1, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			done, total := CountItems(c.body)
			if done != c.done || total != c.total {
				t.Fatalf("got %d/%d, want %d/%d", done, total, c.done, c.total)
			}
		})
	}
}

// @covers AC-TASKS-PLAN-BOARD-OPS-005.2
func TestBodyOf(t *testing.T) {
	cases := []struct{ name, content, want string }{
		{"with frontmatter", "---\nboard: queued\nlist:\n  - [x]\n---\n- [ ] a\n", "- [ ] a\n"},
		{"bom", "\xef\xbb\xbf---\nk: v\n---\nbody\n", "body\n"},
		{"no frontmatter", "- [x] a\n", "- [x] a\n"},
		{"unclosed frontmatter", "---\nk: v\n- [x] a\n", "---\nk: v\n- [x] a\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := BodyOf([]byte(c.content)); got != c.want {
				t.Fatalf("BodyOf = %q, want %q", got, c.want)
			}
		})
	}
}
