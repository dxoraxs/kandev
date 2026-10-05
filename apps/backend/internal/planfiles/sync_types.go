package planfiles

import (
	"context"
	"errors"
	"sync"
	"time"

	taskmodels "github.com/kandev/kandev/internal/task/models"
	taskservice "github.com/kandev/kandev/internal/task/service"
)

// Pass outcomes. The set is closed; it is also the label set of
// plan_files_pass_total.
const (
	OutcomeOK      = "ok"
	OutcomePartial = "partial"
	OutcomeFailed  = "failed"
	// OutcomeSkipped reports that no pass ran: the workspace has no enabled
	// config. It is never recorded or counted.
	OutcomeSkipped = "skipped"
)

// Failure reasons that are not scanner reasons. Together with the scanner's
// reasons they form the closed label set of plan_files_file_errors_total.
const (
	ReasonParse             = "parse"
	ReasonDuplicateExternal = "duplicate_external_id"
	ReasonTaskService       = "task_service"
	ReasonWorkflowMissing   = "workflow_missing"
	ReasonRepositoryList    = "repository_list"
	ReasonWriteFailed       = "write_failed"
	ReasonIndexNotOwned     = "index_not_owned"
	ReasonUnknownDependency = "unknown_dependency"
	ReasonInvalidDependency = "invalid_dependency"
	ReasonInvalidTrack      = "invalid_track"
	ReasonGitStatusFailed   = "git_status_failed"
)

// ErrPassRunning reports that a sync pass or a write already holds the
// workspace lock.
var ErrPassRunning = errors.New("a plan file sync is already running for this workspace")

// ErrNotConfigured reports that the workspace has no plan-file config.
var ErrNotConfigured = errors.New("plan files are not configured for this workspace")

// PassSummary is the result of one sync pass.
type PassSummary struct {
	Outcome    string         `json:"outcome"`
	At         time.Time      `json:"at"`
	Counts     PassCounts     `json:"counts"`
	FileErrors []FileErrorRow `json:"file_errors"`
}

// TaskAccess is the part of the task service the sync pass drives. Satisfied by
// *taskservice.Service.
type TaskAccess interface {
	ListRepositories(ctx context.Context, workspaceID string) ([]*taskmodels.Repository, error)
	GetTask(ctx context.Context, id string) (*taskmodels.Task, error)
	GetTaskByExternalID(ctx context.Context, workspaceID, externalID string) (*taskmodels.Task, error)
	CreateTask(ctx context.Context, req *taskservice.CreateTaskRequest) (taskservice.CreateTaskResult, error)
	SettleExternalID(ctx context.Context, taskID, externalID string) (bool, *taskmodels.Task, error)
	UpdateTask(ctx context.Context, id string, req *taskservice.UpdateTaskRequest) (*taskmodels.Task, error)
	MoveTaskWithOptions(
		ctx context.Context, id, workflowID, workflowStepID string, position int, opts taskservice.MoveTaskOptions,
	) (*taskservice.MoveTaskResult, error)
	ReorderStepTasks(
		ctx context.Context, stepID, band string, orderedTaskIDs []string,
	) (*taskservice.ReorderStepTasksResult, error)
	ListTasks(ctx context.Context, workflowID string) ([]*taskmodels.Task, error)
	ListTaskSessions(ctx context.Context, taskID string) ([]*taskmodels.TaskSession, error)
	AddDependency(ctx context.Context, taskID, dependsOnTaskID string) error
	RemoveDependency(ctx context.Context, taskID, dependsOnTaskID string) error
}

// TaskArchiver archives and restores a single plan task. Satisfied by
// *taskservice.HandoffService.
type TaskArchiver interface {
	ArchiveTaskTree(ctx context.Context, rootID string, cascade bool) (*taskservice.CascadeOutcome, error)
	UnarchiveTaskTree(ctx context.Context, rootID string) (*taskservice.CascadeOutcome, error)
}

// SetSyncDeps installs the task-system collaborators of the sync pass. They are
// set after construction because the archiver is built later in startup than
// the plan-file service. Until both are set, a pass fails without side effects.
func (s *Service) SetSyncDeps(tasks TaskAccess, archiver TaskArchiver) {
	s.tasks = tasks
	s.archiver = archiver
}

// LockWorkspace blocks until the workspace's sync lock is free, takes it, and
// returns the release function. A sync pass and a board write-back hold it for
// their whole run, so the two never interleave.
func (s *Service) LockWorkspace(workspaceID string) func() {
	lock := s.passLock(workspaceID)
	lock.Lock()
	return lock.Unlock
}

// tryLockWorkspace takes the workspace lock when it is free.
func (s *Service) tryLockWorkspace(workspaceID string) (func(), bool) {
	lock := s.passLock(workspaceID)
	if !lock.TryLock() {
		return nil, false
	}
	return lock.Unlock, true
}

func (s *Service) passLock(workspaceID string) *sync.Mutex {
	lock, _ := s.passLocks.LoadOrStore(workspaceID, &sync.Mutex{})
	return lock.(*sync.Mutex)
}
