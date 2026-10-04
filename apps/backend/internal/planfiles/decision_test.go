package planfiles

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/planfiles/format"
	"github.com/kandev/kandev/internal/task/repository/repoerrors"
	wfmodels "github.com/kandev/kandev/internal/workflow/models"
)

const (
	decisionPlan = "docs/plans/a.md"
	decisionDay  = "2026-10-05"
)

// waitingPlan seeds one plan file waiting for the owner, syncs it, and fixes
// the clock of the service.
func waitingPlan(t *testing.T, content string) *syncHarness {
	t.Helper()
	h := newSyncHarness(t)
	h.svc.SetClock(func() time.Time { return time.Date(2026, 10, 5, 12, 0, 0, 0, time.Local) })
	h.writeFile(decisionPlan, content)
	h.sync()
	require.Equal(t, h.step(format.BoardWaitingOwner), h.taskFor(decisionPlan).WorkflowStepID)
	h.tasks.moves = nil
	return h
}

func (h *syncHarness) decide(req DecisionRequest) (DecisionResult, error) {
	h.t.Helper()
	return h.svc.Decide(context.Background(), h.taskFor(decisionPlan).ID, req)
}

func requireDecisionCode(t *testing.T, err error, code string) {
	t.Helper()
	var decisionErr *DecisionError
	require.True(t, errors.As(err, &decisionErr), "got %v", err)
	assert.Equal(t, code, decisionErr.Code)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-002.2
func TestDecide_ReturnOnFileWithoutNotesSectionAddsTheSectionAndMovesToQueue(t *testing.T) {
	h := waitingPlan(t, planDoc("waiting_owner", "A"))

	result, err := h.decide(DecisionRequest{Action: DecisionReturn, Comment: "fix the totals"})

	require.NoError(t, err)
	assert.Equal(t, format.BoardQueued, result.Board)
	assert.Equal(t, "---\nboard: queued\n---\n\n# A\n\nBody of A.\n\n## Owner notes\n\n- "+decisionDay+
		" returned: fix the totals\n", h.read(decisionPlan))
	assert.Equal(t, h.step(format.BoardQueued), h.taskFor(decisionPlan).WorkflowStepID)
	require.Len(t, h.tasks.moves, 1)
	assert.Equal(t, wfmodels.StepTransitionActorSystem, h.tasks.moves[0].Actor)
	row := h.row(decisionPlan)
	assert.Equal(t, h.step(format.BoardQueued), row.SyncedStepID)
	assert.Equal(t, format.ContentHash([]byte(h.read(decisionPlan))), row.ContentHash)
	assert.Empty(t, row.Notice)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-002.2
func TestDecide_AcceptDefaultsToDoneAndKeepsAnEmptyCommentOutOfTheNote(t *testing.T) {
	h := waitingPlan(t, planDoc("waiting_owner", "A"))

	result, err := h.decide(DecisionRequest{Action: DecisionAccept})

	require.NoError(t, err)
	assert.Equal(t, format.BoardDone, result.Board)
	assert.Equal(t, "---\nboard: done\n---\n\n# A\n\nBody of A.\n\n## Owner notes\n\n- "+decisionDay+" accepted\n",
		h.read(decisionPlan))
	assert.Equal(t, h.step(format.BoardDone), h.taskFor(decisionPlan).WorkflowStepID)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-002.2
func TestDecide_AcceptBackToQueueWithComment(t *testing.T) {
	h := waitingPlan(t, planDoc("waiting_owner", "A"))

	result, err := h.decide(DecisionRequest{Action: DecisionAccept, Result: "queued", Comment: "  looks good\r\n\n go on "})

	require.NoError(t, err)
	assert.Equal(t, format.BoardQueued, result.Board)
	assert.Contains(t, h.read(decisionPlan), "- "+decisionDay+" accepted: looks good  go on\n")
	assert.Equal(t, h.step(format.BoardQueued), h.taskFor(decisionPlan).WorkflowStepID)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-002.4
func TestDecide_AppendsToAnExistingNotesSection(t *testing.T) {
	content := planDoc("waiting_owner", "A") + "\n## Owner notes\n\n- 2026-10-01 returned: first\n\n## Tail\n\nx\n"
	h := waitingPlan(t, content)

	_, err := h.decide(DecisionRequest{Action: DecisionReturn, Comment: "second"})

	require.NoError(t, err)
	want := strings.Replace(content, "board: waiting_owner", "board: queued", 1)
	want = strings.Replace(want, "first\n", "first\n- "+decisionDay+" returned: second\n", 1)
	assert.Equal(t, want, h.read(decisionPlan))
}

// @covers AC-TASKS-PLAN-BOARD-OPS-002.4
func TestDecide_CRLFFileKeepsItsLineEndings(t *testing.T) {
	content := strings.ReplaceAll(planDoc("waiting_owner", "A"), "\n", "\r\n")
	h := waitingPlan(t, content)

	_, err := h.decide(DecisionRequest{Action: DecisionReturn, Comment: "again"})

	require.NoError(t, err)
	got := h.read(decisionPlan)
	assert.Equal(t, strings.Count(got, "\n"), strings.Count(got, "\r\n"), "every line ends with CRLF")
	assert.Contains(t, got, "board: queued\r\n")
	assert.Contains(t, got, "## Owner notes\r\n\r\n- "+decisionDay+" returned: again\r\n")
}

// @covers AC-TASKS-PLAN-BOARD-OPS-002.4
func TestDecide_UsesTheConfiguredNotesHeading(t *testing.T) {
	h := waitingPlan(t, planDoc("waiting_owner", "A"))
	req := validRequest()
	req.NotesHeading = ptr("Review log")
	_, err := h.svc.PutConfig(context.Background(), testWorkspace, req)
	require.NoError(t, err)

	_, err = h.decide(DecisionRequest{Action: DecisionReturn, Comment: "redo"})

	require.NoError(t, err)
	assert.Contains(t, h.read(decisionPlan), "\n## Review log\n\n- "+decisionDay+" returned: redo\n")
	assert.NotContains(t, h.read(decisionPlan), "Owner notes")
}

// @covers AC-TASKS-PLAN-BOARD-OPS-002.3
func TestDecide_FileEditedSinceTheLastReadWritesNothingAndKeepsTheNotice(t *testing.T) {
	h := waitingPlan(t, planDoc("waiting_owner", "A"))
	h.writeFile(decisionPlan, planDoc("waiting_owner", "A", "owner: me"))
	snap := h.snapshot(decisionPlan)

	_, err := h.decide(DecisionRequest{Action: DecisionReturn, Comment: "x"})

	requireDecisionCode(t, err, CodeFileChanged)
	h.assertUnchanged(snap)
	assert.Empty(t, h.tasks.moves)
	assert.Empty(t, h.row(decisionPlan).Notice)
	assert.Equal(t, h.step(format.BoardWaitingOwner), h.taskFor(decisionPlan).WorkflowStepID)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-002.1
func TestDecide_TaskOutsideTheWaitingOwnerStepIsRefused(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile(decisionPlan, planDoc("queued", "A"))
	h.sync()
	snap := h.snapshot(decisionPlan)

	_, err := h.decide(DecisionRequest{Action: DecisionAccept})

	requireDecisionCode(t, err, CodeNotWaitingOwner)
	h.assertUnchanged(snap)
}

func TestDecide_TaskWithoutAPlanRowIsNotAPlanTask(t *testing.T) {
	h := newSyncHarness(t)

	_, err := h.svc.Decide(context.Background(), "no-such-task", DecisionRequest{Action: DecisionAccept})

	requireDecisionCode(t, err, CodeNotPlanTask)
}

func TestDecide_DeniedWorkspaceIsRefusedBeforeAnythingElse(t *testing.T) {
	h := waitingPlan(t, planDoc("waiting_owner", "A"))
	h.svc.SetWorkspaceAuthorizer(func(context.Context, string) error { return repoerrors.ErrWorkspaceNotFound })
	snap := h.snapshot(decisionPlan)

	_, err := h.decide(DecisionRequest{Action: DecisionReturn})

	assert.True(t, workspaceDenied(err), "got %v", err)
	h.assertUnchanged(snap)
	assert.Empty(t, h.tasks.moves)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-002.5
func TestDecide_InvalidRequestsWriteNothing(t *testing.T) {
	cases := map[string]DecisionRequest{
		"return without a comment":     {Action: DecisionReturn},
		"return with a blank comment":  {Action: DecisionReturn, Comment: " \r\n "},
		"unknown action":               {Action: "approve"},
		"unknown result":               {Action: DecisionAccept, Result: "waiting_owner"},
		"comment of 501 characters":    {Action: DecisionAccept, Comment: strings.Repeat("a", 501)},
		"comment of 501 wide runes":    {Action: DecisionAccept, Comment: strings.Repeat("é", 501)},
		"empty action":                 {},
		"return with an overlong note": {Action: DecisionReturn, Comment: strings.Repeat("b", 600)},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			h := waitingPlan(t, planDoc("waiting_owner", "A"))
			snap := h.snapshot(decisionPlan)

			_, err := h.decide(req)

			requireDecisionCode(t, err, CodeInvalidDecision)
			h.assertUnchanged(snap)
			assert.Empty(t, h.tasks.moves)
		})
	}
}

func TestDecide_CommentOfExactlyTheLimitIsAccepted(t *testing.T) {
	h := waitingPlan(t, planDoc("waiting_owner", "A"))

	_, err := h.decide(DecisionRequest{Action: DecisionAccept, Comment: strings.Repeat("é", 500)})

	require.NoError(t, err)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-002.2
func TestDecide_NextPassAndWriteBackSeeNoDivergence(t *testing.T) {
	h := waitingPlan(t, planDoc("waiting_owner", "A"))
	_, err := h.decide(DecisionRequest{Action: DecisionReturn, Comment: "redo"})
	require.NoError(t, err)
	snap := h.snapshot(decisionPlan)
	h.tasks.moves = nil

	summary := h.sync()
	h.writeBackTask(decisionPlan)

	h.assertUnchanged(snap)
	assert.Empty(t, h.tasks.moves)
	assert.Equal(t, 0, summary.Counts.Moved)
	assert.Equal(t, h.step(format.BoardQueued), h.taskFor(decisionPlan).WorkflowStepID)
}

func TestDecide_FailedMoveLeavesTheFileForTheNextPassToConverge(t *testing.T) {
	h := waitingPlan(t, planDoc("waiting_owner", "A"))
	h.tasks.setSession(h.taskFor(decisionPlan).ID, "RUNNING")

	result, err := h.decide(DecisionRequest{Action: DecisionReturn, Comment: "redo"})

	require.NoError(t, err)
	assert.Equal(t, format.BoardQueued, result.Board)
	assert.Contains(t, h.read(decisionPlan), "board: queued")
	assert.Equal(t, h.step(format.BoardWaitingOwner), h.row(decisionPlan).SyncedStepID)
	assert.NotEqual(t, format.ContentHash([]byte(h.read(decisionPlan))), h.row(decisionPlan).ContentHash)
}
