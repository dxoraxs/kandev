package planfiles

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kandev/kandev/internal/planfiles/format"
	wfmodels "github.com/kandev/kandev/internal/workflow/models"
)

// Error codes of a rejected operation setting.
const (
	codeInvalidExecutorSteps  = "invalid_executor_steps"
	codeInvalidNotesHeading   = "invalid_notes_heading"
	codeInvalidStaleAfterDays = "invalid_stale_after_days"
	codeInvalidIndexFile      = "invalid_index_file"
)

const (
	maxNotesHeadingChars = 100
	maxStaleAfterDays    = 365
	// DefaultStaleAfterDays is the stale threshold of a config that never set one.
	DefaultStaleAfterDays = 7
)

var indexFilePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*\.md$`)

// ConfigError is a rejected PUT body that carries a machine-readable code.
type ConfigError struct {
	Code    string
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("%s: %s", ErrInvalidConfig, e.Message)
}

// Unwrap lets errors.Is(err, ErrInvalidConfig) match.
func (e *ConfigError) Unwrap() error { return ErrInvalidConfig }

func invalidCode(code, format string, args ...any) error {
	return &ConfigError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// operationSettings are the settings of a config beyond board and directories.
type operationSettings struct {
	executorSteps  map[string]string
	notesHeading   string
	wakeOnDate     bool
	staleAfterDays int
	indexFile      string
}

// resolveOperationSettings overlays the request on the stored config. A field
// the request omits keeps the stored value, or the default when nothing is
// stored. The result is validated and normalized.
func resolveOperationSettings(existing *Config, req *PutConfigRequest) (operationSettings, error) {
	out := operationSettings{
		executorSteps:  map[string]string{},
		wakeOnDate:     true,
		staleAfterDays: DefaultStaleAfterDays,
	}
	if existing != nil {
		out = operationSettings{
			executorSteps:  existing.ExecutorSteps,
			notesHeading:   existing.NotesHeading,
			wakeOnDate:     existing.WakeOnDate,
			staleAfterDays: existing.StaleAfterDays,
			indexFile:      existing.IndexFile,
		}
	}
	if existing != nil && existing.WorkflowID != req.WorkflowID {
		// Executor steps are steps of the old workflow.
		out.executorSteps = map[string]string{}
	}
	if req.ExecutorSteps != nil {
		out.executorSteps = *req.ExecutorSteps
	}
	if req.NotesHeading != nil {
		out.notesHeading = *req.NotesHeading
	}
	if req.WakeOnDate != nil {
		out.wakeOnDate = *req.WakeOnDate
	}
	if req.StaleAfterDays != nil {
		out.staleAfterDays = *req.StaleAfterDays
	}
	if req.IndexFile != nil {
		out.indexFile = *req.IndexFile
	}
	return out, out.normalize()
}

// normalize validates the plain settings and trims text fields. Executor steps
// are checked against the workflow separately.
func (o *operationSettings) normalize() error {
	o.notesHeading = strings.TrimSpace(o.notesHeading)
	if utf8.RuneCountInString(o.notesHeading) > maxNotesHeadingChars {
		return invalidCode(codeInvalidNotesHeading, "notes_heading must be %d characters or fewer", maxNotesHeadingChars)
	}
	if strings.IndexFunc(o.notesHeading, unicode.IsControl) >= 0 {
		return invalidCode(codeInvalidNotesHeading, "notes_heading must be a single line")
	}
	if o.staleAfterDays < 0 || o.staleAfterDays > maxStaleAfterDays {
		return invalidCode(codeInvalidStaleAfterDays, "stale_after_days must be between 0 and %d", maxStaleAfterDays)
	}
	o.indexFile = strings.TrimSpace(o.indexFile)
	if o.indexFile != "" && !indexFilePattern.MatchString(o.indexFile) {
		return invalidCode(codeInvalidIndexFile, "index_file must be a Markdown file name such as INDEX.md")
	}
	return nil
}

// validateExecutorSteps checks that every executor step is a step of the
// workflow that no status maps to, and that names are 1 to ExecutorMaxBytes
// bytes and unique ignoring case. It replaces the map's names by their
// trimmed form.
func (o *operationSettings) validateExecutorSteps(
	mapping map[format.BoardStatus]string, steps []*wfmodels.WorkflowStep,
) error {
	stepIDs := make(map[string]struct{}, len(steps))
	for _, step := range steps {
		stepIDs[step.ID] = struct{}{}
	}
	mapped := make(map[string]struct{}, len(mapping))
	for _, stepID := range mapping {
		mapped[strings.TrimSpace(stepID)] = struct{}{}
	}
	names := make(map[string]struct{}, len(o.executorSteps))
	cleaned := make(map[string]string, len(o.executorSteps))
	for stepID, name := range o.executorSteps {
		if _, ok := stepIDs[stepID]; !ok {
			return invalidCode(codeInvalidExecutorSteps, "executor step %q does not belong to the workflow", stepID)
		}
		if _, ok := mapped[stepID]; ok {
			return invalidCode(codeInvalidExecutorSteps, "executor step %q is already mapped to a status", stepID)
		}
		name = strings.TrimSpace(name)
		if name == "" || len(name) > format.ExecutorMaxBytes {
			return invalidCode(codeInvalidExecutorSteps,
				"executor names must be 1 to %d bytes", format.ExecutorMaxBytes)
		}
		key := strings.ToLower(name)
		if _, dup := names[key]; dup {
			return invalidCode(codeInvalidExecutorSteps, "executor name %q is used twice", name)
		}
		names[key] = struct{}{}
		cleaned[stepID] = name
	}
	o.executorSteps = cleaned
	return nil
}
