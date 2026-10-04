package planfiles

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/common/logger"
	"github.com/kandev/kandev/internal/planfiles/format"
	"github.com/kandev/kandev/internal/task/repository/repoerrors"
)

func newTestRouter(t *testing.T, svc *Service) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router, svc, logger.Default())
	return router
}

func doRequest(t *testing.T, router *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	switch v := body.(type) {
	case nil:
		reader = bytes.NewReader(nil)
	case string:
		reader = bytes.NewReader([]byte(v))
	default:
		raw, err := json.Marshal(v)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

const configPath = "/api/v1/plan-files/config?workspace_id=" + testWorkspace

// @covers AC-TASKS-PLAN-FILES-005.2
func TestHandlers_PutThenGetConfig(t *testing.T) {
	svc, _ := newTestService(t)
	router := newTestRouter(t, svc)

	put := doRequest(t, router, http.MethodPut, configPath, validRequest())
	require.Equal(t, http.StatusOK, put.Code, put.Body.String())

	get := doRequest(t, router, http.MethodGet, configPath, nil)
	require.Equal(t, http.StatusOK, get.Code, get.Body.String())
	var cfg Config
	require.NoError(t, json.Unmarshal(get.Body.Bytes(), &cfg))
	assert.Equal(t, testWorkspace, cfg.WorkspaceID)
	assert.Equal(t, "wf-1", cfg.WorkflowID)
	assert.Equal(t, "dn", cfg.StatusSteps[format.BoardDone])
	assert.Equal(t, []string{"docs/plans"}, cfg.Directories)
}

func TestHandlers_GetConfigAbsentIsNotFound(t *testing.T) {
	svc, _ := newTestService(t)
	router := newTestRouter(t, svc)

	rec := doRequest(t, router, http.MethodGet, configPath, nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// @covers AC-TASKS-PLAN-FILES-005.2
func TestHandlers_NonOwnerGetsNotFound(t *testing.T) {
	svc, fake := newTestService(t)
	router := newTestRouter(t, svc)
	seeded := doRequest(t, router, http.MethodPut, configPath, validRequest())
	require.Equal(t, http.StatusOK, seeded.Code)
	svc.SetWorkspaceAuthorizer(func(context.Context, string) error { return repoerrors.ErrWorkspaceNotFound })

	cases := []struct {
		name, method, path string
		body               any
	}{
		{"get", http.MethodGet, configPath, nil},
		{"put", http.MethodPut, configPath, validRequest()},
		{"board", http.MethodPost, "/api/v1/plan-files/board?workspace_id=" + testWorkspace, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doRequest(t, router, tc.method, tc.path, tc.body)
			assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		})
	}
	assert.Empty(t, fake.created, "a denied caller must not create a workflow")
}

// @covers AC-TASKS-PLAN-FILES-005.2
func TestHandlers_PutRejectsInvalidBodies(t *testing.T) {
	svc, _ := newTestService(t)
	router := newTestRouter(t, svc)

	malformed := doRequest(t, router, http.MethodPut, configPath, "{not json")
	assert.Equal(t, http.StatusBadRequest, malformed.Code)

	badDir := validRequest()
	badDir.Directories = []string{"../outside"}
	rec := doRequest(t, router, http.MethodPut, configPath, badDir)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	foreign := validRequest()
	foreign.WorkflowID = "wf-foreign"
	rec = doRequest(t, router, http.MethodPut, configPath, foreign)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

func TestHandlers_RequireWorkspaceID(t *testing.T) {
	svc, _ := newTestService(t)
	router := newTestRouter(t, svc)

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/plan-files/config"},
		{http.MethodPut, "/api/v1/plan-files/config"},
		{http.MethodPost, "/api/v1/plan-files/board"},
	} {
		rec := doRequest(t, router, tc.method, tc.path, validRequest())
		assert.Equal(t, http.StatusBadRequest, rec.Code, tc.method+" "+tc.path)
	}
}

// @covers AC-TASKS-PLAN-FILES-005.2
func TestHandlers_CreateBoardReturnsWorkflowAndMapping(t *testing.T) {
	svc, _ := newTestService(t)
	router := newTestRouter(t, svc)

	rec := doRequest(t, router, http.MethodPost, "/api/v1/plan-files/board?workspace_id="+testWorkspace, nil)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var result CreateBoardResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, "created-1", result.WorkflowID)
	assert.Len(t, result.StatusSteps, len(VisibleStatuses()))
}
