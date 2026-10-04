// Package format parses plan files (Markdown with a YAML frontmatter block)
// and edits frontmatter keys without disturbing any other byte. It performs no
// I/O.
package format

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
)

const (
	fence = "---"
	utf8B = "\xef\xbb\xbf"
)

// ContentHash returns the lower-case hex SHA-256 of the full file bytes.
func ContentHash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// frontmatter locates the frontmatter block of a file.
//
// Invariants: lines concatenate to the content after the BOM; lines[0] is the
// opening fence and lines[closing] is the closing fence; the body starts at
// lines[closing+1].
type frontmatter struct {
	bom     []byte
	lines   [][]byte
	closing int
}

// locate finds the frontmatter block. ok is false when the file does not start
// with a fence line or the closing fence line is missing.
func locate(content []byte) (frontmatter, bool) {
	var fm frontmatter
	if bytes.HasPrefix(content, []byte(utf8B)) {
		fm.bom = content[:len(utf8B)]
		content = content[len(utf8B):]
	}
	fm.lines = bytes.SplitAfter(content, []byte("\n"))
	if len(fm.lines) > 0 && len(fm.lines[len(fm.lines)-1]) == 0 {
		fm.lines = fm.lines[:len(fm.lines)-1]
	}
	if len(fm.lines) < 2 || !isFence(fm.lines[0]) {
		return fm, false
	}
	for i := 1; i < len(fm.lines); i++ {
		if isFence(fm.lines[i]) {
			fm.closing = i
			return fm, true
		}
	}
	return fm, false
}

func isFence(line []byte) bool {
	return string(trimEOL(line)) == fence
}

func trimEOL(line []byte) []byte {
	line = bytes.TrimSuffix(line, []byte("\n"))
	return bytes.TrimSuffix(line, []byte("\r"))
}

// eolOf returns the line ending of a line, or "" when it has none.
func eolOf(line []byte) string {
	switch {
	case bytes.HasSuffix(line, []byte("\r\n")):
		return "\r\n"
	case bytes.HasSuffix(line, []byte("\n")):
		return "\n"
	}
	return ""
}

// yamlText returns the frontmatter lines between the fences.
func (fm frontmatter) yamlText() []byte {
	return bytes.Join(fm.lines[1:fm.closing], nil)
}

// body returns the bytes after the closing fence line.
func (fm frontmatter) body() []byte {
	return bytes.Join(fm.lines[fm.closing+1:], nil)
}
