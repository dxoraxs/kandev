package planfiles

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/kandev/kandev/internal/common/logger"
	"github.com/kandev/kandev/internal/planfiles/format"
	"github.com/kandev/kandev/internal/planfiles/scan"
	taskmodels "github.com/kandev/kandev/internal/task/models"
	"github.com/kandev/kandev/internal/task/repository/repoerrors"
	taskservice "github.com/kandev/kandev/internal/task/service"
	wfmodels "github.com/kandev/kandev/internal/workflow/models"
)

// ErrInvalidConfig marks a rejected PUT body. Handlers map it to 400.
var ErrInvalidConfig = errors.New("invalid plan files config")

// plansBoardName is the name of a workflow created by CreateBoard.
const plansBoardName = "Plans"

// WorkflowAccess is the part of the task service that plan-file configuration
// needs. Satisfied by *taskservice.Service.
type WorkflowAccess interface {
	GetWorkflow(ctx context.Context, id string) (*taskmodels.Workflow, error)
	CreateWorkflow(ctx context.Context, req *taskservice.CreateWorkflowRequest) (*taskmodels.Workflow, error)
	DeleteWorkflow(ctx context.Context, id string) error
}

// StepAccess is the part of the workflow service that plan-file configuration
// needs. Satisfied by *workflowservice.Service.
type StepAccess interface {
	ListStepsByWorkflow(ctx context.Context, workflowID string) ([]*wfmodels.WorkflowStep, error)
	GetTemplate(ctx context.Context, id string) (*wfmodels.WorkflowTemplate, error)
}

// Service owns plan-file configuration and, through its sync methods, the
// reconciliation of plan files with tasks.
type Service struct {
	store     *Store
	workflows WorkflowAccess
	steps     StepAccess
	logger    *logger.Logger
	// locks serializes config mutations per workspace.
	locks sync.Map // workspaceID -> *sync.Mutex
	// passLocks serializes sync passes and board write-backs per workspace.
	passLocks sync.Map // workspaceID -> *sync.Mutex

	tasks    TaskAccess
	archiver TaskArchiver

	// clock supplies the current time to owner decisions; nil means time.Now.
	clock func() time.Time

	// workspaceAuthorizer enforces per-user workspace scoping. Nil, or a
	// context without caller identity, means unscoped: internal callers such
	// as the poller are allowed.
	workspaceAuthorizer func(context.Context, string) error
}

// NewService creates the plan-file service.
func NewService(store *Store, workflows WorkflowAccess, steps StepAccess, log *logger.Logger) *Service {
	return &Service{
		store:     store,
		workflows: workflows,
		steps:     steps,
		logger:    log.WithFields(zap.String("component", "planfiles-service")),
	}
}

// Store exposes the store for e2e reset and the sync pass.
func (s *Service) Store() *Store {
	return s.store
}

// SetWorkspaceAuthorizer installs the per-user workspace access boundary
// applied before every user-facing read and write.
func (s *Service) SetWorkspaceAuthorizer(authorizer func(context.Context, string) error) {
	if s != nil {
		s.workspaceAuthorizer = authorizer
	}
}

func (s *Service) authorizeWorkspaceAccess(ctx context.Context, workspaceID string) error {
	if s == nil || s.workspaceAuthorizer == nil {
		return nil
	}
	return s.workspaceAuthorizer(ctx, workspaceID)
}

func (s *Service) workspaceLock(workspaceID string) *sync.Mutex {
	lock, _ := s.locks.LoadOrStore(workspaceID, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

// GetConfig returns the workspace's config, or nil when none is stored.
func (s *Service) GetConfig(ctx context.Context, workspaceID string) (*Config, error) {
	if err := s.authorizeWorkspaceAccess(ctx, workspaceID); err != nil {
		return nil, err
	}
	return s.store.GetConfig(ctx, workspaceID)
}

// PutConfig validates and stores the workspace's config.
func (s *Service) PutConfig(ctx context.Context, workspaceID string, req *PutConfigRequest) (*Config, error) {
	if err := s.authorizeWorkspaceAccess(ctx, workspaceID); err != nil {
		return nil, err
	}
	directories, err := normalizeDirectories(req.Directories)
	if err != nil {
		return nil, err
	}
	lock := s.workspaceLock(workspaceID)
	lock.Lock()
	defer lock.Unlock()
	existing, err := s.store.GetConfig(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	ops, err := resolveOperationSettings(existing, req)
	if err != nil {
		return nil, err
	}
	if err := s.validateBoard(ctx, workspaceID, req.WorkflowID, req.StatusSteps, &ops); err != nil {
		return nil, err
	}
	return s.saveConfig(ctx, workspaceID, req, directories, ops)
}

// saveConfig stores a validated config. The caller holds the workspace lock.
func (s *Service) saveConfig(
	ctx context.Context, workspaceID string, req *PutConfigRequest, directories []string, ops operationSettings,
) (*Config, error) {
	return s.store.UpsertConfig(ctx, &Config{
		WorkspaceID:    workspaceID,
		Enabled:        req.Enabled,
		WorkflowID:     req.WorkflowID,
		StatusSteps:    req.StatusSteps,
		Directories:    directories,
		ExecutorSteps:  ops.executorSteps,
		NotesHeading:   ops.notesHeading,
		WakeOnDate:     ops.wakeOnDate,
		StaleAfterDays: ops.staleAfterDays,
		IndexFile:      ops.indexFile,
	})
}

func invalidf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidConfig, fmt.Sprintf(format, args...))
}

// normalizeDirectories validates each directory, cleans it, and drops
// duplicates while keeping the owner's order.
func normalizeDirectories(dirs []string) ([]string, error) {
	if len(dirs) == 0 {
		return nil, invalidf("at least one directory is required")
	}
	seen := make(map[string]struct{}, len(dirs))
	cleaned := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		clean, err := scan.ValidateDirectory(dir)
		if err != nil {
			return nil, invalidf("%v", err)
		}
		if _, dup := seen[clean]; dup {
			continue
		}
		seen[clean] = struct{}{}
		cleaned = append(cleaned, clean)
	}
	return cleaned, nil
}

// validateBoard checks that the workflow belongs to the workspace, that the
// mapping covers every visible status with steps of that workflow, and that
// the executor steps are valid for it.
func (s *Service) validateBoard(
	ctx context.Context, workspaceID, workflowID string, mapping map[format.BoardStatus]string,
	ops *operationSettings,
) error {
	if strings.TrimSpace(workflowID) == "" {
		return invalidf("workflow_id is required")
	}
	workflow, err := s.workflows.GetWorkflow(ctx, workflowID)
	switch {
	case errors.Is(err, repoerrors.ErrWorkflowNotFound), errors.Is(err, repoerrors.ErrWorkspaceNotFound):
		return invalidf("workflow %q was not found in this workspace", workflowID)
	case err != nil:
		return err
	case workflow == nil || workflow.WorkspaceID != workspaceID:
		return invalidf("workflow %q was not found in this workspace", workflowID)
	}
	steps, err := s.steps.ListStepsByWorkflow(ctx, workflowID)
	if err != nil {
		return err
	}
	if err := validateMapping(mapping, steps); err != nil {
		return err
	}
	return ops.validateExecutorSteps(mapping, steps)
}

func validateMapping(mapping map[format.BoardStatus]string, steps []*wfmodels.WorkflowStep) error {
	stepIDs := make(map[string]struct{}, len(steps))
	for _, step := range steps {
		stepIDs[step.ID] = struct{}{}
	}
	visible := make(map[format.BoardStatus]struct{}, len(VisibleStatuses()))
	for _, status := range VisibleStatuses() {
		visible[status] = struct{}{}
	}
	for status := range mapping {
		if _, ok := visible[status]; !ok {
			return invalidf("status %q cannot be mapped to a step", status)
		}
	}
	for _, status := range VisibleStatuses() {
		stepID := strings.TrimSpace(mapping[status])
		if stepID == "" {
			return invalidf("status %q needs a step", status)
		}
		if _, ok := stepIDs[stepID]; !ok {
			return invalidf("step for status %q does not belong to the workflow", status)
		}
	}
	return nil
}

// CreateBoard creates a workflow from the built-in Plans template in the
// workspace and returns it with the status mapping filled in. The config is
// not saved: the owner saves it through PutConfig.
func (s *Service) CreateBoard(ctx context.Context, workspaceID string) (*CreateBoardResult, error) {
	if err := s.authorizeWorkspaceAccess(ctx, workspaceID); err != nil {
		return nil, err
	}
	tmpl, err := s.steps.GetTemplate(ctx, PlansTemplateID)
	if err != nil {
		return nil, fmt.Errorf("load plans template: %w", err)
	}
	templateID := PlansTemplateID
	workflow, err := s.workflows.CreateWorkflow(ctx, &taskservice.CreateWorkflowRequest{
		WorkspaceID:        workspaceID,
		Name:               plansBoardName,
		Description:        tmpl.Description,
		WorkflowTemplateID: &templateID,
	})
	if err != nil {
		return nil, fmt.Errorf("create plans workflow: %w", err)
	}
	mapping, err := s.mapCreatedBoard(ctx, tmpl, workflow.ID)
	if err != nil {
		s.discardBoard(ctx, workflow.ID)
		return nil, err
	}
	return &CreateBoardResult{WorkflowID: workflow.ID, StatusSteps: mapping}, nil
}

func (s *Service) mapCreatedBoard(
	ctx context.Context, tmpl *wfmodels.WorkflowTemplate, workflowID string,
) (map[format.BoardStatus]string, error) {
	created, err := s.steps.ListStepsByWorkflow(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("list plans workflow steps: %w", err)
	}
	return StatusStepsFromTemplate(tmpl, created)
}

// discardBoard removes a workflow whose steps could not be mapped, so a failed
// board creation leaves nothing behind. A failure here is only logged: the
// caller already has the original error.
func (s *Service) discardBoard(ctx context.Context, workflowID string) {
	if err := s.workflows.DeleteWorkflow(ctx, workflowID); err != nil {
		s.logger.Warn("failed to remove unusable plans workflow",
			zap.String("workflow_id", workflowID), zap.Error(err))
	}
}
