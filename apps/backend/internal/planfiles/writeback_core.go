package planfiles

import (
	"context"
	"errors"
	"path"
	"sort"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/kandev/kandev/internal/planfiles/format"
	"github.com/kandev/kandev/internal/planfiles/scan"
	taskmodels "github.com/kandev/kandev/internal/task/models"
)

// conflictNotice is shown on a task whose board edit could not be saved. It is
// task description data, not interface copy.
const conflictNotice = "board edit not saved because the file changed"

const (
	keyBoard    = "board"
	keyPriority = "priority"
	keyOrder    = "order"
)

// boardResult is the outcome of writing one board edit back to its file.
type boardResult struct {
	// outcome is empty when no write was needed.
	outcome string
	// row is the sync row after the attempt.
	row *TaskRow
	// file is the parsed file as it is on disk after the attempt.
	file format.PlanFile
	// moved is the task after it was returned to another step instead of
	// being written back; notice is the notice its row now carries.
	moved  *taskmodels.Task
	notice string
}

// writeBoardEdit writes the board edit a task shows, relative to the state its
// row records, to the plan file. The caller holds the workspace lock. The file
// is re-read and must still hash to the row's content hash; otherwise nothing
// is written and the row carries the conflict notice.
//
// A move into a step that no status maps to (a handoff step) writes nothing and
// only records the step as synced.
func (s *Service) writeBoardEdit(
	ctx context.Context, cfg *Config, root string, row *TaskRow, task *taskmodels.Task,
) boardResult {
	data, outcome := s.loadVerified(ctx, root, row)
	if outcome != "" {
		return boardResult{outcome: outcome, row: row}
	}
	pf, ok := format.Parse(path.Base(row.RelPath), data)
	if !ok {
		s.recordWriteback(writebackFailed, row, errors.New("the file is no longer a plan file"))
		return boardResult{outcome: writebackFailed, row: row}
	}
	if holder, taken := claimHolder(cfg, row, task, pf); taken {
		return s.refuseExecutorClaim(ctx, cfg, row, task, pf, holder)
	}
	keys, handoff := boardKeys(cfg, row, task, pf)
	syncBoard := func(r *TaskRow) { r.SyncedStepID, r.SyncedPriority = task.WorkflowStepID, task.Priority }
	if len(keys) == 0 {
		if handoff {
			incWriteback(s.logger, writebackHandoffSkipped)
		}
		s.saveRowChange(ctx, row, syncBoard)
		return boardResult{row: row, file: pf}
	}
	written, outcome := s.storeKeys(ctx, root, row, data, keys, syncBoard)
	if outcome != writebackWritten {
		return boardResult{outcome: outcome, row: row, file: pf}
	}
	updated, _ := format.Parse(path.Base(row.RelPath), written)
	return boardResult{outcome: outcome, row: row, file: updated}
}

// boardKeys lists the frontmatter keys a task's divergence from its row
// requires. handoff reports a step move that is deliberately not written.
func boardKeys(cfg *Config, row *TaskRow, task *taskmodels.Task, pf format.PlanFile) (keys map[string]string, handoff bool) {
	keys = map[string]string{}
	if task.WorkflowStepID != row.SyncedStepID {
		status, mapped := statusOfStep(cfg, task.WorkflowStepID, pf.Board)
		switch {
		case !mapped:
			handoff = executorEntryKeys(cfg, task.WorkflowStepID, pf, keys)
		case status != pf.Board:
			keys[keyBoard] = string(status)
		}
	}
	if task.Priority != row.SyncedPriority && task.Priority != pf.Priority {
		keys[keyPriority] = task.Priority
	}
	return keys, handoff
}

// statusOfStep returns the status a step stands for. When the step stands for
// the file's current status that status wins; otherwise the first mapped
// status in name order does.
func statusOfStep(cfg *Config, stepID string, current format.BoardStatus) (format.BoardStatus, bool) {
	if current != "" && cfg.StatusSteps[current] == stepID {
		return current, true
	}
	statuses := make([]string, 0, len(cfg.StatusSteps))
	for status, id := range cfg.StatusSteps {
		if id == stepID {
			statuses = append(statuses, string(status))
		}
	}
	if len(statuses) == 0 {
		return "", false
	}
	sort.Strings(statuses)
	return format.BoardStatus(statuses[0]), true
}

// boardDiverged reports that a plan task on the plan board no longer matches
// the step or priority its last sync applied.
func boardDiverged(cfg *Config, row *TaskRow, task *taskmodels.Task) bool {
	if task.ArchivedAt != nil || task.WorkflowID != cfg.WorkflowID {
		return false
	}
	return task.WorkflowStepID != row.SyncedStepID || task.Priority != row.SyncedPriority
}

// loadVerified reads the plan file of row. It returns the bytes only while
// they still hash to the row's content hash; a changed file records the
// conflict notice, an unreadable one counts as a failed write.
func (s *Service) loadVerified(ctx context.Context, root string, row *TaskRow) ([]byte, string) {
	file, err := scan.ReadFile(root, row.RelPath)
	if err != nil {
		s.recordWriteback(writebackFailed, row, err)
		return nil, writebackFailed
	}
	if format.ContentHash(file.Content) != row.ContentHash {
		s.markConflict(ctx, row)
		return nil, writebackConflict
	}
	return file.Content, ""
}

// markConflict records that a board edit was not saved because the file moved
// on, and counts it.
func (s *Service) markConflict(ctx context.Context, row *TaskRow) {
	incWriteback(s.logger, writebackConflict)
	s.saveRowChange(ctx, row, func(r *TaskRow) { r.Notice = conflictNotice })
}

func (s *Service) recordWriteback(outcome string, row *TaskRow, err error) {
	incWriteback(s.logger, outcome)
	s.logger.Warn("plan file write-back failed",
		zap.String("repository_id", row.RepositoryID), zap.String("rel_path", row.RelPath), zap.Error(err))
}

// storeKeys rewrites the plan file with keys set, guarded by a compare-and-swap
// on the row's content hash, and stores the row with its new hash and apply's
// changes. It returns the new bytes on success.
func (s *Service) storeKeys(
	ctx context.Context, root string, row *TaskRow, data []byte, keys map[string]string, apply func(*TaskRow),
) ([]byte, string) {
	out, err := format.SetKeys(data, keys)
	if err != nil {
		s.recordWriteback(writebackFailed, row, err)
		return nil, writebackFailed
	}
	err = scan.WriteFile(root, row.RelPath, out, row.ContentHash, format.ContentHash)
	switch {
	case errors.Is(err, scan.ErrHashMismatch):
		s.markConflict(ctx, row)
		return nil, writebackConflict
	case err != nil:
		s.recordWriteback(writebackFailed, row, err)
		return nil, writebackFailed
	}
	incWriteback(s.logger, writebackWritten)
	s.saveRowChange(ctx, row, func(r *TaskRow) {
		r.ContentHash, r.Notice = format.ContentHash(out), ""
		apply(r)
	})
	return out, writebackWritten
}

// saveRowChange applies change to row and stores it when something differs.
// A storage failure is logged: the file already holds the truth and the next
// pass converges the row.
func (s *Service) saveRowChange(ctx context.Context, row *TaskRow, change func(*TaskRow)) {
	before := *row
	change(row)
	if sameSyncState(&before, row) {
		return
	}
	row.LastSeenAt = time.Now().UTC()
	if err := s.store.UpsertTaskRow(ctx, row); err != nil {
		s.logger.Warn("plan file write-back could not store the sync row",
			zap.String("rel_path", row.RelPath), zap.Error(err))
	}
}

// repoRoots maps repository IDs of a workspace to their local paths.
func (s *Service) repoRoots(ctx context.Context, workspaceID string) (map[string]string, error) {
	repos, err := s.tasks.ListRepositories(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	roots := make(map[string]string, len(repos))
	for _, repo := range repos {
		if repo.SourceType == localSourceType && repo.LocalPath != "" {
			roots[repo.ID] = repo.LocalPath
		}
	}
	return roots, nil
}

// writeOrders writes the `order` value of each listed task to its plan file.
// The caller holds the workspace lock. Files that changed since they were last
// read are skipped with the conflict notice; the next pass restores the board.
func (s *Service) writeOrders(ctx context.Context, workspaceID string, values map[string]float64) {
	roots, err := s.repoRoots(ctx, workspaceID)
	if err != nil {
		s.logger.Warn("plan file order write-back could not list repositories", zap.Error(err))
		return
	}
	ids := make([]string, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		row, err := s.store.GetTaskRow(ctx, id)
		if err != nil || row == nil || roots[row.RepositoryID] == "" {
			continue
		}
		s.writeOrder(ctx, roots[row.RepositoryID], row, values[id])
	}
}

func (s *Service) writeOrder(ctx context.Context, root string, row *TaskRow, value float64) {
	data, outcome := s.loadVerified(ctx, root, row)
	if outcome != "" {
		return
	}
	key := newOrderKey(&value, row.RelPath, row.RepositoryID)
	keys := map[string]string{keyOrder: strconv.FormatFloat(value, 'f', -1, 64)}
	s.storeKeys(ctx, root, row, data, keys, func(r *TaskRow) { r.SyncedOrderKey = key.String() })
}
