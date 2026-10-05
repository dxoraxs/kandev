package planfiles

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/planfiles/format"
	taskmodels "github.com/kandev/kandev/internal/task/models"
	"github.com/kandev/kandev/internal/task/repository/repoerrors"
)

const (
	waitingURL = "/api/v1/plan-files/waiting-owner"
	thirdWS    = "ws-3"
	fourthWS   = "ws-4"
)

type fakeWorkspaceLister struct {
	workspaces []*taskmodels.Workspace
	err        error
}

func (f fakeWorkspaceLister) ListWorkspaces(context.Context) ([]*taskmodels.Workspace, error) {
	return f.workspaces, f.err
}

// failingWorkspaceTasks fails task reads of the workflows it lists.
type failingWorkspaceTasks struct {
	*fakeTaskSystem
	failWorkflows map[string]bool
}

func (f failingWorkspaceTasks) ListTasks(ctx context.Context, workflowID string) ([]*taskmodels.Task, error) {
	if f.failWorkflows[workflowID] {
		return nil, errors.New("database is locked")
	}
	return f.fakeTaskSystem.ListTasks(ctx, workflowID)
}

type waitingFixture struct {
	t     *testing.T
	svc   *Service
	tasks *fakeTaskSystem
	seq   int
}

func newWaitingFixture(t *testing.T) *waitingFixture {
	t.Helper()
	svc, _ := newTestService(t)
	tasks := newFakeTaskSystem()
	tasks.repos = []*taskmodels.Repository{
		{ID: "repo-1", WorkspaceID: testWorkspace, Name: "kandev", SourceType: "local", LocalPath: "/x"},
		{ID: "repo-2", WorkspaceID: otherWS, Name: "dmhive", SourceType: "local", LocalPath: "/y"},
	}
	svc.SetSyncDeps(tasks, tasks)
	f := &waitingFixture{t: t, svc: svc, tasks: tasks}
	f.workspaces(
		&taskmodels.Workspace{ID: testWorkspace, Name: "kandev"},
		&taskmodels.Workspace{ID: otherWS, Name: "dmhive"},
	)
	return f
}

func (f *waitingFixture) workspaces(ws ...*taskmodels.Workspace) {
	f.svc.SetWorkspaceLister(fakeWorkspaceLister{workspaces: ws})
}

// board stores an enabled config whose waiting_owner step is "<workflow>-wo".
func (f *waitingFixture) board(workspaceID, workflowID string, enabled bool) {
	f.t.Helper()
	cfg := sampleConfig(workspaceID)
	cfg.Enabled, cfg.WorkflowID = enabled, workflowID
	cfg.StatusSteps = map[format.BoardStatus]string{
		format.BoardWaitingOwner: workflowID + "-wo", format.BoardQueued: workflowID + "-q",
	}
	_, err := f.svc.store.UpsertConfig(context.Background(), cfg)
	require.NoError(f.t, err)
}

// plan seeds a plan task in step and a sync row for it. meta is the task
// metadata.
func (f *waitingFixture) plan(workspaceID, workflowID, step, title, repo string, meta map[string]any) string {
	f.t.Helper()
	f.seq++
	id := "t" + string(rune('a'+f.seq))
	task := f.tasks.seedTask(id, workflowID, step, "")
	live := f.tasks.tasks[task.ID]
	live.WorkspaceID, live.Title, live.Metadata = workspaceID, title, meta
	row := sampleTaskRow(id, "docs/plans/"+id+".md")
	row.WorkspaceID, row.RepositoryID, row.SyncedStepID = workspaceID, repo, step
	row.ExternalID = "plan-file:" + id
	require.NoError(f.t, f.svc.store.UpsertTaskRow(context.Background(), row))
	return id
}

func cardMeta(date, executor string) map[string]any {
	card := map[string]any{}
	if date != "" {
		card["date"] = date
	}
	if executor != "" {
		card["executor"] = map[string]any{"name": executor, "kind": "agent"}
	}
	return map[string]any{"card_display": card}
}

func titles(items []WaitingItem) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Title)
	}
	return out
}

// @covers AC-TASKS-PLAN-BOARD-OPS-009.1
func TestWaitingOwner_ListsBothWorkspacesSortedByDateWithUndatedLast(t *testing.T) {
	f := newWaitingFixture(t)
	f.board(testWorkspace, "wf-1", true)
	f.board(otherWS, "wf-foreign", true)
	f.plan(testWorkspace, "wf-1", "wf-1-wo", "Undated B", "repo-1", nil)
	f.plan(otherWS, "wf-foreign", "wf-foreign-wo", "Late", "repo-2", cardMeta("2026-11-01", "Claude"))
	f.plan(testWorkspace, "wf-1", "wf-1-wo", "Early", "repo-1", cardMeta("2026-10-05", ""))
	f.plan(otherWS, "wf-foreign", "wf-foreign-wo", "Undated A", "repo-2", nil)

	out, err := f.svc.WaitingOwner(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"Early", "Late", "Undated A", "Undated B"}, titles(out.Items))
	assert.Empty(t, out.FailedWorkspaces)
	late := out.Items[1]
	assert.Equal(t, WaitingItem{
		WorkspaceID: otherWS, WorkspaceName: "dmhive", TaskID: late.TaskID, Title: "Late",
		RepositoryName: "dmhive", RelPath: late.RelPath, Date: "2026-11-01", Executor: "Claude",
		Priority: taskmodels.TaskPriorityMedium,
	}, late)
	assert.Equal(t, "kandev", out.Items[0].RepositoryName)
}

func TestWaitingOwner_SkipsPlansOutsideTheWaitingStepAndInvalidDates(t *testing.T) {
	f := newWaitingFixture(t)
	f.board(testWorkspace, "wf-1", true)
	f.plan(testWorkspace, "wf-1", "wf-1-q", "Queued", "repo-1", nil)
	f.plan(testWorkspace, "wf-1", "wf-1-wo", "Bad date", "repo-1", cardMeta("2026-13-45", ""))
	moved := f.plan(testWorkspace, "wf-1", "wf-1-wo", "Moved away", "repo-1", nil)
	f.tasks.tasks[moved].WorkflowStepID = "wf-1-q"

	out, err := f.svc.WaitingOwner(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"Bad date"}, titles(out.Items))
	assert.Empty(t, out.Items[0].Date)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-009.1
func TestWaitingOwner_TiesBreakByTitleThenWorkspaceThenTask(t *testing.T) {
	f := newWaitingFixture(t)
	f.board(testWorkspace, "wf-1", true)
	f.board(otherWS, "wf-foreign", true)
	first := f.plan(otherWS, "wf-foreign", "wf-foreign-wo", "Same", "repo-2", cardMeta("2026-10-05", ""))
	second := f.plan(testWorkspace, "wf-1", "wf-1-wo", "Same", "repo-1", cardMeta("2026-10-05", ""))
	third := f.plan(testWorkspace, "wf-1", "wf-1-wo", "Same", "repo-1", cardMeta("2026-10-05", ""))
	f.plan(testWorkspace, "wf-1", "wf-1-wo", "Aaa", "repo-1", cardMeta("2026-10-05", ""))

	out, err := f.svc.WaitingOwner(context.Background())

	require.NoError(t, err)
	require.Len(t, out.Items, 4)
	assert.Equal(t, "Aaa", out.Items[0].Title)
	// "dmhive" sorts before "kandev"; ties inside a workspace fall to the task ID.
	assert.Equal(t, []string{first, second, third}, []string{
		out.Items[1].TaskID, out.Items[2].TaskID, out.Items[3].TaskID,
	})
}

// @covers AC-TASKS-PLAN-BOARD-OPS-009.1
func TestWaitingOwner_OmitsWorkspacesTheCallerCannotAccessWithoutListingThemAsFailed(t *testing.T) {
	f := newWaitingFixture(t)
	f.board(testWorkspace, "wf-1", true)
	f.board(otherWS, "wf-foreign", true)
	f.plan(testWorkspace, "wf-1", "wf-1-wo", "Mine", "repo-1", nil)
	f.plan(otherWS, "wf-foreign", "wf-foreign-wo", "Forbidden plan", "repo-2", nil)
	f.svc.SetWorkspaceAuthorizer(func(_ context.Context, id string) error {
		if id == otherWS {
			return repoerrors.ErrWorkspaceNotFound
		}
		return nil
	})

	out, err := f.svc.WaitingOwner(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"Mine"}, titles(out.Items))
	assert.Empty(t, out.FailedWorkspaces)
}

func TestWaitingOwner_OmitsDisabledAndUnconfiguredWorkspacesWithoutError(t *testing.T) {
	f := newWaitingFixture(t)
	f.workspaces(
		&taskmodels.Workspace{ID: testWorkspace, Name: "kandev"},
		&taskmodels.Workspace{ID: otherWS, Name: "dmhive"},
		&taskmodels.Workspace{ID: thirdWS, Name: "none"},
	)
	f.board(testWorkspace, "wf-1", false)
	f.board(otherWS, "wf-foreign", true)
	f.plan(testWorkspace, "wf-1", "wf-1-wo", "Disabled", "repo-1", nil)

	out, err := f.svc.WaitingOwner(context.Background())

	require.NoError(t, err)
	assert.Empty(t, out.Items)
	assert.Empty(t, out.FailedWorkspaces)
}

func TestWaitingOwner_BoardWithoutWaitingOwnerStepContributesNothing(t *testing.T) {
	f := newWaitingFixture(t)
	cfg := sampleConfig(testWorkspace)
	_, err := f.svc.store.UpsertConfig(context.Background(), cfg)
	require.NoError(t, err)
	f.plan(testWorkspace, "wf-1", "", "Odd", "repo-1", nil)

	out, err := f.svc.WaitingOwner(context.Background())

	require.NoError(t, err)
	assert.Empty(t, out.Items)
	assert.Empty(t, out.FailedWorkspaces)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-009.3
func TestWaitingOwner_FailedWorkspaceIsNamedAndOthersStillListed(t *testing.T) {
	f := newWaitingFixture(t)
	f.board(testWorkspace, "wf-1", true)
	f.board(otherWS, "wf-foreign", true)
	f.plan(testWorkspace, "wf-1", "wf-1-wo", "Mine", "repo-1", nil)
	f.plan(otherWS, "wf-foreign", "wf-foreign-wo", "Hidden by failure", "repo-2", nil)
	f.svc.SetSyncDeps(failingWorkspaceTasks{f.tasks, map[string]bool{"wf-foreign": true}}, f.tasks)

	out, err := f.svc.WaitingOwner(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"Mine"}, titles(out.Items))
	assert.Equal(t, []FailedWorkspace{{WorkspaceID: otherWS, WorkspaceName: "dmhive"}}, out.FailedWorkspaces)
}

func TestWaitingOwner_NotWiredAndListerFailureAreErrors(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.WaitingOwner(context.Background())
	require.Error(t, err)

	f := newWaitingFixture(t)
	f.svc.SetWorkspaceLister(fakeWorkspaceLister{err: errors.New("boom")})
	_, err = f.svc.WaitingOwner(context.Background())
	require.Error(t, err)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-009.1
func TestHandlers_WaitingOwnerEmptyListsAreArrays(t *testing.T) {
	f := newWaitingFixture(t)

	rec := doRequest(t, newTestRouter(t, f.svc), http.MethodGet, waitingURL, nil)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"items":[],"failed_workspaces":[]}`, rec.Body.String())
}

// @covers AC-TASKS-PLAN-BOARD-OPS-009.1
func TestHandlers_WaitingOwnerReturnsItemsWithTheDocumentedShape(t *testing.T) {
	f := newWaitingFixture(t)
	f.board(testWorkspace, "wf-1", true)
	id := f.plan(testWorkspace, "wf-1", "wf-1-wo", "Plan", "repo-1", cardMeta("2026-10-05", "Claude"))

	rec := doRequest(t, newTestRouter(t, f.svc), http.MethodGet, waitingURL, nil)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Items, 1)
	item := body.Items[0]
	assert.Equal(t, id, item["task_id"])
	assert.Equal(t, "kandev", item["workspace_name"])
	assert.Equal(t, "kandev", item["repository_name"])
	assert.Equal(t, "2026-10-05", item["date"])
	assert.Equal(t, "Claude", item["executor"])
	for _, key := range []string{"workspace_id", "title", "rel_path", "priority"} {
		assert.Contains(t, item, key)
	}
}

func TestHandlers_WaitingOwnerHidesAForbiddenWorkspacePlans(t *testing.T) {
	f := newWaitingFixture(t)
	f.board(testWorkspace, "wf-1", true)
	f.board(otherWS, "wf-foreign", true)
	f.plan(testWorkspace, "wf-1", "wf-1-wo", "Visible plan", "repo-1", nil)
	f.plan(otherWS, "wf-foreign", "wf-foreign-wo", "Secret plan", "repo-2", nil)
	f.svc.SetWorkspaceAuthorizer(func(_ context.Context, id string) error {
		if id == otherWS {
			return repoerrors.ErrWorkspaceNotFound
		}
		return nil
	})

	rec := doRequest(t, newTestRouter(t, f.svc), http.MethodGet, waitingURL, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Visible plan")
	assert.NotContains(t, rec.Body.String(), "Secret plan")
	assert.NotContains(t, rec.Body.String(), "dmhive")
}

func TestHandlers_WaitingOwnerWithoutWiringIs500(t *testing.T) {
	svc, _ := newTestService(t)

	rec := doRequest(t, newTestRouter(t, svc), http.MethodGet, waitingURL, nil)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
