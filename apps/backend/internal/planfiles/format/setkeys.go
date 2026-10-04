package format

import (
	"bytes"
	"errors"
	"sort"
	"strings"
)

// ErrUnsupportedValueShape reports a key whose current value cannot be
// replaced by a line-level edit: a block scalar, a flow collection, an
// anchor/alias/tag, a value that continues on following lines, a key that
// appears more than once, or an edit whose result does not decode to the
// original mapping with only the requested keys changed.
var ErrUnsupportedValueShape = errors.New("unsupported frontmatter value shape")

// ErrNoFrontmatter reports content without a closing-fenced frontmatter block.
var ErrNoFrontmatter = errors.New("no frontmatter block")

// ErrInvalidValue reports a replacement value that is not a single line.
var ErrInvalidValue = errors.New("frontmatter value must be a single line")

// SetKeys sets top-level frontmatter keys to the given plain scalar values.
// An existing key keeps its spelling, separator spacing, trailing comment, and
// line ending; only the value text changes. A missing key is inserted as
// `key: value` immediately before the closing fence, in sorted key order, with
// the file's line ending. Every byte outside the edited and inserted lines,
// including the body, is returned unchanged.
func SetKeys(content []byte, values map[string]string) ([]byte, error) {
	fm, ok := locate(content)
	if !ok {
		return nil, ErrNoFrontmatter
	}
	keys := make([]string, 0, len(values))
	for k, v := range values {
		if strings.ContainsAny(v, "\r\n") || k == "" || strings.ContainsAny(k, ":\r\n \t") {
			return nil, ErrInvalidValue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	eol := eolOf(fm.lines[0])
	var inserted [][]byte
	for _, k := range keys {
		idx, err := findKey(fm, k)
		if err != nil {
			return nil, err
		}
		if idx < 0 {
			inserted = append(inserted, []byte(k+": "+values[k]+eol))
			continue
		}
		line, err := rewriteLine(fm, idx, k, values[k])
		if err != nil {
			return nil, err
		}
		fm.lines[idx] = line
	}
	out := assemble(fm, inserted)
	if !verifyEdit(content, out, values) {
		return nil, ErrUnsupportedValueShape
	}
	return out, nil
}

func assemble(fm frontmatter, inserted [][]byte) []byte {
	parts := make([][]byte, 0, len(fm.lines)+len(inserted)+1)
	parts = append(parts, fm.bom)
	parts = append(parts, fm.lines[:fm.closing]...)
	parts = append(parts, inserted...)
	parts = append(parts, fm.lines[fm.closing:]...)
	return bytes.Join(parts, nil)
}

// findKey returns the index of the single top-level line for key, or -1 when
// absent. More than one match is unsupported.
func findKey(fm frontmatter, key string) (int, error) {
	found := -1
	for i := 1; i < fm.closing; i++ {
		if !bytes.HasPrefix(fm.lines[i], []byte(key+":")) {
			continue
		}
		if found >= 0 {
			return -1, ErrUnsupportedValueShape
		}
		found = i
	}
	return found, nil
}

// rewriteLine replaces the value of the key line at idx.
func rewriteLine(fm frontmatter, idx int, key, value string) ([]byte, error) {
	line := fm.lines[idx]
	eol := eolOf(line)
	rest := string(trimEOL(line))[len(key)+1:]

	sep := rest[:len(rest)-len(strings.TrimLeft(rest, " \t"))]
	rest = rest[len(sep):]
	_, comment, err := splitValue(rest)
	if err != nil {
		return nil, err
	}
	if continuesBelow(fm, idx) {
		return nil, ErrUnsupportedValueShape
	}
	if sep == "" {
		sep = " "
	}
	out := key + ":" + sep + value
	if comment != "" {
		out += comment
	}
	return []byte(out + eol), nil
}

// splitValue splits the text after `key:` and its separator into the value and
// the unparsed tail: trailing whitespace and any `#` comment, with the
// whitespace before `#`. A comment that directly follows an empty value gets a
// single leading space.
func splitValue(rest string) (value, comment string, err error) {
	if rest == "" {
		return "", "", nil
	}
	switch rest[0] {
	case '|', '>', '[', '{', '&', '*', '!':
		return "", "", ErrUnsupportedValueShape
	case '#':
		return "", " " + rest, nil
	case '"', '\'':
		end := closingQuote(rest)
		if end < 0 {
			return "", "", ErrUnsupportedValueShape
		}
		return rest[:end+1], rest[end+1:], nil
	}
	value = strings.TrimRight(rest[:plainCommentStart(rest)], " \t")
	return value, rest[len(value):], nil
}

// plainCommentStart returns the index of the `#` that starts a comment in a
// plain scalar, or len(rest) when there is none.
func plainCommentStart(rest string) int {
	for i := 1; i < len(rest); i++ {
		if rest[i] == '#' && (rest[i-1] == ' ' || rest[i-1] == '\t') {
			return i
		}
	}
	return len(rest)
}

// closingQuote returns the index of the quote that closes the quoted scalar at
// the start of s, or -1 when it does not close on this line.
func closingQuote(s string) int {
	q := s[0]
	for i := 1; i < len(s); i++ {
		switch {
		case q == '"' && s[i] == '\\':
			i++
		case s[i] == q && q == '\'' && i+1 < len(s) && s[i+1] == '\'':
			i++
		case s[i] == q:
			return i
		}
	}
	return -1
}

// continuesBelow reports whether the value of the key at idx continues on the
// following lines: an indented line or a same-level block sequence entry.
func continuesBelow(fm frontmatter, idx int) bool {
	for i := idx + 1; i < fm.closing; i++ {
		text := string(trimEOL(fm.lines[i]))
		trimmed := strings.TrimSpace(text)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		return text[0] == ' ' || text[0] == '\t' || strings.HasPrefix(text, "- ") || text == "-"
	}
	return false
}
