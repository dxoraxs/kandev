package planfiles

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/planfiles/format"
)

const createPath = "/api/v1/plan-files/plans?workspace_id=" + testWorkspace

type createBody struct {
	TaskID       string `json:"task_id"`
	RepositoryID string `json:"repository_id"`
	RelPath      string `json:"rel_path"`
	Error        string `json:"error"`
	Code         string `json:"code"`
}

func postCreate(t *testing.T, h *syncHarness, path string, body any) (int, createBody) {
	t.Helper()
	rec := doRequest(t, newTestRouter(t, h.svc), http.MethodPost, path, body)
	var out createBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out), rec.Body.String())
	return rec.Code, out
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.4
func TestHandlers_CreatePlanAnswers201WithTheTaskOfTheNewFile(t *testing.T) {
	h := createHarness(t)

	status, body := postCreate(t, h, createPath, map[string]string{
		"repository_id": testRepoID, "directory": plansDir, "title": "Fix: totals #2"})

	assert.Equal(t, http.StatusCreated, status)
	assert.Equal(t, "docs/plans/fix-totals-2.md", body.RelPath)
	assert.Equal(t, testRepoID, body.RepositoryID)
	assert.Equal(t, h.taskFor(body.RelPath).ID, body.TaskID)
	assert.Equal(t, h.step(format.BoardQueued), h.taskFor(body.RelPath).WorkflowStepID)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.3
func TestHandlers_CreatePlanErrorCodes(t *testing.T) {
	valid := map[string]string{"repository_id": testRepoID, "directory": plansDir, "title": "Same"}
	t.Run("file exists", func(t *testing.T) {
		h := createHarness(t)
		first, _ := postCreate(t, h, createPath, valid)
		require.Equal(t, http.StatusCreated, first)
		before := h.read("docs/plans/same.md")
		status, body := postCreate(t, h, createPath, valid)
		assert.Equal(t, http.StatusConflict, status)
		assert.Equal(t, CodeFileExists, body.Code)
		assert.Equal(t, before, h.read("docs/plans/same.md"))
	})
	t.Run("invalid plan", func(t *testing.T) {
		h := createHarness(t)
		status, body := postCreate(t, h, createPath, map[string]string{
			"repository_id": testRepoID, "directory": plansDir, "title": "T", "file_name": "../x.md"})
		assert.Equal(t, http.StatusBadRequest, status)
		assert.Equal(t, CodeInvalidPlan, body.Code)
	})
	t.Run("malformed payload", func(t *testing.T) {
		h := createHarness(t)
		status, body := postCreate(t, h, createPath, "{not json")
		assert.Equal(t, http.StatusBadRequest, status)
		assert.Equal(t, CodeInvalidPlan, body.Code)
	})
	t.Run("repository not found", func(t *testing.T) {
		h := createHarness(t)
		status, body := postCreate(t, h, createPath, map[string]string{
			"repository_id": "nope", "directory": plansDir, "title": "T"})
		assert.Equal(t, http.StatusNotFound, status)
		assert.Equal(t, CodeRepositoryNotFound, body.Code)
	})
	t.Run("missing workspace id", func(t *testing.T) {
		h := createHarness(t)
		status, _ := postCreate(t, h, "/api/v1/plan-files/plans", valid)
		assert.Equal(t, http.StatusBadRequest, status)
	})
	t.Run("no config", func(t *testing.T) {
		h := createHarness(t)
		require.NoError(t, h.svc.store.DeleteConfig(context.Background(), testWorkspace))
		rec := doRequest(t, newTestRouter(t, h.svc), http.MethodPost, createPath, valid)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
