package format

import "strings"

// CountItems counts Markdown task-list items outside fenced code blocks. A
// item is a `-` or `*` marker, any indentation, then `[ ]`, `[x]`, or `[X]`.
func CountItems(body string) (done, total int) {
	inFence := false
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimRight(line, "\r")
		if isCodeFence(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		open, isItem := taskItem(line)
		if !isItem {
			continue
		}
		total++
		if !open {
			done++
		}
	}
	return done, total
}

// taskItem reports whether the line is a task-list item and whether it is
// still open.
func taskItem(line string) (open, ok bool) {
	rest := strings.TrimLeft(line, " \t")
	if len(rest) < 5 || (rest[0] != '-' && rest[0] != '*') || (rest[1] != ' ' && rest[1] != '\t') {
		return false, false
	}
	rest = strings.TrimLeft(rest[1:], " \t")
	if len(rest) < 3 || rest[0] != '[' || rest[2] != ']' {
		return false, false
	}
	if len(rest) > 3 && rest[3] != ' ' && rest[3] != '\t' {
		return false, false
	}
	switch rest[1] {
	case ' ':
		return true, true
	case 'x', 'X':
		return false, true
	}
	return false, false
}

// BodyOf returns the text after the frontmatter block, or the whole content
// when the file has none.
func BodyOf(content []byte) string {
	fm, ok := locate(content)
	if !ok {
		return string(content)
	}
	return string(fm.body())
}
