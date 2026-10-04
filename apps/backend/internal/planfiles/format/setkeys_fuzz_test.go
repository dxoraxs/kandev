package format

import (
	"bytes"
	"testing"
)

// @covers AC-TASKS-PLAN-FILES-003.4
func FuzzSetKeys(f *testing.F) {
	f.Add([]byte("---\nboard: queued\n---\nbody\n"))
	f.Add([]byte("---\r\nboard: queued # c\r\norder: 1\r\n---\r\n# H\r\n"))
	f.Add([]byte("\xef\xbb\xbf---\nboard: |\n  x\n---\nb"))
	f.Add([]byte("---\nboard: \"a # b\" # c\ntitle:\n  - x\n---"))
	f.Add([]byte("---\nboard: [a]\n---\n"))
	f.Add([]byte("no frontmatter"))
	values := map[string]string{"board": "done", "order": "1.5", "executor": "x"}
	f.Fuzz(func(t *testing.T, in []byte) {
		out, err := SetKeys(in, values)
		if err != nil {
			return
		}
		inFM, ok := locate(in)
		if !ok {
			t.Fatal("SetKeys succeeded without frontmatter")
		}
		outFM, ok := locate(out)
		if !ok {
			t.Fatalf("output lost frontmatter: %q", out)
		}
		if !bytes.Equal(inFM.body(), outFM.body()) || !bytes.Equal(inFM.bom, outFM.bom) {
			t.Fatalf("body or BOM changed: in %q out %q", in, out)
		}
	})
}
