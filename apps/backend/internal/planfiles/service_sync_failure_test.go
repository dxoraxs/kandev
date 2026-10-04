package planfiles

import (
	"context"
	"errors"
	"expvar"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/planfiles/scan"
	taskmodels "github.com/kandev/kandev/internal/task/models"
)

func errorReasons(summary PassSummary) map[string]string {
	out := map[string]string{}
	for _, fe := range summary.FileErrors {
		out[fe.RelPath] = fe.Reason
	}
	return out
}

// @covers AC-TASKS-PLAN-FILES-006.4
func TestSync_UnreadableFileIsReportedAndItsTaskKept(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	h.writeFile("docs/plans/b.md", planDoc("queued", "B"))
	h.sync()
	h.writeFile("docs/plans/a.md", "---\nboard: queued\n---\n"+strings.Repeat("x", scan.MaxPlanFileBytes+1))
	h.writeFile("docs/plans/b.md", planDoc("done", "B"))

	summary := h.sync()

	assert.Equal(t, OutcomePartial, summary.Outcome)
	assert.Equal(t, string(scan.ReasonTooLarge), errorReasons(summary)["docs/plans/a.md"])
	assert.Nil(t, h.taskFor("docs/plans/a.md").ArchivedAt, "a file that cannot be read is not a deleted file")
	assert.Equal(t, h.step("done"), h.taskFor("docs/plans/b.md").WorkflowStepID, "other files still sync")
	assert.Equal(t, 0, summary.Counts.Archived)
	for _, fe := range summary.FileErrors {
		assert.Equal(t, testRepoName, fe.RepositoryName)
	}
}

// @covers AC-TASKS-PLAN-FILES-006.4
func TestSync_MissingRepositoryRootIsReportedAndTasksKept(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	h.sync()
	require.NoError(t, os.RemoveAll(h.root))

	summary := h.sync()

	assert.Equal(t, OutcomePartial, summary.Outcome)
	assert.Equal(t, string(scan.ReasonRootMissing), errorReasons(summary)[""])
	assert.Nil(t, h.taskFor("docs/plans/a.md").ArchivedAt)
}

// @covers AC-TASKS-PLAN-FILES-006.3
func TestSync_RemoteRepositoriesAreNeverScanned(t *testing.T) {
	h := newSyncHarness(t)
	remoteRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(remoteRoot, "docs/plans"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(remoteRoot, "docs/plans/r.md"), []byte(planDoc("queued", "Remote")), 0o644))
	h.tasks.repos = append(h.tasks.repos, &taskmodels.Repository{
		ID: "repo-remote", WorkspaceID: testWorkspace, Name: "remote", SourceType: "provider", LocalPath: remoteRoot,
	})
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))

	summary := h.sync()

	assert.Equal(t, 1, summary.Counts.Created)
	assert.Nil(t, h.tasks.byExternalID(defaultExternalID("repo-remote", "docs/plans/r.md")))
	assert.Equal(t, OutcomeOK, summary.Outcome)
}

// @covers AC-TASKS-PLAN-FILES-006.3
func TestSync_TasksOfARepositoryThatTurnedRemoteAreLeftAlone(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	h.sync()
	h.tasks.repos[0].SourceType = "provider"

	summary := h.sync()

	assert.Equal(t, 0, summary.Counts.Archived)
	assert.Nil(t, h.taskFor("docs/plans/a.md").ArchivedAt)
}

func TestSync_TaskServiceErrorFailsOnlyThatFile(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	h.writeFile("docs/plans/b.md", planDoc("queued", "B"))
	h.tasks.failCreate[defaultExternalID(testRepoID, "docs/plans/a.md")] = errors.New("boom")

	summary := h.sync()

	assert.Equal(t, OutcomePartial, summary.Outcome)
	assert.Equal(t, PassCounts{Created: 1, Failed: 1}, summary.Counts)
	assert.Equal(t, ReasonTaskService, errorReasons(summary)["docs/plans/a.md"])
	assert.NotNil(t, h.taskFor("docs/plans/b.md"))
}

func TestSync_MissingWorkflowFailsThePassWithoutTouchingTasks(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	delete(h.wf.workflows, "wf-1")

	summary := h.sync()

	assert.Equal(t, OutcomeFailed, summary.Outcome)
	assert.Equal(t, ReasonWorkflowMissing, summary.FileErrors[0].Reason)
	assert.Zero(t, h.tasks.writeCount())
	cfg, err := h.svc.GetConfig(context.Background(), testWorkspace)
	require.NoError(t, err)
	assert.False(t, cfg.LastPassOK)
}

// @covers AC-TASKS-PLAN-FILES-005.3
func TestSync_RecordsPassStatusForTheConfigResponse(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	h.writeFile("docs/plans/bad.md", planDoc("someday", "Bad"))

	h.sync()

	cfg, err := h.svc.GetConfig(context.Background(), testWorkspace)
	require.NoError(t, err)
	require.NotNil(t, cfg.LastPassAt)
	assert.True(t, cfg.LastPassOK)
	assert.Equal(t, 2, cfg.LastCounts.Created)
	require.Len(t, cfg.LastFileErrors, 1)
	assert.Equal(t, FileErrorRow{
		RepositoryID: testRepoID, RepositoryName: testRepoName, RelPath: "docs/plans/bad.md", Reason: ReasonParse,
	}, cfg.LastFileErrors[0])
}

// @covers AC-TASKS-PLAN-FILES-005.4
func TestSync_DisabledWorkspaceIsNotScanned(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	req := validRequest()
	req.Enabled = false
	_, err := h.svc.PutConfig(context.Background(), testWorkspace, req)
	require.NoError(t, err)

	summary := h.sync()

	assert.Equal(t, OutcomeSkipped, summary.Outcome)
	assert.Zero(t, h.tasks.writeCount())
	cfg, err := h.svc.GetConfig(context.Background(), testWorkspace)
	require.NoError(t, err)
	assert.Nil(t, cfg.LastPassAt, "a skipped pass is not recorded")
}

func TestSync_WithoutDependenciesFailsLoudly(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.PutConfig(context.Background(), testWorkspace, validRequest())
	require.NoError(t, err)

	_, err = svc.SyncWorkspace(context.Background(), testWorkspace)

	assert.ErrorIs(t, err, errSyncNotWired)
}

// @covers AC-TASKS-PLAN-FILES-002.8
func TestSync_RunningSessionAlsoDefersArchive(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	h.sync()
	id := h.taskFor("docs/plans/a.md").ID
	h.tasks.setSession(id, taskmodels.TaskSessionStateRunning)
	h.removeFile("docs/plans/a.md")

	summary := h.sync()

	assert.Nil(t, h.tasks.task(id).ArchivedAt, "archiving would cancel the running turn")
	assert.Equal(t, 0, summary.Counts.Archived)

	h.tasks.setSession(id, taskmodels.TaskSessionStateCompleted)
	summary = h.sync()

	assert.NotNil(t, h.tasks.task(id).ArchivedAt)
	assert.Equal(t, 1, summary.Counts.Archived)
}

// @covers AC-TASKS-PLAN-FILES-004.1
func TestSync_RenamedFileKeepsItsTaskThroughExternalID(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/old.md", planDoc("queued", "A", "external_id: keep-me"))
	h.sync()
	id := h.tasks.byExternalID("keep-me").ID
	h.removeFile("docs/plans/old.md")
	h.writeFile("docs/plans/new.md", planDoc("queued", "A", "external_id: keep-me"))

	summary := h.sync()

	assert.Equal(t, 0, summary.Counts.Archived)
	assert.Equal(t, 0, summary.Counts.Created)
	assert.Nil(t, h.tasks.task(id).ArchivedAt)
	row, err := h.svc.Store().GetTaskRow(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "docs/plans/new.md", row.RelPath)
}

func TestSync_DeletedTaskGetsANewOneAndStaleRowIsDropped(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	h.sync()
	oldID := h.taskFor("docs/plans/a.md").ID
	delete(h.tasks.tasks, oldID)

	summary := h.sync()

	assert.Equal(t, 1, summary.Counts.Created)
	assert.NotEqual(t, oldID, h.taskFor("docs/plans/a.md").ID)
	rows, err := h.svc.Store().ListTaskRows(context.Background(), testWorkspace)
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

// @covers AC-TASKS-PLAN-FILES-002.4
func TestSync_NoticeShowsUntilTheFileChanges(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	h.sync()
	id := h.taskFor("docs/plans/a.md").ID
	row, err := h.svc.Store().GetTaskRow(context.Background(), id)
	require.NoError(t, err)
	row.Notice = "board edit not saved"
	require.NoError(t, h.svc.Store().UpsertTaskRow(context.Background(), row))

	h.sync()
	assert.Contains(t, h.tasks.task(id).Description, "> Notice: board edit not saved")

	h.writeFile("docs/plans/a.md", planDoc("queued", "A changed"))
	h.sync()
	assert.NotContains(t, h.tasks.task(id).Description, "Notice")
	row, err = h.svc.Store().GetTaskRow(context.Background(), id)
	require.NoError(t, err)
	assert.Empty(t, row.Notice)
}

func TestSync_StoresSyncedStateForWriteBack(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("waiting_owner", "A", "priority: high", "order: 4"))

	h.sync()

	task := h.taskFor("docs/plans/a.md")
	row, err := h.svc.Store().GetTaskRow(context.Background(), task.ID)
	require.NoError(t, err)
	assert.Equal(t, h.step("waiting_owner"), row.SyncedStepID)
	assert.Equal(t, "high", row.SyncedPriority)
	assert.Equal(t, "0:4:docs/plans/a.md", row.SyncedOrderKey)
	assert.Equal(t, testRepoID, row.RepositoryID)
	assert.NotEmpty(t, row.ContentHash)
}

func counterValue(m *expvar.Map, key string) int64 {
	if v, ok := m.Get(key).(*expvar.Int); ok {
		return v.Value()
	}
	return 0
}

func TestSync_CountsPassesAndFileErrorsByClosedLabels(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/bad.md", planDoc("someday", "Bad"))
	okBefore := counterValue(passTotal, "outcome="+OutcomeOK)
	partialBefore := counterValue(passTotal, "outcome="+OutcomePartial)
	parseBefore := counterValue(fileErrorsTotal, "reason="+ReasonParse)

	h.sync()
	h.removeFile("docs/plans/bad.md")
	h.sync()

	assert.Equal(t, partialBefore+1, counterValue(passTotal, "outcome="+OutcomePartial))
	assert.Equal(t, okBefore+1, counterValue(passTotal, "outcome="+OutcomeOK))
	assert.Equal(t, parseBefore+1, counterValue(fileErrorsTotal, "reason="+ReasonParse))
}

func TestSyncDueWorkspaces_RunsEnabledAndSkipsBusyWorkspaces(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	unlock := h.svc.LockWorkspace(testWorkspace)

	h.svc.SyncDueWorkspaces(context.Background())
	assert.Zero(t, h.tasks.writeCount(), "a held lock leaves the workspace for the next tick")
	unlock()

	h.svc.SyncDueWorkspaces(context.Background())
	assert.NotNil(t, h.tasks.byExternalID(defaultExternalID(testRepoID, "docs/plans/a.md")))
}

func TestSyncNow_RefusesWhileLockedAndWhenUnconfigured(t *testing.T) {
	h := newSyncHarness(t)
	unlock := h.svc.LockWorkspace(testWorkspace)
	_, err := h.svc.SyncNow(context.Background(), testWorkspace)
	assert.ErrorIs(t, err, ErrPassRunning)
	unlock()

	_, err = h.svc.SyncNow(context.Background(), otherWS)
	assert.ErrorIs(t, err, ErrNotConfigured)

	summary, err := h.svc.SyncNow(context.Background(), testWorkspace)
	require.NoError(t, err)
	assert.Equal(t, OutcomeOK, summary.Outcome)
}
