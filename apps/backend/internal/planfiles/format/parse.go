package format

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kandev/kandev/internal/task/models"
	"github.com/kandev/kandev/internal/task/service"
)

// BoardStatus is the closed set of plan board statuses.
type BoardStatus string

// Board statuses accepted in the `board` frontmatter key.
const (
	BoardQueued          BoardStatus = "queued"
	BoardInProgress      BoardStatus = "in_progress"
	BoardWaitingOwner    BoardStatus = "waiting_owner"
	BoardWaitingExternal BoardStatus = "waiting_external"
	BoardDeferred        BoardStatus = "deferred"
	BoardDone            BoardStatus = "done"
	BoardHidden          BoardStatus = "hidden"
)

// ExecutorMaxBytes is the maximum UTF-8 byte length of the `executor` value.
const ExecutorMaxBytes = 40

// Frontmatter key names.
const (
	keyBoard      = "board"
	keyTitle      = "title"
	keyPriority   = "priority"
	keyOrder      = "order"
	keyExecutor   = "executor"
	keyDependsOn  = "depends_on"
	keyExternalID = "external_id"
)

var validBoards = map[BoardStatus]struct{}{
	BoardQueued: {}, BoardInProgress: {}, BoardWaitingOwner: {}, BoardWaitingExternal: {},
	BoardDeferred: {}, BoardDone: {}, BoardHidden: {},
}

// PlanFile is the typed view of a plan file. An invalid optional field is
// left at its zero value (Priority falls back to its default) and reported in
// ParseErrors.
type PlanFile struct {
	Board       BoardStatus
	Title       string
	Priority    string
	Order       *float64
	Executor    string
	Date        string
	DependsOn   []string
	ExternalID  string
	Tracks      []string
	Body        string
	ParseErrors []string
	Hash        string
}

// Parse reads a plan file. name is the file name and supplies the title
// fallback (its stem). ok is false when the content has no closing-fenced
// frontmatter, the frontmatter is not a valid YAML mapping, or it has no
// `board` key; such files are not plan files. An invalid `board` value is a
// parse error, not ok == false.
func Parse(name string, content []byte) (PlanFile, bool) {
	fm, ok := locate(content)
	if !ok {
		return PlanFile{}, false
	}
	var fields map[string]yaml.Node
	if err := yaml.Unmarshal(fm.yamlText(), &fields); err != nil {
		return PlanFile{}, false
	}
	boardNode, hasBoard := fields[keyBoard]
	if !hasBoard {
		return PlanFile{}, false
	}
	pf := PlanFile{Body: string(fm.body()), Hash: ContentHash(content), Priority: models.TaskPriorityMedium}
	pf.decodeBoard(boardNode)
	pf.decodeOptional(fields)
	pf.decodeTracks(fields)
	pf.decodeDate(fields)
	if pf.Title == "" {
		pf.Title = fallbackTitle(name, pf.Body)
	}
	return pf, true
}

func (pf *PlanFile) fail(key string, err error) {
	pf.ParseErrors = append(pf.ParseErrors, fmt.Sprintf("%s: %v", key, err))
}

func (pf *PlanFile) decodeBoard(n yaml.Node) {
	s, err := scalarString(n)
	if err != nil {
		pf.fail(keyBoard, err)
		return
	}
	status := BoardStatus(strings.ToLower(strings.TrimSpace(s)))
	if _, valid := validBoards[status]; !valid {
		pf.fail(keyBoard, fmt.Errorf("unknown status %q", s))
		return
	}
	pf.Board = status
}

func (pf *PlanFile) decodeOptional(fields map[string]yaml.Node) {
	if n, ok := fields[keyTitle]; ok {
		if s, err := scalarString(n); err != nil {
			pf.fail(keyTitle, err)
		} else {
			pf.Title = strings.TrimSpace(s)
		}
	}
	if n, ok := fields[keyPriority]; ok {
		pf.decodePriority(n)
	}
	if n, ok := fields[keyOrder]; ok {
		pf.decodeOrder(n)
	}
	if n, ok := fields[keyExecutor]; ok {
		pf.decodeExecutor(n)
	}
	if n, ok := fields[keyDependsOn]; ok {
		pf.decodeDependsOn(n)
	}
	if n, ok := fields[keyExternalID]; ok {
		pf.decodeExternalID(n)
	}
}

func (pf *PlanFile) decodePriority(n yaml.Node) {
	s, err := scalarString(n)
	if err != nil {
		pf.fail(keyPriority, err)
		return
	}
	p := strings.ToLower(strings.TrimSpace(s))
	if p == "" {
		return
	}
	if err := models.ValidateTaskPriority(p); err != nil {
		pf.fail(keyPriority, err)
		return
	}
	pf.Priority = p
}

func (pf *PlanFile) decodeOrder(n yaml.Node) {
	if n.Kind != yaml.ScalarNode || isNull(n) {
		if !isNull(n) {
			pf.fail(keyOrder, errors.New("must be a number"))
		}
		return
	}
	var f float64
	if err := n.Decode(&f); err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		pf.fail(keyOrder, errors.New("must be a finite number"))
		return
	}
	pf.Order = &f
}

func (pf *PlanFile) decodeExecutor(n yaml.Node) {
	s, err := scalarString(n)
	if err != nil {
		pf.fail(keyExecutor, err)
		return
	}
	s = strings.TrimSpace(s)
	if len(s) > ExecutorMaxBytes {
		pf.fail(keyExecutor, fmt.Errorf("must be %d UTF-8 bytes or fewer", ExecutorMaxBytes))
		return
	}
	pf.Executor = s
}

func (pf *PlanFile) decodeDependsOn(n yaml.Node) {
	if isNull(n) {
		return
	}
	if n.Kind != yaml.SequenceNode {
		pf.fail(keyDependsOn, errors.New("must be a list of file names"))
		return
	}
	deps := make([]string, 0, len(n.Content))
	for _, item := range n.Content {
		s, err := scalarString(*item)
		s = strings.TrimSpace(s)
		if err != nil || s == "" {
			pf.fail(keyDependsOn, errors.New("entries must be non-empty file names"))
			return
		}
		deps = append(deps, s)
	}
	pf.DependsOn = deps
}

func (pf *PlanFile) decodeExternalID(n yaml.Node) {
	s, err := scalarString(n)
	if err != nil {
		pf.fail(keyExternalID, err)
		return
	}
	// Reuses the task service rules so plan-file identifiers and API
	// identifiers are interchangeable.
	norm, err := service.NormalizeExternalID(s)
	if err != nil {
		pf.fail(keyExternalID, err)
		return
	}
	pf.ExternalID = norm
}

func isNull(n yaml.Node) bool {
	return n.Kind == yaml.ScalarNode && n.Tag == "!!null"
}

// scalarString returns the text of a scalar node. A null scalar is the empty
// string.
func scalarString(n yaml.Node) (string, error) {
	if n.Kind != yaml.ScalarNode {
		return "", errors.New("must be a text value")
	}
	if isNull(n) {
		return "", nil
	}
	return n.Value, nil
}

// fallbackTitle returns the first level-one heading outside fenced code
// blocks, else the file name without its extension.
func fallbackTitle(name, body string) string {
	inFence := false
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimRight(line, "\r")
		if isCodeFence(line) {
			inFence = !inFence
			continue
		}
		if inFence || !strings.HasPrefix(line, "# ") {
			continue
		}
		if t := strings.TrimSpace(line[2:]); t != "" {
			return t
		}
	}
	base := filepath.Base(name)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// isCodeFence reports whether a line (without its line ending) opens or closes
// a fenced code block.
func isCodeFence(line string) bool {
	return strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~")
}
