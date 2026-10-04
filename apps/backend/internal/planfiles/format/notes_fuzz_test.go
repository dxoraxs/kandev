package format

import (
	"bytes"
	"testing"
)

// assertOnlyInserted fails unless deleting one contiguous run of bytes from
// out restores in exactly.
func assertOnlyInserted(t *testing.T, in, out []byte) {
	t.Helper()
	n := len(out) - len(in)
	if n <= 0 {
		t.Fatalf("output is not longer than input: in %q out %q", in, out)
	}
	prefix := 0
	for prefix < len(in) && in[prefix] == out[prefix] {
		prefix++
	}
	for k := prefix; k >= 0; k-- {
		if bytes.Equal(out[k+n:], in[k:]) {
			return
		}
	}
	t.Fatalf("output is not the input plus one insertion: in %q out %q", in, out)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-002.4
func FuzzAppendNote(f *testing.F) {
	f.Add([]byte("---\nboard: queued\n---\n# T\n\n## Owner notes\n\nx\n\n## Other\n"), "Owner notes", "note")
	f.Add([]byte("---\r\nboard: queued\r\n---\r\nbody"), "Owner notes", "note")
	f.Add([]byte("\xef\xbb\xbf---\nboard: queued\n---\n```\n## Owner notes\n"), "owner notes", "n")
	f.Add([]byte("---\nboard: queued\n---"), "H", "")
	f.Add([]byte("---\nboard: queued\n---\n## H"), " h ", "n")
	f.Add([]byte("---\nboard: queued\n---\n## H\n  "), "H", "n")
	f.Fuzz(func(t *testing.T, in []byte, heading, note string) {
		out, err := AppendNote(in, heading, note)
		if err != nil {
			return
		}
		assertOnlyInserted(t, in, out)
		inFM, ok := locate(in)
		if !ok {
			t.Fatal("AppendNote succeeded without frontmatter")
		}
		outFM, ok := locate(out)
		if !ok || !bytes.Equal(inFM.bom, outFM.bom) || !bytes.Equal(inFM.yamlText(), outFM.yamlText()) {
			t.Fatalf("frontmatter changed: in %q out %q", in, out)
		}
		if !bytes.Contains(out, []byte(note)) {
			t.Fatalf("note missing from output: %q", out)
		}
	})
}
