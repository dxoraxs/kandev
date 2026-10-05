package planfiles

import (
	"context"
	"sort"
	"time"

	"go.uber.org/zap"

	"github.com/kandev/kandev/internal/planfiles/format"
	taskmodels "github.com/kandev/kandev/internal/task/models"
)

const (
	cardDisplayKey = "card_display"
	dateLayout     = "2006-01-02"
)

// WorkspaceLister lists the workspaces visible to the caller. Satisfied by
// *taskservice.Service.
type WorkspaceLister interface {
	ListWorkspaces(ctx context.Context) ([]*taskmodels.Workspace, error)
}

// SetWorkspaceLister installs the workspace source of the cross-workspace
// waiting-owner list.
func (s *Service) SetWorkspaceLister(lister WorkspaceLister) {
	s.workspaces = lister
}

// WaitingItem is one plan that waits for the owner.
type WaitingItem struct {
	WorkspaceID    string `json:"workspace_id"`
	WorkspaceName  string `json:"workspace_name"`
	TaskID         string `json:"task_id"`
	Title          string `json:"title"`
	RepositoryName string `json:"repository_name"`
	RelPath        string `json:"rel_path"`
	Date           string `json:"date"`
	Executor       string `json:"executor"`
	Priority       string `json:"priority"`
}

// FailedWorkspace names a workspace whose waiting plans could not be read.
type FailedWorkspace struct {
	WorkspaceID   string `json:"workspace_id"`
	WorkspaceName string `json:"workspace_name"`
}

// WaitingOwnerResult is the response of GET /plan-files/waiting-owner.
type WaitingOwnerResult struct {
	Items            []WaitingItem     `json:"items"`
	FailedWorkspaces []FailedWorkspace `json:"failed_workspaces"`
}

// WaitingOwner lists the plan tasks that sit in the waiting_owner step of every
// workspace the caller may access and whose plan files are enabled. A workspace
// the caller may not access, or one without an enabled board, contributes
// nothing; a workspace whose reads fail is reported in FailedWorkspaces and
// never hides the others. Items are ordered by date ascending with undated
// plans last, then by title, workspace name, and task ID.
func (s *Service) WaitingOwner(ctx context.Context) (WaitingOwnerResult, error) {
	if s.tasks == nil || s.workspaces == nil {
		return WaitingOwnerResult{}, errSyncNotWired
	}
	workspaces, err := s.workspaces.ListWorkspaces(ctx)
	if err != nil {
		return WaitingOwnerResult{}, err
	}
	result := WaitingOwnerResult{Items: []WaitingItem{}, FailedWorkspaces: []FailedWorkspace{}}
	for _, ws := range workspaces {
		if ws == nil {
			continue
		}
		items, include, err := s.waitingInWorkspace(ctx, ws)
		switch {
		case err != nil:
			s.logger.Warn("waiting plans of a workspace could not be read", zap.String("workspace_id", ws.ID), zap.Error(err))
			result.FailedWorkspaces = append(result.FailedWorkspaces, FailedWorkspace{WorkspaceID: ws.ID, WorkspaceName: ws.Name})
		case include:
			result.Items = append(result.Items, items...)
		}
	}
	sortWaiting(result.Items)
	return result, nil
}

// waitingInWorkspace reads one workspace. include is false for a workspace that
// is not accessible or has no enabled board with a waiting_owner step.
func (s *Service) waitingInWorkspace(ctx context.Context, ws *taskmodels.Workspace) ([]WaitingItem, bool, error) {
	if err := s.authorizeWorkspaceAccess(ctx, ws.ID); err != nil {
		if workspaceDenied(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	cfg, err := s.store.GetConfig(ctx, ws.ID)
	if err != nil {
		return nil, false, err
	}
	if cfg == nil || !cfg.Enabled || cfg.StatusSteps[format.BoardWaitingOwner] == "" {
		return nil, false, nil
	}
	rows, err := s.store.ListTaskRows(ctx, ws.ID)
	if err != nil {
		return nil, false, err
	}
	live, names, err := s.liveWaiting(ctx, ws.ID, cfg)
	if err != nil {
		return nil, false, err
	}
	items := make([]WaitingItem, 0, len(live))
	for _, row := range rows {
		if task, ok := live[row.TaskID]; ok {
			items = append(items, waitingItem(ws, row, task, names[row.RepositoryID]))
		}
	}
	return items, true, nil
}

// liveWaiting reads the unarchived tasks of the board that sit in the
// waiting_owner step, by task ID, and the repository names by repository ID.
func (s *Service) liveWaiting(
	ctx context.Context, workspaceID string, cfg *Config,
) (map[string]*taskmodels.Task, map[string]string, error) {
	tasks, err := s.tasks.ListTasks(ctx, cfg.WorkflowID)
	if err != nil {
		return nil, nil, err
	}
	repos, err := s.tasks.ListRepositories(ctx, workspaceID)
	if err != nil {
		return nil, nil, err
	}
	waitingStep := cfg.StatusSteps[format.BoardWaitingOwner]
	live := make(map[string]*taskmodels.Task, len(tasks))
	for _, task := range tasks {
		if task.WorkflowStepID == waitingStep && task.ArchivedAt == nil {
			live[task.ID] = task
		}
	}
	names := make(map[string]string, len(repos))
	for _, repo := range repos {
		names[repo.ID] = repo.Name
	}
	return live, names, nil
}

func waitingItem(ws *taskmodels.Workspace, row *TaskRow, task *taskmodels.Task, repoName string) WaitingItem {
	date, executor := cardFacts(task.Metadata)
	return WaitingItem{
		WorkspaceID: ws.ID, WorkspaceName: ws.Name, TaskID: task.ID, Title: task.Title,
		RepositoryName: repoName, RelPath: row.RelPath, Date: date, Executor: executor,
		Priority: task.Priority,
	}
}

// cardFacts reads the date and executor name from the task's card_display
// metadata. A value that is not a real calendar date or a non-empty name reads
// as absent.
func cardFacts(metadata map[string]interface{}) (date, executor string) {
	card, _ := metadata[cardDisplayKey].(map[string]interface{})
	if raw, ok := card["date"].(string); ok {
		if _, err := time.Parse(dateLayout, raw); err == nil {
			date = raw
		}
	}
	if exec, ok := card["executor"].(map[string]interface{}); ok {
		name, _ := exec["name"].(string)
		executor = oneLine(name)
	}
	return date, executor
}

func sortWaiting(items []WaitingItem) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if (a.Date == "") != (b.Date == "") {
			return a.Date != ""
		}
		if a.Date != b.Date {
			return a.Date < b.Date
		}
		if a.Title != b.Title {
			return a.Title < b.Title
		}
		if a.WorkspaceName != b.WorkspaceName {
			return a.WorkspaceName < b.WorkspaceName
		}
		return a.TaskID < b.TaskID
	})
}
