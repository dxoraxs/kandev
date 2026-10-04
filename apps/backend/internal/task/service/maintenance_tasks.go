package service

import (
	"context"

	"github.com/kandev/kandev/internal/task/models"
)

// Task metadata keys that tie a task to the repository maintenance action that
// created it.
const (
	MetaKeyMaintenanceKind         = "repository_maintenance_kind"
	MetaKeyMaintenanceRepositoryID = "repository_maintenance_repository_id"
)

const maintenanceTaskPageSize = 100

// FindActiveMaintenanceTask returns the non-archived task of the given kind for
// the repository whose latest session is not terminal, together with that
// session's ID. A task that has no session yet (for example after a failed
// launch) is active and yields an empty session ID. It returns nil when no
// such task exists.
func (s *Service) FindActiveMaintenanceTask(
	ctx context.Context, workspaceID, repositoryID, kind string,
) (*models.Task, string, error) {
	for page := 1; ; page++ {
		tasks, total, err := s.ListTasksByWorkspace(
			ctx, workspaceID, "", repositoryID, "", page, maintenanceTaskPageSize, "", false, false, false, false,
		)
		if err != nil {
			return nil, "", err
		}
		for _, task := range tasks {
			if !isMaintenanceTask(task, kind, repositoryID) {
				continue
			}
			sessionID, active, err := s.latestSessionActivity(ctx, task.ID)
			if err != nil {
				return nil, "", err
			}
			if active {
				return task, sessionID, nil
			}
		}
		if len(tasks) == 0 || page*maintenanceTaskPageSize >= total {
			return nil, "", nil
		}
	}
}

func isMaintenanceTask(task *models.Task, kind, repositoryID string) bool {
	return task.Metadata != nil &&
		models.StringFromAny(task.Metadata[MetaKeyMaintenanceKind]) == kind &&
		models.StringFromAny(task.Metadata[MetaKeyMaintenanceRepositoryID]) == repositoryID
}

// latestSessionActivity reports the latest session of a task and whether it is
// still live. A task without sessions counts as active.
func (s *Service) latestSessionActivity(ctx context.Context, taskID string) (string, bool, error) {
	sessions, err := s.ListTaskSessions(ctx, taskID)
	if err != nil {
		return "", false, err
	}
	var latest *models.TaskSession
	for _, session := range sessions {
		if latest == nil || session.StartedAt.After(latest.StartedAt) {
			latest = session
		}
	}
	if latest == nil {
		return "", true, nil
	}
	switch latest.State {
	case models.TaskSessionStateCompleted, models.TaskSessionStateFailed, models.TaskSessionStateCancelled:
		return latest.ID, false, nil
	}
	return latest.ID, true, nil
}

// FirstMaintenanceWorkflow returns the workspace's first visible workflow by
// sort order, skipping excludeWorkflowID (the plan board). It returns nil when
// none qualifies.
func (s *Service) FirstMaintenanceWorkflow(
	ctx context.Context, workspaceID, excludeWorkflowID string,
) (*models.Workflow, error) {
	workflows, err := s.ListWorkflows(ctx, workspaceID, false)
	if err != nil {
		return nil, err
	}
	for _, workflow := range workflows {
		if workflow.ID != excludeWorkflowID {
			return workflow, nil
		}
	}
	return nil, nil
}
