package format

import (
	"errors"
	"testing"
)

// @covers AC-TASKS-PLAN-BOARD-OPS-002.4
func TestAppendNote(t *testing.T) {
	cases := []struct {
		name, in, heading, note, want string
	}{
		{
			"after last non-blank of section",
			"---\nboard: queued\n---\n# T\n\n## Owner notes\n\nfirst\n\n\n## Other\nx\n",
			"Owner notes", "second",
			"---\nboard: queued\n---\n# T\n\n## Owner notes\n\nfirst\nsecond\n\n\n## Other\nx\n",
		},
		{
			"empty section",
			"---\nboard: queued\n---\n## Owner notes\n\n## Other\n",
			"Owner notes", "n",
			"---\nboard: queued\n---\n## Owner notes\nn\n\n## Other\n",
		},
		{
			"case and space insensitive heading",
			"---\nboard: queued\n---\n##   owner NOTES  \nold\n",
			"  Owner Notes ", "n",
			"---\nboard: queued\n---\n##   owner NOTES  \nold\nn\n",
		},
		{
			"heading inside fence ignored",
			"---\nboard: queued\n---\n```\n## Owner notes\n```\n",
			"Owner notes", "n",
			"---\nboard: queued\n---\n```\n## Owner notes\n```\n\n## Owner notes\n\nn\n",
		},
		{
			"fenced heading inside section does not end it",
			"---\nboard: queued\n---\n## Owner notes\n```\n## Other\n```\ntail\n",
			"Owner notes", "n",
			"---\nboard: queued\n---\n## Owner notes\n```\n## Other\n```\ntail\nn\n",
		},
		{
			"missing section, file ends with newline",
			"---\nboard: queued\n---\nbody\n",
			"Owner notes", "n",
			"---\nboard: queued\n---\nbody\n\n## Owner notes\n\nn\n",
		},
		{
			"missing section, file ends with blank line",
			"---\nboard: queued\n---\nbody\n\n",
			"Owner notes", "n",
			"---\nboard: queued\n---\nbody\n\n## Owner notes\n\nn\n",
		},
		{
			"missing section, no trailing newline",
			"---\nboard: queued\n---\nbody",
			"Owner notes", "n",
			"---\nboard: queued\n---\nbody\n\n## Owner notes\n\nn\n",
		},
		{
			"section is last line without newline",
			"---\nboard: queued\n---\n## Owner notes\nold",
			"Owner notes", "n",
			"---\nboard: queued\n---\n## Owner notes\nold\nn\n",
		},
		{
			"heading is last line without newline",
			"---\nboard: queued\n---\n## Owner notes",
			"Owner notes", "n",
			"---\nboard: queued\n---\n## Owner notes\nn\n",
		},
		{
			"frontmatter only",
			"---\nboard: queued\n---",
			"Owner notes", "n",
			"---\nboard: queued\n---\n\n## Owner notes\n\nn\n",
		},
		{
			"crlf",
			"---\r\nboard: queued\r\n---\r\n## Owner notes\r\nold\r\n\r\n## Other\r\n",
			"Owner notes", "n",
			"---\r\nboard: queued\r\n---\r\n## Owner notes\r\nold\r\nn\r\n\r\n## Other\r\n",
		},
		{
			"trailing lone CR on the last line",
			"---\r\nboard: queued\r\n---\r",
			"Owner notes", "n",
			"---\r\nboard: queued\r\n---\r\n\r\n## Owner notes\r\n\r\nn\r\n",
		},
		{
			"crlf missing section",
			"\xef\xbb\xbf---\r\nboard: queued\r\n---\r\nbody\r\n",
			"Owner notes", "n",
			"\xef\xbb\xbf---\r\nboard: queued\r\n---\r\nbody\r\n\r\n## Owner notes\r\n\r\nn\r\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := AppendNote([]byte(c.in), c.heading, c.note)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != c.want {
				t.Fatalf("got  %q\nwant %q", got, c.want)
			}
			assertOnlyInserted(t, []byte(c.in), got)
			if pf, ok := Parse("p.md", got); !ok || len(pf.ParseErrors) != 0 {
				t.Fatalf("result no longer parses: %q", got)
			}
		})
	}
}

func TestAppendNoteRejections(t *testing.T) {
	in := []byte("---\nboard: queued\n---\nbody\n")
	for _, note := range []string{"a\nb", "a\rb", "a\r\nb"} {
		if _, err := AppendNote(in, "Owner notes", note); !errors.Is(err, ErrInvalidValue) {
			t.Errorf("note %q: err = %v", note, err)
		}
	}
	for _, heading := range []string{"", "  ", "a\nb"} {
		if _, err := AppendNote(in, heading, "n"); !errors.Is(err, ErrInvalidValue) {
			t.Errorf("heading %q: err = %v", heading, err)
		}
	}
	if _, err := AppendNote([]byte("no frontmatter"), "Owner notes", "n"); !errors.Is(err, ErrNoFrontmatter) {
		t.Errorf("err = %v", err)
	}
}
