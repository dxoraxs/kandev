package handlers

import (
	"context"
	"errors"
	"strings"

	taskrepo "github.com/kandev/kandev/internal/task/repository"
	"github.com/kandev/kandev/internal/worktree"
)

const maintenanceKindRepositoryCleanup = "repository_cleanup"

// protectedListEmpty is rendered for a protected list with no entries.
const protectedListEmpty = "none"

// RepositoryWorktreeReader lists the Kandev worktree records of a repository.
type RepositoryWorktreeReader interface {
	GetAllByRepositoryID(ctx context.Context, repositoryID string) ([]*worktree.Worktree, error)
}

// SetRepositoryCleanup enables the repository cleanup maintenance kind. The
// kind stays unavailable without worktree records, because the protected list
// cannot be built without them.
func (h *TaskHandlers) SetRepositoryCleanup(enabled bool, worktrees RepositoryWorktreeReader) {
	h.repositoryCleanup = enabled
	h.repositoryWorktrees = worktrees
}

func repositoryCleanupAvailable(h *TaskHandlers) bool {
	return h.repositoryCleanup && h.repositoryWorktrees != nil
}

func repositoryCleanupVariables(ctx context.Context, h *TaskHandlers, in maintenanceInput) (map[string]string, error) {
	branches, paths, err := h.protectedWorktrees(ctx, in.repository.ID)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"repository_name":     in.repository.Name,
		"repository_path":     in.repository.LocalPath,
		"default_branch":      in.repository.DefaultBranch,
		"protected_branches":  protectedList(branches),
		"protected_worktrees": protectedList(paths),
	}, nil
}

// protectedWorktrees returns the branches and paths of the repository's
// worktrees whose task exists and is not archived. A read failure is returned,
// never skipped: a missing entry could let the agent remove a live worktree.
func (h *TaskHandlers) protectedWorktrees(ctx context.Context, repositoryID string) (branches, paths []string, err error) {
	records, err := h.repositoryWorktrees.GetAllByRepositoryID(ctx, repositoryID)
	if err != nil {
		return nil, nil, err
	}
	live := make(map[string]bool)
	for _, record := range records {
		if record == nil {
			continue
		}
		active, known := live[record.TaskID]
		if !known {
			active, err = h.taskIsLive(ctx, record.TaskID)
			if err != nil {
				return nil, nil, err
			}
			live[record.TaskID] = active
		}
		if !active {
			continue
		}
		branches = appendUnique(branches, record.Branch)
		paths = appendUnique(paths, record.Path)
	}
	return branches, paths, nil
}

func (h *TaskHandlers) taskIsLive(ctx context.Context, taskID string) (bool, error) {
	if taskID == "" {
		return false, nil
	}
	task, err := h.service.GetTask(ctx, taskID)
	if errors.Is(err, taskrepo.ErrTaskNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return task != nil && task.ArchivedAt == nil, nil
}

func appendUnique(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func protectedList(values []string) string {
	if len(values) == 0 {
		return protectedListEmpty
	}
	return strings.Join(values, "\n")
}
