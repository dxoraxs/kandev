package planfiles

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/kandev/kandev/internal/planfiles/format"
	taskmodels "github.com/kandev/kandev/internal/task/models"
	taskservice "github.com/kandev/kandev/internal/task/service"
	wfmodels "github.com/kandev/kandev/internal/workflow/models"
)

// keyExecutor is the frontmatter key that names the executor holding a plan.
const keyExecutor = "executor"

var errNoInProgressStep = errors.New("no step is mapped to the in-progress status")

// claimNoticePrefix starts the notice of a task that was refused an executor
// step because the plan is held by another executor. It is task description
// data, not interface copy.
const claimNoticePrefix = "plan is already taken by "

// sameExecutor compares executor names after trimming, ignoring case.
func sameExecutor(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// executorBoard reports a file status in which an executor step may hold a
// plan.
func executorBoard(board format.BoardStatus) bool {
	return board == format.BoardQueued || board == format.BoardInProgress
}

// handoffHolds reports that a task may stay in a handoff step that no status
// maps to: always for a step without an executor name, and for an executor
// step only while the file is queued or in progress and names that executor.
func handoffHolds(cfg *Config, pf format.PlanFile, stepID string) bool {
	name, ok := cfg.ExecutorSteps[stepID]
	if !ok {
		return true
	}
	return executorBoard(pf.Board) && sameExecutor(pf.Executor, name)
}

// executorEntryKeys adds the keys a move into stepID requires to keys. It
// reports handoff when the step is a handoff step that writes nothing: an
// unnamed step, or an executor step entered while the file is not queued or in
// progress.
func executorEntryKeys(cfg *Config, stepID string, pf format.PlanFile, keys map[string]string) (handoff bool) {
	name, ok := cfg.ExecutorSteps[stepID]
	if !ok || !executorBoard(pf.Board) {
		return true
	}
	if !sameExecutor(pf.Executor, name) {
		keys[keyExecutor] = executorScalar(name)
	}
	return false
}

// claimHolder returns the executor that already holds the plan when the task
// just entered an executor step for another executor.
func claimHolder(cfg *Config, row *TaskRow, task *taskmodels.Task, pf format.PlanFile) (string, bool) {
	if task.WorkflowStepID == row.SyncedStepID {
		return "", false
	}
	name, ok := cfg.ExecutorSteps[task.WorkflowStepID]
	if !ok || pf.Board != format.BoardInProgress || pf.Executor == "" || sameExecutor(pf.Executor, name) {
		return "", false
	}
	return pf.Executor, true
}

// refuseExecutorClaim returns a task that entered an executor step held by
// another executor to the in-progress step, without writing the file. The row
// records that step and the notice together, so neither the write-back nor the
// next pass reads the move back as a board edit. The caller holds the
// workspace lock.
func (s *Service) refuseExecutorClaim(
	ctx context.Context, cfg *Config, row *TaskRow, task *taskmodels.Task, pf format.PlanFile, holder string,
) boardResult {
	target := cfg.StatusSteps[format.BoardInProgress]
	if target == "" {
		s.recordWriteback(writebackFailed, row, errNoInProgressStep)
		return boardResult{outcome: writebackFailed, row: row, file: pf}
	}
	result, err := s.tasks.MoveTaskWithOptions(ctx, task.ID, cfg.WorkflowID, target, 0,
		taskservice.MoveTaskOptions{StepHistoryActor: wfmodels.StepTransitionActorSystem})
	if err != nil {
		s.recordWriteback(writebackFailed, row, err)
		return boardResult{outcome: writebackFailed, row: row, file: pf}
	}
	moved := *task
	moved.WorkflowID, moved.WorkflowStepID = cfg.WorkflowID, target
	if result != nil && result.Task != nil {
		moved = *result.Task
	}
	notice := claimNoticePrefix + holder
	s.saveRowChange(ctx, row, func(r *TaskRow) { r.SyncedStepID, r.Notice = target, notice })
	s.logger.Info("plan executor step refused: the plan is held by another executor",
		zap.String("repository_id", row.RepositoryID), zap.String("rel_path", row.RelPath))
	return boardResult{row: row, file: pf, moved: &moved, notice: notice}
}

// executorScalar renders name so that format.Parse reads it back unchanged
// from a single `key: value` line. A name that is a plain YAML string is left
// as is; any other name (a mapping, a number, a comment, an indicator) becomes
// a double-quoted scalar.
func executorScalar(name string) string {
	var decoded any
	if err := yaml.Unmarshal([]byte(name), &decoded); err == nil {
		if text, ok := decoded.(string); ok && text == name && plainLine(name) {
			return name
		}
	}
	return strconv.Quote(strings.ToValidUTF8(name, "�"))
}

// plainLine reports that s is a single line with no control characters, which
// YAML would otherwise fold or escape.
func plainLine(s string) bool {
	return strings.IndexFunc(s, func(r rune) bool { return r < 0x20 || r == 0x7f || r == 0x85 || r == 0x2028 || r == 0x2029 }) < 0
}
