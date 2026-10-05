package planfiles

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kandev/kandev/internal/planfiles/format"
	taskservice "github.com/kandev/kandev/internal/task/service"
)

const (
	externalIDPrefix = "plan-file:"
	taskRoutePrefix  = "/t/"
	quotePrefix      = "> "

	// maxDescriptionBodyBytes bounds the plan body copied into a task
	// description, because descriptions travel in board list payloads.
	maxDescriptionBodyBytes = 16 << 10
	// lineCutWindow is how far back from the byte limit a cut may move to end
	// on a line break.
	lineCutWindow = 4 << 10
)

// defaultExternalID is the task external identifier of a plan file that sets
// no `external_id` of its own.
func defaultExternalID(repositoryID, relPath string) string {
	return externalIDPrefix + repositoryID + ":" + relPath
}

// oneLine collapses every run of whitespace, including line breaks, so file
// controlled text cannot break out of a single-line construct.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// projectTitle is the task title of a plan file: the plan title on one line,
// truncated to the task title limit.
func projectTitle(pf format.PlanFile) string {
	return taskservice.TruncateTaskTitle(oneLine(pf.Title))
}

// dependencyLink is a `depends_on` entry whose plan task exists.
type dependencyLink struct {
	Name   string
	TaskID string
}

// descriptionInput carries everything the description header names.
type descriptionInput struct {
	RepositoryName string
	RelPath        string
	File           format.PlanFile
	Dependencies   []dependencyLink
	Notice         string
}

// projectDescription is the task description: a block-quoted header naming the
// plan file, followed by the file body without its frontmatter. Every header
// value is reduced to one line so file content cannot end the quote early.
func projectDescription(in descriptionInput) string {
	var b strings.Builder
	quote := func(layout string, args ...any) {
		b.WriteString(quotePrefix)
		b.WriteString(fmt.Sprintf(layout, args...))
		b.WriteString("\n")
	}
	quote("Plan file: `%s` `%s`", headerCode(in.RepositoryName), headerCode(in.RelPath))
	if executor := oneLine(in.File.Executor); executor != "" {
		quote("Executor: %s", executor)
	}
	if len(in.Dependencies) > 0 {
		links := make([]string, 0, len(in.Dependencies))
		for _, dep := range in.Dependencies {
			links = append(links, fmt.Sprintf("[%s](%s%s)", linkText(dep.Name), taskRoutePrefix, dep.TaskID))
		}
		quote("Depends on: %s", strings.Join(links, ", "))
	}
	for _, msg := range in.File.ParseErrors {
		quote("Parse error: %s", oneLine(msg))
	}
	if notice := oneLine(in.Notice); notice != "" {
		quote("Notice: %s", notice)
	}
	body, truncated := capBody(strings.TrimLeft(in.File.Body, "\r\n"))
	if truncated {
		quote("Truncated: the plan body is longer than %d KiB; the full plan is in the file.", maxDescriptionBodyBytes>>10)
	}
	b.WriteString("\n")
	b.WriteString(body)
	return b.String()
}

// capBody keeps at most maxDescriptionBodyBytes of body, ending on a line
// break when one is close to the limit and on a rune boundary otherwise.
func capBody(body string) (string, bool) {
	if len(body) <= maxDescriptionBodyBytes {
		return body, false
	}
	cut := maxDescriptionBodyBytes
	if nl := strings.LastIndexByte(body[:cut], '\n'); nl >= cut-lineCutWindow {
		return body[:nl+1], true
	}
	for cut > 0 && !utf8.RuneStart(body[cut]) {
		cut--
	}
	return body[:cut], true
}

func headerCode(s string) string {
	return strings.ReplaceAll(oneLine(s), "`", "'")
}

func linkText(s string) string {
	return strings.NewReplacer("[", "", "]", "").Replace(oneLine(s))
}

// dependencyPath resolves a `depends_on` file name against the directory of
// the plan that lists it. A name that leaves that directory resolves to "".
func dependencyPath(planRelPath, name string) string {
	clean := path.Clean(strings.ReplaceAll(strings.TrimSpace(name), `\`, "/"))
	if clean == "." || clean == ".." || strings.Contains(clean, "/") {
		return ""
	}
	return path.Join(path.Dir(planRelPath), clean)
}

// orderKey is the desired position of a plan task inside its step: files with
// an `order` come first by ascending value, the rest follow in path order.
type orderKey struct {
	unordered bool
	order     float64
	relPath   string
	repoID    string
}

func newOrderKey(order *float64, relPath, repositoryID string) orderKey {
	key := orderKey{relPath: relPath, repoID: repositoryID, unordered: order == nil}
	if order != nil {
		key.order = *order
	}
	return key
}

func (k orderKey) less(other orderKey) bool {
	switch {
	case k.unordered != other.unordered:
		return !k.unordered
	case k.order != other.order:
		return k.order < other.order
	case k.relPath != other.relPath:
		return k.relPath < other.relPath
	}
	return k.repoID < other.repoID
}

// String is the form stored as the task row's synced order key.
func (k orderKey) String() string {
	bucket, value := "0", strconv.FormatFloat(k.order, 'f', -1, 64)
	if k.unordered {
		bucket, value = "1", ""
	}
	return bucket + ":" + value + ":" + k.relPath
}
