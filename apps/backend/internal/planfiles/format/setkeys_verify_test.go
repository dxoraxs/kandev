package format

import (
	"errors"
	"testing"
)

// @covers AC-TASKS-PLAN-FILES-003.4
func TestSetKeysRefusesKeySpellingsTheLineEditorCannotMatch(t *testing.T) {
	cases := map[string]string{
		"double-quoted key":      "---\n\"board\": queued\n---\nbody\n",
		"single-quoted key":      "---\n'board': queued\n---\nbody\n",
		"space before colon":     "---\nboard : queued\n---\nbody\n",
		"explicit key indicator": "---\n? board\n: queued\n---\nbody\n",
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			out, err := SetKeys([]byte(in), map[string]string{"board": "done"})
			if !errors.Is(err, ErrUnsupportedValueShape) {
				t.Fatalf("SetKeys err = %v, out = %q; want ErrUnsupportedValueShape", err, out)
			}
		})
	}
}

// @covers AC-TASKS-PLAN-FILES-003.4
func TestSetKeysResultParsesWithTheWrittenValues(t *testing.T) {
	in := "---\nboard: queued # status\ntitle: \"A: B\"\n---\n# Heading\n"
	out, err := SetKeys([]byte(in), map[string]string{"board": "done", "order": "12.5"})
	if err != nil {
		t.Fatalf("SetKeys: %v", err)
	}
	pf, ok := Parse("plan.md", out)
	if !ok || pf.Board != BoardDone || pf.Order == nil || *pf.Order != 12.5 || pf.Title != "A: B" {
		t.Fatalf("Parse after SetKeys = %+v, ok=%v", pf, ok)
	}
}
