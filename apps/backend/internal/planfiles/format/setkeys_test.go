package format

import (
	"errors"
	"testing"
)

// @covers AC-TASKS-PLAN-FILES-003.4
func TestSetKeysByteExact(t *testing.T) {
	cases := []struct {
		name, in, want string
		values         map[string]string
	}{
		{
			"replace keeps comment and order",
			"---\ntitle: T\nboard: queued   # lane\norder: 1\n---\nbody\n",
			"---\ntitle: T\nboard: done   # lane\norder: 1\n---\nbody\n",
			map[string]string{"board": "done"},
		},
		{
			"separator spacing preserved",
			"---\nboard:    queued\n---\n",
			"---\nboard:    done\n---\n",
			map[string]string{"board": "done"},
		},
		{
			"no separator gets one space",
			"---\nboard:\n---\n",
			"---\nboard: done\n---\n",
			map[string]string{"board": "done"},
		},
		{
			"empty value with comment",
			"---\nboard: # c\n---\n",
			"---\nboard: done # c\n---\n",
			map[string]string{"board": "done"},
		},
		{
			"quoted value replaced with comment",
			"---\nboard: \"queued\" # c\n---\n",
			"---\nboard: done # c\n---\n",
			map[string]string{"board": "done"},
		},
		{
			"hash inside quotes is not a comment",
			"---\ntitle: 'a # b'\n---\n",
			"---\ntitle: x\n---\n",
			map[string]string{"title": "x"},
		},
		{
			"hash without space is not a comment",
			"---\ntitle: a#b\n---\n",
			"---\ntitle: x\n---\n",
			map[string]string{"title": "x"},
		},
		{
			"CRLF preserved on edit and insert",
			"---\r\nboard: queued\r\n---\r\nbody\r\n",
			"---\r\nboard: done\r\norder: 2\r\n---\r\nbody\r\n",
			map[string]string{"board": "done", "order": "2"},
		},
		{
			"insert sorted before closing fence",
			"---\nboard: queued\n# keep\n---\n# H\n",
			"---\nboard: queued\n# keep\nexecutor: x\norder: 1\n---\n# H\n",
			map[string]string{"order": "1", "executor": "x"},
		},
		{
			"BOM preserved",
			"\xef\xbb\xbf---\nboard: queued\n---\nb",
			"\xef\xbb\xbf---\nboard: done\n---\nb",
			map[string]string{"board": "done"},
		},
		{
			"closing fence without final newline",
			"---\nboard: queued\n---",
			"---\nboard: queued\norder: 3\n---",
			map[string]string{"order": "3"},
		},
		{
			"body lookalike key untouched",
			"---\nboard: queued\n---\nboard: other\n",
			"---\nboard: done\n---\nboard: other\n",
			map[string]string{"board": "done"},
		},
		{
			"unrelated key with shared prefix untouched",
			"---\nboard_x: 1\nboard: queued\n---\n",
			"---\nboard_x: 1\nboard: done\n---\n",
			map[string]string{"board": "done"},
		},
		{
			"following comment line is not continuation",
			"---\nboard: queued\n  # indented comment\norder: 1\n---\n",
			"---\nboard: done\n  # indented comment\norder: 1\n---\n",
			map[string]string{"board": "done"},
		},
		{
			"no-op set of empty map",
			"---\nboard: queued\n---\n",
			"---\nboard: queued\n---\n",
			map[string]string{},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := SetKeys([]byte(c.in), c.values)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != c.want {
				t.Fatalf("got  %q\nwant %q", got, c.want)
			}
		})
	}
}

func TestSetKeysUnsupportedShapes(t *testing.T) {
	cases := map[string]string{
		"block scalar literal": "---\nboard: |\n  queued\n---\n",
		"block scalar folded":  "---\nboard: >\n  queued\n---\n",
		"flow sequence":        "---\nboard: [queued]\n---\n",
		"flow mapping":         "---\nboard: {a: b}\n---\n",
		"anchor":               "---\nboard: &a queued\n---\n",
		"alias":                "---\nboard: *a\n---\n",
		"tag":                  "---\nboard: !!str queued\n---\n",
		"continued plain":      "---\nboard: queued\n  more\n---\n",
		"block sequence":       "---\nboard:\n  - queued\n---\n",
		"same-level sequence":  "---\nboard:\n- queued\n---\n",
		"unterminated quote":   "---\nboard: \"queued\n  more\"\n---\n",
		"duplicate key":        "---\nboard: queued\nboard: done\n---\n",
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := SetKeys([]byte(in), map[string]string{"board": "done"})
			if !errors.Is(err, ErrUnsupportedValueShape) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestSetKeysErrors(t *testing.T) {
	if _, err := SetKeys([]byte("no frontmatter\n"), map[string]string{"a": "b"}); !errors.Is(err, ErrNoFrontmatter) {
		t.Fatalf("err = %v", err)
	}
	if _, err := SetKeys([]byte("---\na: 1\n---\n"), map[string]string{"a": "x\ny"}); !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("err = %v", err)
	}
}

func TestSetKeysInputNotMutated(t *testing.T) {
	in := []byte("---\nboard: queued\n---\nbody\n")
	orig := string(in)
	if _, err := SetKeys(in, map[string]string{"board": "done", "order": "1"}); err != nil {
		t.Fatal(err)
	}
	if string(in) != orig {
		t.Fatalf("input mutated: %q", in)
	}
}

func TestSetKeysResultParses(t *testing.T) {
	in := []byte("---\nboard: queued\ntitle: T # c\n---\n# H\n")
	out, err := SetKeys(in, map[string]string{"board": "done", "order": "12.5"})
	if err != nil {
		t.Fatal(err)
	}
	pf, ok := Parse("p.md", out)
	if !ok || pf.Board != BoardDone || pf.Order == nil || *pf.Order != 12.5 || pf.Title != "T" {
		t.Fatalf("ok=%v pf=%+v", ok, pf)
	}
}
