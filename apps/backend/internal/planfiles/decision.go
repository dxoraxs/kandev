package planfiles

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"go.uber.org/zap"

	"github.com/kandev/kandev/internal/planfiles/format"
	"github.com/kandev/kandev/internal/planfiles/scan"
	taskmodels "github.com/kandev/kandev/internal/task/models"
	taskservice "github.com/kandev/kandev/internal/task/service"
	wfmodels "github.com/kandev/kandev/internal/workflow/models"
)

// Decision actions and results. The actions are the label set of
// plan_files_decision_total.
const (
	DecisionAccept = "accept"
	DecisionReturn = "return"

	resultDone   = "done"
	resultQueued = "queued"

	maxDecisionCommentChars = 500
	noteDateLayout          = "2006-01-02"
)

// Decision error codes. They are the machine-readable codes of the HTTP API.
const (
	CodeNotPlanTask     = "not_plan_task"
	CodeNotWaitingOwner = "not_waiting_owner"
	CodeFileChanged     = "file_changed"
	CodeInvalidDecision = "invalid_decision"
)

// DecisionError is a rejected decision with a machine-readable code.
type DecisionError struct {
	Code    string
	Message string
}

func (e *DecisionError) Error() string { return e.Message }

var (
	// ErrNotPlanTask reports that no plan file backs the task.
	ErrNotPlanTask = &DecisionError{Code: CodeNotPlanTask, Message: "the task is not a plan file task"}
	// ErrNotWaitingOwner reports that the task is not in the waiting-owner step.
	ErrNotWaitingOwner = &DecisionError{Code: CodeNotWaitingOwner, Message: "the plan is not waiting for the owner"}
	// ErrFileChanged reports that the plan file changed since it was last read.
	ErrFileChanged = &DecisionError{Code: CodeFileChanged, Message: "the plan file changed since it was last read"}
)

func invalidDecision(format string, args ...any) error {
	return &DecisionError{Code: CodeInvalidDecision, Message: fmt.Sprintf(format, args...)}
}

// DecisionRequest is the body of POST /plan-files/tasks/:taskId/decision.
type DecisionRequest struct {
	Action  string `json:"action"`
	Result  string `json:"result,omitempty"`
	Comment string `json:"comment,omitempty"`
}

// DecisionResult is the board status the decision wrote.
type DecisionResult struct {
	Board format.BoardStatus `json:"board"`
}

var lineBreakRun = regexp.MustCompile(`[\r\n]+`)

// SetClock replaces the clock of the service: owner notes, new file names, and
// every sync pass read the current time from it.
func (s *Service) SetClock(clock func() time.Time) {
	s.clock = clock
}

// currentTime is the service clock, time.Now unless SetClock replaced it.
func (s *Service) currentTime() time.Time {
	if s.clock != nil {
		return s.clock()
	}
	return time.Now()
}

func (s *Service) today() string {
	return dayOf(s.currentTime())
}

// decisionEdit is a validated decision: the status to write and the note line.
type decisionEdit struct {
	action string
	board  format.BoardStatus
	note   string
}

// planDecision validates req and builds the status and note it writes.
func planDecision(req DecisionRequest, today string) (decisionEdit, error) {
	comment := strings.TrimSpace(lineBreakRun.ReplaceAllString(strings.TrimSpace(req.Comment), " "))
	if utf8.RuneCountInString(comment) > maxDecisionCommentChars {
		return decisionEdit{}, invalidDecision("the comment is longer than %d characters", maxDecisionCommentChars)
	}
	switch req.Action {
	case DecisionAccept:
		board := format.BoardDone
		switch req.Result {
		case "", resultDone:
		case resultQueued:
			board = format.BoardQueued
		default:
			return decisionEdit{}, invalidDecision("unknown result %q", req.Result)
		}
		note := "- " + today + " accepted"
		if comment != "" {
			note += ": " + comment
		}
		return decisionEdit{action: DecisionAccept, board: board, note: note}, nil
	case DecisionReturn:
		if comment == "" {
			return decisionEdit{}, invalidDecision("a returned plan needs a comment")
		}
		return decisionEdit{
			action: DecisionReturn, board: format.BoardQueued, note: "- " + today + " returned: " + comment,
		}, nil
	default:
		return decisionEdit{}, invalidDecision("unknown action %q", req.Action)
	}
}

// Decide applies the owner's decision on a plan waiting for them: it writes the
// new board status and a dated note to the plan file in one compare-and-swap
// write, moves the task to the step mapped to the status, and records both in
// the sync row so neither the write-back nor the next pass reads them as edits.
func (s *Service) Decide(ctx context.Context, taskID string, req DecisionRequest) (DecisionResult, error) {
	row, err := s.store.GetTaskRow(ctx, taskID)
	if err != nil {
		return DecisionResult{}, err
	}
	if row == nil {
		return DecisionResult{}, ErrNotPlanTask
	}
	if err := s.authorizeWorkspaceAccess(ctx, row.WorkspaceID); err != nil {
		return DecisionResult{}, err
	}
	if s.tasks == nil {
		return DecisionResult{}, errSyncNotWired
	}
	unlock := s.LockWorkspace(row.WorkspaceID)
	defer unlock()
	return s.decideLocked(ctx, taskID, req)
}

// decideLocked runs a decision with the workspace lock held.
func (s *Service) decideLocked(ctx context.Context, taskID string, req DecisionRequest) (DecisionResult, error) {
	row, err := s.store.GetTaskRow(ctx, taskID)
	if err != nil {
		return DecisionResult{}, err
	}
	if row == nil {
		return DecisionResult{}, ErrNotPlanTask
	}
	cfg, err := s.store.GetConfig(ctx, row.WorkspaceID)
	if err != nil {
		return DecisionResult{}, err
	}
	if cfg == nil || !cfg.Enabled {
		return DecisionResult{}, ErrNotConfigured
	}
	task, err := s.tasks.GetTask(ctx, taskID)
	if err != nil {
		return DecisionResult{}, err
	}
	if waiting := cfg.StatusSteps[format.BoardWaitingOwner]; waiting == "" || task.WorkflowStepID != waiting {
		return DecisionResult{}, ErrNotWaitingOwner
	}
	roots, err := s.repoRoots(ctx, row.WorkspaceID)
	if err != nil {
		return DecisionResult{}, err
	}
	root := roots[row.RepositoryID]
	if root == "" {
		return DecisionResult{}, fmt.Errorf("repository %s has no local path", row.RepositoryID)
	}
	file, err := scan.ReadFile(root, row.RelPath)
	if err != nil {
		return DecisionResult{}, err
	}
	if format.ContentHash(file.Content) != row.ContentHash {
		return DecisionResult{}, ErrFileChanged
	}
	edit, err := planDecision(req, s.today())
	if err != nil {
		return DecisionResult{}, err
	}
	return s.applyDecision(ctx, cfg, root, row, task, file.Content, edit)
}

// applyDecision writes the edit and moves the task. A move that fails leaves the
// row untouched: the file then differs from the row, so the next pass reads it
// as an edit and moves the task.
func (s *Service) applyDecision(
	ctx context.Context, cfg *Config, root string, row *TaskRow, task *taskmodels.Task,
	content []byte, edit decisionEdit,
) (DecisionResult, error) {
	out, err := composeEdit(content, map[string]string{keyBoard: string(edit.board)}, notesHeading(cfg), edit.note)
	if err != nil {
		return DecisionResult{}, fmt.Errorf("compose plan decision: %w", err)
	}
	err = scan.WriteFile(root, row.RelPath, out, row.ContentHash, format.ContentHash)
	if errors.Is(err, scan.ErrHashMismatch) {
		return DecisionResult{}, ErrFileChanged
	}
	if err != nil {
		return DecisionResult{}, err
	}
	incDecision(s.logger, edit.action)
	target := cfg.StatusSteps[edit.board]
	if target != task.WorkflowStepID {
		_, err = s.tasks.MoveTaskWithOptions(ctx, task.ID, cfg.WorkflowID, target, 0,
			taskservice.MoveTaskOptions{StepHistoryActor: wfmodels.StepTransitionActorSystem})
		if err != nil {
			s.logger.Warn("plan decision written but the task could not be moved",
				zap.String("rel_path", row.RelPath), zap.Error(err))
			return DecisionResult{Board: edit.board}, nil
		}
	}
	s.saveRowChange(ctx, row, func(r *TaskRow) {
		r.ContentHash, r.SyncedStepID, r.Notice = format.ContentHash(out), target, ""
	})
	return DecisionResult{Board: edit.board}, nil
}
