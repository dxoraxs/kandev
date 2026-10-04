package planfiles

import (
	"context"
	"path"

	"go.uber.org/zap"

	"github.com/kandev/kandev/internal/planfiles/format"
	"github.com/kandev/kandev/internal/planfiles/scan"
	taskmodels "github.com/kandev/kandev/internal/task/models"
)

// UnadaptedCounts counts, per local repository of the workspace, the Markdown
// files in the scanned directories that have no board key. The configured
// directories are used, or DefaultDirectories when the workspace has no
// config. A non-empty repositoryID restricts the scan to that repository.
// Repositories with a count of zero, remote repositories, and repositories that
// cannot be scanned are omitted. Nothing is stored.
func (s *Service) UnadaptedCounts(ctx context.Context, workspaceID, repositoryID string) ([]UnadaptedRepo, error) {
	if err := s.authorizeWorkspaceAccess(ctx, workspaceID); err != nil {
		return nil, err
	}
	if s.tasks == nil {
		return nil, errSyncNotWired
	}
	cfg, err := s.store.GetConfig(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	dirs := DefaultDirectories()
	if cfg != nil {
		dirs = cfg.Directories
	}
	repos, err := s.tasks.ListRepositories(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := []UnadaptedRepo{}
	for _, repo := range repos {
		if repo.SourceType != localSourceType || repo.LocalPath == "" {
			continue
		}
		if repositoryID != "" && repo.ID != repositoryID {
			continue
		}
		if row, ok := s.unadaptedInRepo(repo, dirs); ok {
			result = append(result, row)
		}
	}
	return result, nil
}

func (s *Service) unadaptedInRepo(repo *taskmodels.Repository, dirs []string) (UnadaptedRepo, bool) {
	files, scanErrs := scan.ScanRepository(repo.LocalPath, dirs)
	for _, fe := range scanErrs {
		if fe.RelPath == "" {
			s.logger.Warn("unadapted plan file scan failed",
				zap.String("repository_id", repo.ID), zap.String("reason", string(fe.Reason)))
			return UnadaptedRepo{}, false
		}
	}
	found := map[string]struct{}{}
	count := 0
	for _, f := range files {
		if _, ok := format.Parse(path.Base(f.RelPath), f.Content); ok {
			continue
		}
		count++
		found[path.Dir(f.RelPath)] = struct{}{}
	}
	if count == 0 {
		return UnadaptedRepo{}, false
	}
	directories := sortedKeys(found)
	return UnadaptedRepo{
		RepositoryID: repo.ID, RepositoryName: repo.Name, Count: count, Directories: directories,
	}, true
}

// EnsureBoard gives a workspace without a plan-file config a Plans board and
// an enabled config over DefaultDirectories. It reports whether it created
// them; a workspace that already has a config, enabled or not, is left as is.
func (s *Service) EnsureBoard(ctx context.Context, workspaceID string) (bool, error) {
	if err := s.authorizeWorkspaceAccess(ctx, workspaceID); err != nil {
		return false, err
	}
	lock := s.workspaceLock(workspaceID)
	lock.Lock()
	defer lock.Unlock()
	cfg, err := s.store.GetConfig(ctx, workspaceID)
	if err != nil || cfg != nil {
		return false, err
	}
	board, err := s.CreateBoard(ctx, workspaceID)
	if err != nil {
		return false, err
	}
	req := &PutConfigRequest{
		Enabled: true, WorkflowID: board.WorkflowID, StatusSteps: board.StatusSteps,
		Directories: DefaultDirectories(),
	}
	ops := operationSettings{executorSteps: map[string]string{}, wakeOnDate: true, staleAfterDays: DefaultStaleAfterDays}
	if _, err := s.saveConfig(ctx, workspaceID, req, req.Directories, ops); err != nil {
		s.discardBoard(ctx, board.WorkflowID)
		return false, err
	}
	return true, nil
}
