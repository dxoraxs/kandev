package planfiles

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/common/logger"
	taskmodels "github.com/kandev/kandev/internal/task/models"
	"github.com/kandev/kandev/internal/task/repository/repoerrors"
)

const plainDoc = "# Plain plan\n\nNo frontmatter here.\n"

func writeRepoFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
}

func TestPass_CountsUnadaptedFilesWithoutErrors(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", plainDoc)
	h.writeFile("docs/plans/b.md", plainDoc)
	h.writeFile("docs/plans/c.md", plainDoc)
	h.writeFile("docs/plans/plan.md", planDoc("queued", "Real plan"))

	summary := h.sync()

	assert.Equal(t, 3, summary.Counts.Unadapted)
	assert.Equal(t, 1, summary.Counts.Created)
	assert.Equal(t, OutcomeOK, summary.Outcome)
	assert.Empty(t, summary.FileErrors)
	cfg, err := h.svc.GetConfig(context.Background(), testWorkspace)
	require.NoError(t, err)
	assert.Equal(t, 3, cfg.LastCounts.Unadapted)
}

func TestPass_UnadaptedCountResetsOnNextPass(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", plainDoc)
	assert.Equal(t, 1, h.sync().Counts.Unadapted)

	h.writeFile("docs/plans/a.md", planDoc("queued", "Adapted"))

	assert.Equal(t, 0, h.sync().Counts.Unadapted)
}

type unadaptedFixture struct {
	svc   *Service
	local string
	clean string
}

func newUnadaptedFixture(t *testing.T) *unadaptedFixture {
	t.Helper()
	svc, _ := newTestService(t)
	tasks := newFakeTaskSystem()
	local, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	clean, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	tasks.repos = []*taskmodels.Repository{
		{ID: "r-local", WorkspaceID: testWorkspace, Name: "local-repo", SourceType: "local", LocalPath: local},
		{ID: "r-clean", WorkspaceID: testWorkspace, Name: "clean-repo", SourceType: "local", LocalPath: clean},
		{ID: "r-remote", WorkspaceID: testWorkspace, Name: "remote-repo", SourceType: "provider", LocalPath: ""},
	}
	svc.SetSyncDeps(tasks, tasks)
	writeRepoFile(t, local, "docs/plans/a.md", plainDoc)
	writeRepoFile(t, local, "docs/superpowers/plans/b.md", plainDoc)
	writeRepoFile(t, local, "docs/superpowers/plans/ok.md", planDoc("queued", "Adapted"))
	writeRepoFile(t, local, "docs/other/ignored.md", plainDoc)
	writeRepoFile(t, clean, "docs/plans/ok.md", planDoc("done", "Adapted"))
	return &unadaptedFixture{svc: svc, local: local, clean: clean}
}

func TestUnadaptedCounts_UsesDefaultDirectoriesWithoutConfig(t *testing.T) {
	f := newUnadaptedFixture(t)

	got, err := f.svc.UnadaptedCounts(context.Background(), testWorkspace, "")

	require.NoError(t, err)
	require.Len(t, got, 1, "remote and zero-count repositories are omitted")
	assert.Equal(t, UnadaptedRepo{
		RepositoryID: "r-local", RepositoryName: "local-repo", Count: 2,
		Directories: []string{"docs/plans", "docs/superpowers/plans"},
	}, got[0])
}

func TestUnadaptedCounts_UsesConfiguredDirectories(t *testing.T) {
	f := newUnadaptedFixture(t)
	_, err := f.svc.PutConfig(context.Background(), testWorkspace, validRequest())
	require.NoError(t, err)

	got, err := f.svc.UnadaptedCounts(context.Background(), testWorkspace, "")

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, 1, got[0].Count)
	assert.Equal(t, []string{"docs/plans"}, got[0].Directories)
}

func TestUnadaptedCounts_FiltersByRepository(t *testing.T) {
	f := newUnadaptedFixture(t)

	got, err := f.svc.UnadaptedCounts(context.Background(), testWorkspace, "r-clean")
	require.NoError(t, err)
	assert.Empty(t, got)

	got, err = f.svc.UnadaptedCounts(context.Background(), testWorkspace, "r-local")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "r-local", got[0].RepositoryID)
}

func TestUnadaptedCounts_StoresNothing(t *testing.T) {
	f := newUnadaptedFixture(t)

	_, err := f.svc.UnadaptedCounts(context.Background(), testWorkspace, "")
	require.NoError(t, err)

	cfg, err := f.svc.GetConfig(context.Background(), testWorkspace)
	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestUnadaptedCounts_ForeignWorkspaceIsNotFound(t *testing.T) {
	f := newUnadaptedFixture(t)
	f.svc.SetWorkspaceAuthorizer(func(context.Context, string) error { return repoerrors.ErrWorkspaceNotFound })

	_, err := f.svc.UnadaptedCounts(context.Background(), testWorkspace, "")

	assert.ErrorIs(t, err, repoerrors.ErrWorkspaceNotFound)
}

func TestHandlers_UnadaptedReturnsRepositories(t *testing.T) {
	f := newUnadaptedFixture(t)
	router := newTestRouter(t, f.svc)

	rec := doRequest(t, router, http.MethodGet, "/api/v1/plan-files/unadapted?workspace_id="+testWorkspace, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"repositories":[{"repository_id":"r-local","repository_name":"local-repo","count":2,
		"directories":["docs/plans","docs/superpowers/plans"]}]}`, rec.Body.String())
}

func TestHandlers_UnadaptedEmptyIsEmptyList(t *testing.T) {
	f := newUnadaptedFixture(t)
	router := newTestRouter(t, f.svc)

	rec := doRequest(t, router, http.MethodGet,
		"/api/v1/plan-files/unadapted?workspace_id="+testWorkspace+"&repository_id=r-clean", nil)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"repositories":[]}`, rec.Body.String())
}

func TestHandlers_UnadaptedRequiresWorkspaceAndHidesForeign(t *testing.T) {
	f := newUnadaptedFixture(t)
	router := newTestRouter(t, f.svc)
	rec := doRequest(t, router, http.MethodGet, "/api/v1/plan-files/unadapted", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	f.svc.SetWorkspaceAuthorizer(func(context.Context, string) error { return repoerrors.ErrWorkspaceNotFound })
	rec = doRequest(t, router, http.MethodGet, "/api/v1/plan-files/unadapted?workspace_id="+testWorkspace, nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestEnsureBoard_CreatesBoardAndEnabledConfigOnce(t *testing.T) {
	svc, fake := newTestService(t)
	ctx := context.Background()

	created, err := svc.EnsureBoard(ctx, "ws-new")
	require.NoError(t, err)
	assert.True(t, created)
	cfg, err := svc.GetConfig(ctx, "ws-new")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, DefaultDirectories(), cfg.Directories)
	assert.Equal(t, "created-1", cfg.WorkflowID)
	assert.Len(t, fake.created, 1)

	created, err = svc.EnsureBoard(ctx, "ws-new")
	require.NoError(t, err)
	assert.False(t, created)
	assert.Len(t, fake.created, 1, "a second call must not create another board")
}

func TestEnsureBoard_LeavesExistingDisabledConfigAlone(t *testing.T) {
	svc, fake := newTestService(t)
	ctx := context.Background()
	req := validRequest()
	req.Enabled = false
	_, err := svc.PutConfig(ctx, testWorkspace, req)
	require.NoError(t, err)

	created, err := svc.EnsureBoard(ctx, testWorkspace)

	require.NoError(t, err)
	assert.False(t, created)
	assert.Empty(t, fake.created)
	cfg, err := svc.GetConfig(ctx, testWorkspace)
	require.NoError(t, err)
	assert.False(t, cfg.Enabled)
	assert.Equal(t, "wf-1", cfg.WorkflowID)
}

func TestEnsureBoard_ForeignWorkspaceIsNotFound(t *testing.T) {
	svc, fake := newTestService(t)
	svc.SetWorkspaceAuthorizer(func(context.Context, string) error { return repoerrors.ErrWorkspaceNotFound })

	_, err := svc.EnsureBoard(context.Background(), testWorkspace)

	assert.ErrorIs(t, err, repoerrors.ErrWorkspaceNotFound)
	assert.Empty(t, fake.created)
}

func TestEnsureBoard_DiscardsCreatedBoardWhenSavingConfigFails(t *testing.T) {
	fake := newFakeWorkflows(t)
	store := setupTestStore(t)
	_, err := store.db.Exec(`CREATE TRIGGER reject_config BEFORE INSERT ON plan_file_configs
		BEGIN SELECT RAISE(ABORT, 'insert rejected'); END`)
	require.NoError(t, err)
	svc := NewService(store, fake, fake, logger.Default())

	created, err := svc.EnsureBoard(context.Background(), "ws-new")

	require.Error(t, err)
	assert.False(t, created)
	assert.Equal(t, []string{"created-1"}, fake.deleted, "the board created for the config is removed")
	assert.NotContains(t, fake.workflows, "created-1")
}
