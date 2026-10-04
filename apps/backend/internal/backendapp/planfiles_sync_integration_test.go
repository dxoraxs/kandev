package backendapp

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/common/logger"
	"github.com/kandev/kandev/internal/events/bus"
	"github.com/kandev/kandev/internal/planfiles"
	"github.com/kandev/kandev/internal/planfiles/format"
	taskmodels "github.com/kandev/kandev/internal/task/models"
	taskservice "github.com/kandev/kandev/internal/task/service"
	wfmodels "github.com/kandev/kandev/internal/workflow/models"
)

const planBoardID = "wf-plans"

// planSyncFixture is the real task service, SQLite repositories, workflow
// service, and handoff service wired the way production wires them, plus a
// plan-file service and a temporary local repository.
type planSyncFixture struct {
	t        *testing.T
	h        *pluginMoveErrorBindingHarness
	taskSvc  *taskservice.Service
	handoff  *taskservice.HandoffService
	svc      *planfiles.Service
	repoRoot string
	steps    map[format.BoardStatus]string
	events   atomic.Int32
}

func newPlanSyncFixture(t *testing.T) *planSyncFixture {
	t.Helper()
	h := newPluginMoveErrorBindingHarness(t)
	ctx := context.Background()
	taskSvc, ok := h.adapter.svc.(*taskservice.Service)
	require.True(t, ok, "the harness wires the real task service")
	taskSvc.SetWorkspacePolicyAttacher(testWorkspacePolicyAttacher{})
	log, err := logger.NewLogger(logger.LoggingConfig{Level: "error", Format: "json", OutputPath: "stdout"})
	require.NoError(t, err)
	handoff := taskservice.NewHandoffService(h.repo, h.repo, taskservice.NewDocumentService(h.repo, log), nil, nil, log)
	handoff.SetTaskEventPublisher(taskSvc)

	require.NoError(t, h.repo.CreateWorkflow(ctx, &taskmodels.Workflow{ID: planBoardID, WorkspaceID: "ws-1", Name: "Plans"}))
	steps := map[format.BoardStatus]string{}
	for i, status := range planfiles.VisibleStatuses() {
		id := "plan-step-" + string(status)
		require.NoError(t, h.workflowSvc.CreateStep(ctx, &wfmodels.WorkflowStep{
			ID: id, WorkflowID: planBoardID, Name: string(status), Position: i,
		}))
		steps[status] = id
	}
	root := initLocalGitFixture(t, t.TempDir())
	resolved, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)
	_, err = taskSvc.CreateRepository(ctx, &taskservice.CreateRepositoryRequest{
		WorkspaceID: "ws-1", Name: "city", SourceType: "local", LocalPath: resolved, DefaultBranch: "main",
	})
	require.NoError(t, err)

	store, err := planfiles.NewStore(h.db, h.db)
	require.NoError(t, err)
	svc := planfiles.NewService(store, taskSvc, h.workflowSvc, log)
	svc.SetSyncDeps(taskSvc, handoff)
	_, err = svc.PutConfig(ctx, "ws-1", &planfiles.PutConfigRequest{
		Enabled: true, WorkflowID: planBoardID, StatusSteps: steps, Directories: []string{"docs/plans"},
	})
	require.NoError(t, err)

	f := &planSyncFixture{t: t, h: h, taskSvc: taskSvc, handoff: handoff, svc: svc, repoRoot: resolved, steps: steps}
	_, err = h.eventBus.Subscribe("task.*", func(context.Context, *bus.Event) error {
		f.events.Add(1)
		return nil
	})
	require.NoError(t, err)
	return f
}

func (f *planSyncFixture) writePlan(name, content string) {
	f.t.Helper()
	dir := filepath.Join(f.repoRoot, "docs", "plans")
	require.NoError(f.t, os.MkdirAll(dir, 0o755))
	require.NoError(f.t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

func (f *planSyncFixture) removePlan(name string) {
	f.t.Helper()
	require.NoError(f.t, os.Remove(filepath.Join(f.repoRoot, "docs", "plans", name)))
}

func (f *planSyncFixture) sync() planfiles.PassSummary {
	f.t.Helper()
	summary, err := f.svc.SyncWorkspace(context.Background(), "ws-1")
	require.NoError(f.t, err)
	return summary
}

func (f *planSyncFixture) taskByExternalID(ext string) *taskmodels.Task {
	f.t.Helper()
	task, err := f.taskSvc.GetTaskByExternalID(context.Background(), "ws-1", ext)
	require.NoError(f.t, err)
	return task
}

func (f *planSyncFixture) planTask(name string) *taskmodels.Task {
	return f.taskByExternalID("plan-file:" + f.repoID() + ":docs/plans/" + name)
}

func (f *planSyncFixture) repoID() string {
	f.t.Helper()
	repos, err := f.taskSvc.ListRepositories(context.Background(), "ws-1")
	require.NoError(f.t, err)
	require.Len(f.t, repos, 1)
	return repos[0].ID
}

// stepOrder returns the titles of a step's tasks in board order.
func (f *planSyncFixture) stepOrder(stepID string) []string {
	f.t.Helper()
	tasks, err := f.taskSvc.ListTasks(context.Background(), planBoardID)
	require.NoError(f.t, err)
	var inStep []*taskmodels.Task
	for _, task := range tasks {
		if task.WorkflowStepID == stepID {
			inStep = append(inStep, task)
		}
	}
	sort.SliceStable(inStep, func(i, j int) bool { return taskmodels.StepOrderLess(inStep[i], inStep[j]) })
	titles := make([]string, len(inStep))
	for i, task := range inStep {
		titles[i] = task.Title
	}
	return titles
}

func planFile(board, heading string, extra ...string) string {
	out := "---\nboard: " + board + "\n"
	for _, line := range extra {
		out += line + "\n"
	}
	return out + "---\n\n# " + heading + "\n\nBody of " + heading + ".\n"
}

// @covers AC-TASKS-PLAN-FILES-002.3
// @covers AC-TASKS-PLAN-FILES-002.5
// @covers AC-TASKS-PLAN-FILES-002.7
func TestPlanFilesSync_RealTaskServiceCreatesOrdersAndStaysQuiet(t *testing.T) {
	f := newPlanSyncFixture(t)
	ctx := context.Background()
	queued := f.steps[format.BoardQueued]
	_, err := f.taskSvc.CreateTask(ctx, &taskservice.CreateTaskRequest{
		WorkspaceID: "ws-1", WorkflowID: planBoardID, WorkflowStepID: queued, Title: "Manual task",
	})
	require.NoError(t, err)
	f.writePlan("a.md", planFile("queued", "Alpha", "order: 2", "executor: codex"))
	f.writePlan("b.md", planFile("in_progress", "Beta", "priority: high"))
	f.writePlan("c.md", planFile("queued", "Gamma", "order: 1"))

	first := f.sync()

	assert.Equal(t, planfiles.OutcomeOK, first.Outcome, "%+v", first.FileErrors)
	assert.Equal(t, 3, first.Counts.Created)
	assert.Equal(t, []string{"Manual task", "Gamma", "[codex] Alpha"}, f.stepOrder(queued))
	beta := f.planTask("b.md")
	assert.Equal(t, f.steps[format.BoardInProgress], beta.WorkflowStepID)
	assert.Equal(t, "high", beta.Priority)
	assert.Contains(t, beta.Description, "> Plan file: `city` `docs/plans/b.md`")
	require.Len(t, beta.Repositories, 1)
	before := f.events.Load()
	updatedAt := beta.UpdatedAt

	second := f.sync()

	assert.Equal(t, planfiles.PassCounts{}, second.Counts)
	assert.Equal(t, before, f.events.Load(), "an unchanged pass publishes no task event")
	assert.Equal(t, updatedAt, f.planTask("b.md").UpdatedAt)
}

// @covers AC-TASKS-PLAN-FILES-002.6
// @covers AC-TASKS-PLAN-FILES-004.2
func TestPlanFilesSync_RealTaskServiceAdoptsArchivesAndRestores(t *testing.T) {
	f := newPlanSyncFixture(t)
	ctx := context.Background()
	legacy, err := f.taskSvc.CreateTask(ctx, &taskservice.CreateTaskRequest{
		WorkspaceID: "ws-1", WorkflowID: "wf-home", WorkflowStepID: "step-home", Title: "Legacy plan",
		ExternalID: "legacy-1",
	})
	require.NoError(t, err)
	_, err = f.handoff.ArchiveTaskTree(ctx, legacy.Task.ID, false)
	require.NoError(t, err)
	f.writePlan("d.md", planFile("waiting_owner", "Delta", "external_id: legacy-1"))

	summary := f.sync()

	adopted, err := f.taskSvc.GetTask(ctx, legacy.Task.ID)
	require.NoError(t, err)
	assert.Nil(t, adopted.ArchivedAt)
	assert.Equal(t, planBoardID, adopted.WorkflowID)
	assert.Equal(t, f.steps[format.BoardWaitingOwner], adopted.WorkflowStepID)
	assert.Equal(t, "Delta", adopted.Title)
	assert.Equal(t, 0, summary.Counts.Created)
	assert.Equal(t, 1, summary.Counts.Unarchived)

	f.removePlan("d.md")
	summary = f.sync()
	archived, err := f.taskSvc.GetTask(ctx, legacy.Task.ID)
	require.NoError(t, err)
	assert.NotNil(t, archived.ArchivedAt)
	assert.Equal(t, 1, summary.Counts.Archived)

	f.writePlan("d.md", planFile("done", "Delta", "external_id: legacy-1"))
	f.sync()
	restored, err := f.taskSvc.GetTask(ctx, legacy.Task.ID)
	require.NoError(t, err)
	assert.Nil(t, restored.ArchivedAt, "the same task returns instead of a new one")
	assert.Equal(t, f.steps[format.BoardDone], restored.WorkflowStepID)
}

// @covers AC-TASKS-PLAN-FILES-002.8
func TestPlanFilesSync_RealTaskServiceDefersMoveWhileTurnRuns(t *testing.T) {
	f := newPlanSyncFixture(t)
	ctx := context.Background()
	f.writePlan("a.md", planFile("queued", "Alpha"))
	f.sync()
	task := f.planTask("a.md")
	require.NoError(t, f.h.repo.CreateTaskSession(ctx, &taskmodels.TaskSession{
		ID: "sess-run", TaskID: task.ID, State: taskmodels.TaskSessionStateRunning, IsPrimary: true,
	}))
	f.writePlan("a.md", planFile("done", "Alpha"))

	summary := f.sync()

	assert.Equal(t, f.steps[format.BoardQueued], f.planTask("a.md").WorkflowStepID)
	assert.Equal(t, 0, summary.Counts.Failed)

	require.NoError(t, f.h.repo.UpdateTaskSessionState(ctx, "sess-run", taskmodels.TaskSessionStateCompleted, ""))
	f.sync()
	assert.Equal(t, f.steps[format.BoardDone], f.planTask("a.md").WorkflowStepID)
}
