package format

import (
	"bytes"
	"strings"
)

const noteHeadingPrefix = "## "

// AppendNote inserts a one-line note into the section of the file headed
// `## <heading>` and returns the new content; every input byte is preserved
// and only the inserted bytes are added. The heading matches by TrimSpace and
// EqualFold, outside fenced code blocks. The note goes after the last
// non-blank line of the section. A missing section is appended at the end of
// the file, after one blank line, followed by a blank line and the note.
// Inserted text uses the line ending of the frontmatter's opening fence.
func AppendNote(content []byte, heading, line string) ([]byte, error) {
	heading = strings.TrimSpace(heading)
	if heading == "" || strings.ContainsAny(heading, "\r\n") || strings.ContainsAny(line, "\r\n") {
		return nil, ErrInvalidValue
	}
	fm, ok := locate(content)
	if !ok {
		return nil, ErrNoFrontmatter
	}
	eol := eolOf(fm.lines[0])
	lines := fm.lines
	anchor, found := noteAnchor(lines, fm.closing+1, heading)
	if !found {
		return appendSection(content, lines, eol, heading, line), nil
	}
	offset := len(fm.bom)
	for _, l := range lines[:anchor+1] {
		offset += len(l)
	}
	insert := terminator(lines[anchor], eol) + line + eol
	return spliceBytes(content, offset, insert), nil
}

// noteAnchor returns the index of the line after which a note belongs: the
// last non-blank line of the matching section, or its heading line.
func noteAnchor(lines [][]byte, from int, heading string) (int, bool) {
	inFence := false
	anchor := -1
	for i := from; i < len(lines); i++ {
		text := string(trimEOL(lines[i]))
		fenceLine := isCodeFence(text)
		if fenceLine {
			inFence = !inFence
		}
		if !fenceLine && !inFence && strings.HasPrefix(text, noteHeadingPrefix) {
			if anchor >= 0 {
				return anchor, true
			}
			if strings.EqualFold(strings.TrimSpace(text[len(noteHeadingPrefix):]), heading) {
				anchor = i
			}
			continue
		}
		if anchor >= 0 && (fenceLine || strings.TrimSpace(text) != "") {
			anchor = i
		}
	}
	return anchor, anchor >= 0
}

func appendSection(content []byte, lines [][]byte, eol, heading, note string) []byte {
	var b strings.Builder
	last := lines[len(lines)-1]
	b.WriteString(terminator(last, eol))
	if len(bytes.TrimSpace(last)) != 0 {
		b.WriteString(eol)
	}
	b.WriteString(noteHeadingPrefix + heading + eol + eol + note + eol)
	return spliceBytes(content, len(content), b.String())
}

// terminator returns the bytes that end a line: nothing when it already ends,
// a lone LF after a trailing CR, else the file's line ending.
func terminator(line []byte, eol string) string {
	switch {
	case eolOf(line) != "":
		return ""
	case bytes.HasSuffix(line, []byte("\r")):
		return "\n"
	}
	return eol
}

func spliceBytes(content []byte, at int, insert string) []byte {
	out := make([]byte, 0, len(content)+len(insert))
	out = append(out, content[:at]...)
	out = append(out, insert...)
	return append(out, content[at:]...)
}
