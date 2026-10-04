package planfiles

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/task/repository/repoerrors"
)

const syncPath = "/api/v1/plan-files/sync?workspace_id=" + testWorkspace

// @covers AC-TASKS-PLAN-FILES-005.3
func TestHandlers_SyncRunsOnePassAndConfigShowsItsStatus(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	h.writeFile("docs/plans/bad.md", planDoc("someday", "Bad"))
	router := newTestRouter(t, h.svc)

	rec := doRequest(t, router, http.MethodPost, syncPath, nil)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var summary PassSummary
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &summary))
	assert.Equal(t, OutcomePartial, summary.Outcome)
	assert.Equal(t, 2, summary.Counts.Created)
	require.Len(t, summary.FileErrors, 1)
	assert.Equal(t, "docs/plans/bad.md", summary.FileErrors[0].RelPath)

	get := doRequest(t, router, http.MethodGet, configPath, nil)
	require.Equal(t, http.StatusOK, get.Code)
	var cfg Config
	require.NoError(t, json.Unmarshal(get.Body.Bytes(), &cfg))
	require.NotNil(t, cfg.LastPassAt)
	assert.Equal(t, 2, cfg.LastCounts.Created)
	assert.Len(t, cfg.LastFileErrors, 1)
}

func TestHandlers_SyncIsConflictWhileThePassLockIsHeld(t *testing.T) {
	h := newSyncHarness(t)
	router := newTestRouter(t, h.svc)
	unlock := h.svc.LockWorkspace(testWorkspace)

	rec := doRequest(t, router, http.MethodPost, syncPath, nil)
	unlock()

	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Equal(t, http.StatusOK, doRequest(t, router, http.MethodPost, syncPath, nil).Code)
}

func TestHandlers_SyncWithoutConfigIsNotFound(t *testing.T) {
	svc, _ := newTestService(t)
	router := newTestRouter(t, svc)

	rec := doRequest(t, router, http.MethodPost, syncPath, nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandlers_SyncRequiresWorkspaceID(t *testing.T) {
	svc, _ := newTestService(t)
	router := newTestRouter(t, svc)

	rec := doRequest(t, router, http.MethodPost, "/api/v1/plan-files/sync", nil)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// @covers AC-TASKS-PLAN-FILES-005.3
func TestHandlers_SyncDeniedWorkspaceIsNotFoundAndRunsNothing(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	router := newTestRouter(t, h.svc)
	h.svc.SetWorkspaceAuthorizer(func(context.Context, string) error { return repoerrors.ErrWorkspaceNotFound })

	rec := doRequest(t, router, http.MethodPost, syncPath, nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Zero(t, h.tasks.writeCount())
}

func TestHandlers_SyncStorageFailureIsGeneric500(t *testing.T) {
	h := newSyncHarness(t)
	h.svc.SetSyncDeps(nil, nil)
	router := newTestRouter(t, h.svc)

	rec := doRequest(t, router, http.MethodPost, syncPath, nil)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "dependencies")
}
