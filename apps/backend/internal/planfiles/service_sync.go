package planfiles

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"github.com/kandev/kandev/internal/task/repository/repoerrors"
)

// errSyncNotWired reports a pass requested before SetSyncDeps ran.
var errSyncNotWired = errors.New("plan file sync dependencies are not configured")

// SyncWorkspace runs one sync pass for a workspace, waiting for any pass or
// write that holds the workspace lock. A workspace without an enabled config
// is skipped. Failures of single files and of the whole pass are recorded in
// the returned summary; the error is reserved for storage failures.
func (s *Service) SyncWorkspace(ctx context.Context, workspaceID string) (PassSummary, error) {
	unlock := s.LockWorkspace(workspaceID)
	defer unlock()
	return s.runPass(ctx, workspaceID)
}

// SyncNow is the user-facing "Sync now": it authorizes the caller, refuses to
// queue behind a running pass (ErrPassRunning), and runs one pass.
func (s *Service) SyncNow(ctx context.Context, workspaceID string) (PassSummary, error) {
	if err := s.authorizeWorkspaceAccess(ctx, workspaceID); err != nil {
		return PassSummary{}, err
	}
	cfg, err := s.store.GetConfig(ctx, workspaceID)
	if err != nil {
		return PassSummary{}, err
	}
	if cfg == nil {
		return PassSummary{}, ErrNotConfigured
	}
	unlock, ok := s.tryLockWorkspace(workspaceID)
	if !ok {
		return PassSummary{}, ErrPassRunning
	}
	defer unlock()
	return s.runPass(ctx, workspaceID)
}

// SyncDueWorkspaces runs a pass for every workspace whose sync is enabled. A
// workspace that is busy is left for the next tick.
func (s *Service) SyncDueWorkspaces(ctx context.Context) {
	configs, err := s.store.ListEnabledConfigs(ctx)
	if err != nil {
		s.logger.Warn("failed to list plan file configs", zap.Error(err))
		return
	}
	for _, cfg := range configs {
		if ctx.Err() != nil {
			return
		}
		s.syncIfIdle(ctx, cfg.WorkspaceID)
	}
}

func (s *Service) syncIfIdle(ctx context.Context, workspaceID string) {
	unlock, ok := s.tryLockWorkspace(workspaceID)
	if !ok {
		s.logger.Debug("plan file sync skipped: workspace is busy")
		return
	}
	defer unlock()
	if _, err := s.runPass(ctx, workspaceID); err != nil {
		s.logger.Warn("plan file sync failed", zap.String("workspace_id", workspaceID), zap.Error(err))
	}
}

// runPass executes one pass under the workspace lock and records its status.
func (s *Service) runPass(ctx context.Context, workspaceID string) (PassSummary, error) {
	cfg, err := s.store.GetConfig(ctx, workspaceID)
	if err != nil {
		return PassSummary{}, err
	}
	if cfg == nil || !cfg.Enabled {
		return PassSummary{Outcome: OutcomeSkipped, FileErrors: []FileErrorRow{}}, nil
	}
	if s.tasks == nil || s.archiver == nil {
		return PassSummary{}, errSyncNotWired
	}
	p := newPass(s, cfg, s.currentTime().UTC())
	runErr := p.run(ctx)
	summary := p.summary(runErr)
	statusErr := s.store.RecordPassStatus(
		ctx, workspaceID, summary.At, summary.Outcome != OutcomeFailed, summary.Counts, summary.FileErrors)
	incPass(s.logger, summary.Outcome)
	for _, fe := range summary.FileErrors {
		incFileError(s.logger, fe.Reason)
	}
	return summary, statusErr
}

// isTaskGone reports that a task read found no task.
func isTaskGone(err error) bool {
	return errors.Is(err, repoerrors.ErrTaskNotFound)
}
