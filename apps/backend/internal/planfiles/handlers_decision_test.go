package planfiles

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/planfiles/format"
	"github.com/kandev/kandev/internal/task/repository/repoerrors"
)

func decisionURL(taskID string) string {
	return "/api/v1/plan-files/tasks/" + taskID + "/decision"
}

type decisionBody struct {
	Board format.BoardStatus `json:"board"`
	Error string             `json:"error"`
	Code  string             `json:"code"`
}

func postDecision(t *testing.T, h *syncHarness, taskID string, body any) (int, decisionBody) {
	t.Helper()
	rec := doRequest(t, newTestRouter(t, h.svc), http.MethodPost, decisionURL(taskID), body)
	var out decisionBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out), rec.Body.String())
	return rec.Code, out
}

// @covers AC-TASKS-PLAN-BOARD-OPS-002.2
func TestHandlers_DecisionReturnsTheNewBoardAndMovesTheTask(t *testing.T) {
	h := waitingPlan(t, planDoc("waiting_owner", "A"))

	status, body := postDecision(t, h, h.taskFor(decisionPlan).ID,
		map[string]string{"action": "return", "comment": "fix the totals"})

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, format.BoardQueued, body.Board)
	assert.Equal(t, h.step(format.BoardQueued), h.taskFor(decisionPlan).WorkflowStepID)
	assert.Contains(t, h.read(decisionPlan), "returned: fix the totals")
}

func TestHandlers_DecisionAcceptWithResultQueued(t *testing.T) {
	h := waitingPlan(t, planDoc("waiting_owner", "A"))

	status, body := postDecision(t, h, h.taskFor(decisionPlan).ID,
		map[string]string{"action": "accept", "result": "queued"})

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, format.BoardQueued, body.Board)
}

func TestHandlers_DecisionErrorCodes(t *testing.T) {
	t.Run("unknown task", func(t *testing.T) {
		h := waitingPlan(t, planDoc("waiting_owner", "A"))
		status, body := postDecision(t, h, "nope", map[string]string{"action": "accept"})
		assert.Equal(t, http.StatusNotFound, status)
		assert.Equal(t, CodeNotPlanTask, body.Code)
	})
	t.Run("not waiting for the owner", func(t *testing.T) {
		h := newSyncHarness(t)
		h.writeFile(decisionPlan, planDoc("queued", "A"))
		h.sync()
		status, body := postDecision(t, h, h.taskFor(decisionPlan).ID, map[string]string{"action": "accept"})
		assert.Equal(t, http.StatusConflict, status)
		assert.Equal(t, CodeNotWaitingOwner, body.Code)
	})
	t.Run("file changed", func(t *testing.T) {
		h := waitingPlan(t, planDoc("waiting_owner", "A"))
		h.writeFile(decisionPlan, planDoc("waiting_owner", "A", "owner: me"))
		status, body := postDecision(t, h, h.taskFor(decisionPlan).ID, map[string]string{"action": "accept"})
		assert.Equal(t, http.StatusConflict, status)
		assert.Equal(t, CodeFileChanged, body.Code)
	})
	t.Run("invalid decision", func(t *testing.T) {
		h := waitingPlan(t, planDoc("waiting_owner", "A"))
		status, body := postDecision(t, h, h.taskFor(decisionPlan).ID, map[string]string{"action": "return"})
		assert.Equal(t, http.StatusBadRequest, status)
		assert.Equal(t, CodeInvalidDecision, body.Code)
		assert.NotEmpty(t, body.Error)
	})
	t.Run("malformed body", func(t *testing.T) {
		h := waitingPlan(t, planDoc("waiting_owner", "A"))
		status, body := postDecision(t, h, h.taskFor(decisionPlan).ID, "{not json")
		assert.Equal(t, http.StatusBadRequest, status)
		assert.Equal(t, CodeInvalidDecision, body.Code)
	})
}

func TestHandlers_DecisionDeniedWorkspaceIsNotFoundAndWritesNothing(t *testing.T) {
	h := waitingPlan(t, planDoc("waiting_owner", "A"))
	h.svc.SetWorkspaceAuthorizer(func(context.Context, string) error { return repoerrors.ErrWorkspaceNotFound })
	snap := h.snapshot(decisionPlan)

	status, _ := postDecision(t, h, h.taskFor(decisionPlan).ID, map[string]string{"action": "accept"})

	assert.Equal(t, http.StatusNotFound, status)
	h.assertUnchanged(snap)
	assert.Empty(t, h.tasks.moves)
}

func TestDecide_CountsTheAppliedAction(t *testing.T) {
	h := waitingPlan(t, planDoc("waiting_owner", "A"))
	count := func() int64 {
		v, _ := decisionTotal.Get("action=return").(interface{ Value() int64 })
		if v == nil {
			return 0
		}
		return v.Value()
	}
	before := count()

	_, err := h.decide(DecisionRequest{Action: DecisionReturn, Comment: "x"})

	require.NoError(t, err)
	assert.Equal(t, before+1, count())
}
