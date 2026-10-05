package planfiles

import (
	"context"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kandev/kandev/internal/planfiles/format"
	"github.com/kandev/kandev/internal/planfiles/scan"
	taskmodels "github.com/kandev/kandev/internal/task/models"
)

const (
	maxPlanFileNameBytes = 120
	maxSlugBytes         = 80
	maxPlanTitleChars    = 200
	planFileExt          = ".md"
	fallbackNameLayout   = "20060102-1504"
	fallbackNamePrefix   = "plan-"
)

// Create error codes. They are the machine-readable codes of the HTTP API.
const (
	CodeFileExists  = "file_exists"
	CodeInvalidPlan = "invalid_plan"
)

var planFileNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*\.md$`)

// CreatePlanError is a rejected plan creation with a machine-readable code.
type CreatePlanError struct {
	Code    string
	Message string
}

func (e *CreatePlanError) Error() string { return e.Message }

func invalidPlan(msg string, args ...any) error {
	return &CreatePlanError{Code: CodeInvalidPlan, Message: fmt.Sprintf(msg, args...)}
}

// CreatePlanRequest is the body of POST /plan-files/plans.
type CreatePlanRequest struct {
	RepositoryID string `json:"repository_id"`
	Directory    string `json:"directory"`
	Title        string `json:"title"`
	FileName     string `json:"file_name,omitempty"`
	Priority     string `json:"priority,omitempty"`
	Executor     string `json:"executor,omitempty"`
	Body         string `json:"body,omitempty"`
}

// CreatePlanResult identifies the plan file that was created and its task.
type CreatePlanResult struct {
	TaskID       string `json:"task_id"`
	RepositoryID string `json:"repository_id"`
	RelPath      string `json:"rel_path"`
}

// validatedPlan is a request that passed validation: the directory in its
// configured form, the final file name, and the file content.
type validatedPlan struct {
	relPath string
	content []byte
}

// slugOf keeps the ASCII letters and digits of title, lowercased, joins runs
// with a dash, and cuts the result at maxSlugBytes.
func slugOf(title string) string {
	var b strings.Builder
	pendingDash := false
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		case r >= 'A' && r <= 'Z':
			r += 'a' - 'A'
		default:
			pendingDash = b.Len() > 0
			continue
		}
		if pendingDash {
			b.WriteByte('-')
			pendingDash = false
		}
		b.WriteRune(r)
	}
	slug := b.String()
	if len(slug) > maxSlugBytes {
		slug = strings.TrimRight(slug[:maxSlugBytes], "-")
	}
	return slug
}

func (s *Service) defaultFileName(title string) string {
	if slug := slugOf(title); slug != "" {
		return slug + planFileExt
	}
	now := time.Now
	if s.clock != nil {
		now = s.clock
	}
	return fallbackNamePrefix + now().In(time.Local).Format(fallbackNameLayout) + planFileExt
}

// configuredDirectory returns the configured form of dir, or false when dir is
// not one of the scanned directories.
func configuredDirectory(cfg *Config, dir string) (string, bool) {
	cleaned, err := scan.ValidateDirectory(dir)
	if err != nil {
		return "", false
	}
	for _, configured := range cfg.Directories {
		if c, err := scan.ValidateDirectory(configured); err == nil && c == cleaned {
			return c, true
		}
	}
	return "", false
}

// validatePlan checks req against the config and builds the file to create.
func (s *Service) validatePlan(cfg *Config, req CreatePlanRequest) (validatedPlan, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return validatedPlan{}, invalidPlan("a title is required")
	}
	if utf8.RuneCountInString(title) > maxPlanTitleChars {
		return validatedPlan{}, invalidPlan("the title is longer than %d characters", maxPlanTitleChars)
	}
	dir, ok := configuredDirectory(cfg, req.Directory)
	if !ok {
		return validatedPlan{}, invalidPlan("the directory is not one of the scanned directories")
	}
	name := strings.TrimSpace(req.FileName)
	if name == "" {
		name = s.defaultFileName(title)
	}
	if len(name) > maxPlanFileNameBytes || !planFileNamePattern.MatchString(name) {
		return validatedPlan{}, invalidPlan("the file name must match %s and be at most %d bytes",
			planFileNamePattern.String(), maxPlanFileNameBytes)
	}
	if cfg.IndexFile != "" && name == cfg.IndexFile {
		return validatedPlan{}, invalidPlan("the file name is reserved for the plan index")
	}
	priority := strings.TrimSpace(req.Priority)
	if priority == "" {
		priority = taskmodels.TaskPriorityMedium
	}
	if err := taskmodels.ValidateTaskPriority(priority); err != nil {
		return validatedPlan{}, invalidPlan("unknown priority %q", priority)
	}
	executor := strings.TrimSpace(req.Executor)
	if len(executor) > format.ExecutorMaxBytes || strings.ContainsAny(executor, "\r\n") {
		return validatedPlan{}, invalidPlan("the executor must be one line of at most %d bytes", format.ExecutorMaxBytes)
	}
	content := format.NewPlan(format.NewPlanInput{Title: title, Priority: priority, Executor: executor, Body: req.Body})
	if len(content) > scan.MaxPlanFileBytes {
		return validatedPlan{}, invalidPlan("the plan is larger than %d bytes", scan.MaxPlanFileBytes)
	}
	return validatedPlan{relPath: path.Join(dir, name), content: content}, nil
}

// CreatePlan writes a new plan file in the named directory of a local
// repository and runs a pass so the plan task exists on return. An existing
// file is never replaced (CodeFileExists), and nothing is written when the
// request is invalid.
func (s *Service) CreatePlan(ctx context.Context, workspaceID string, req CreatePlanRequest) (CreatePlanResult, error) {
	if err := s.authorizeWorkspaceAccess(ctx, workspaceID); err != nil {
		return CreatePlanResult{}, err
	}
	if s.tasks == nil {
		return CreatePlanResult{}, errSyncNotWired
	}
	unlock := s.LockWorkspace(workspaceID)
	defer unlock()
	cfg, err := s.store.GetConfig(ctx, workspaceID)
	if err != nil {
		return CreatePlanResult{}, err
	}
	if cfg == nil || !cfg.Enabled {
		return CreatePlanResult{}, ErrNotConfigured
	}
	plan, err := s.validatePlan(cfg, req)
	if err != nil {
		return CreatePlanResult{}, err
	}
	root, err := s.localRoot(ctx, workspaceID, req.RepositoryID)
	if err != nil {
		return CreatePlanResult{}, err
	}
	if err := writeNewPlan(root, plan); err != nil {
		return CreatePlanResult{}, err
	}
	if _, err := s.runPass(ctx, workspaceID); err != nil {
		return CreatePlanResult{}, err
	}
	return s.createdPlanResult(ctx, workspaceID, req.RepositoryID, plan.relPath)
}

// localRoot returns the local path of one of the workspace's local repositories.
func (s *Service) localRoot(ctx context.Context, workspaceID, repositoryID string) (string, error) {
	repos, err := s.localRepositories(ctx, workspaceID)
	if err != nil {
		return "", err
	}
	for _, repo := range repos {
		if repo.ID == repositoryID {
			return repo.LocalPath, nil
		}
	}
	return "", &CreatePlanError{Code: CodeRepositoryNotFound,
		Message: "the repository is not a local repository of the workspace"}
}

func writeNewPlan(root string, plan validatedPlan) error {
	err := scan.CreateFile(root, plan.relPath, plan.content)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, scan.ErrExists):
		return &CreatePlanError{Code: CodeFileExists, Message: "a file with this name already exists"}
	case errors.Is(err, scan.ErrOutsideRoot):
		return invalidPlan("the directory resolves outside the repository")
	default:
		return err
	}
}

func (s *Service) createdPlanResult(ctx context.Context, workspaceID, repositoryID, relPath string) (CreatePlanResult, error) {
	rows, err := s.store.ListTaskRows(ctx, workspaceID)
	if err != nil {
		return CreatePlanResult{}, err
	}
	for _, row := range rows {
		if row.RepositoryID == repositoryID && row.RelPath == relPath {
			return CreatePlanResult{TaskID: row.TaskID, RepositoryID: repositoryID, RelPath: relPath}, nil
		}
	}
	return CreatePlanResult{}, fmt.Errorf("the plan file %s was created but its task is not on the board yet", relPath)
}
