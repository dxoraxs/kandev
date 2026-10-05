package backendapp

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/common/logger"
	officesqlite "github.com/kandev/kandev/internal/office/repository/sqlite"
	"github.com/kandev/kandev/internal/orchestrator"
	"github.com/kandev/kandev/internal/planfiles"
	taskmodels "github.com/kandev/kandev/internal/task/models"
	taskservice "github.com/kandev/kandev/internal/task/service"
	wfmodels "github.com/kandev/kandev/internal/workflow/models"
	v1 "github.com/kandev/kandev/pkg/api/v1"
)

// gateSpy wraps the task service as the orchestrator's dependency reader and
// records every verdict of the auto-start gate.
type gateSpy struct {
	*taskservice.Service
	mu       sync.Mutex
	verdicts map[string][]bool
}

func (g *gateSpy) DependencyGate(ctx context.Context, taskID string) (bool, string, error) {
	blocked, reason, err := g.Service.DependencyGate(ctx, taskID)
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.verdicts == nil {
		g.verdicts = map[string][]bool{}
	}
	g.verdicts[taskID] = append(g.verdicts[taskID], blocked)
	return blocked, reason, err
}

func (g *gateSpy) verdictsFor(taskID string) []bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]bool(nil), g.verdicts[taskID]...)
}

// withDependencyStore wires the same task_blockers store production wires into
// the task service.
func (f *planSyncFixture) withDependencyStore() *planSyncFixture {
	f.t.Helper()
	officeRepo, err := officesqlite.NewWithDB(f.h.db, f.h.db, nil)
	require.NoError(f.t, err)
	f.taskSvc.SetBlockerRepository(officeRepo)
	return f
}

func (f *planSyncFixture) dependencyIDs(taskID string) []string {
	f.t.Helper()
	task, err := f.taskSvc.GetTask(context.Background(), taskID)
	require.NoError(f.t, err)
	view := f.taskSvc.BuildDependencyViews(context.Background(), []*taskmodels.Task{task})[taskID]
	ids := make([]string, 0, len(view.DependsOn))
	for _, ref := range view.DependsOn {
		ids = append(ids, ref.ID)
	}
	return ids
}

// A plan that depends on an unfinished plan is blocked through the real task
// service; a cycle is rejected for one file and the pass completes.
func TestPlanFilesDependencies_RealTaskServiceBlocksAndRejectsCycles(t *testing.T) {
	f := newPlanSyncFixture(t).withDependencyStore()
	f.writePlan("a.md", planFile("queued", "A"))
	f.writePlan("b.md", planFile("queued", "B", "depends_on: [a.md]"))
	f.writePlan("c.md", planFile("queued", "C", "depends_on: [d.md]"))
	f.writePlan("d.md", planFile("queued", "D", "depends_on: [c.md]"))

	summary := f.sync()

	a, b := f.planTask("a.md"), f.planTask("b.md")
	assert.Equal(t, []string{a.ID}, f.dependencyIDs(b.ID))
	blocked, reason, err := f.taskSvc.DependencyGate(context.Background(), b.ID)
	require.NoError(t, err)
	assert.True(t, blocked)
	assert.Equal(t, taskservice.BlockedReasonPending, reason)
	require.Len(t, summary.FileErrors, 1)
	assert.Equal(t, "docs/plans/d.md", summary.FileErrors[0].RelPath)
	assert.Equal(t, planfiles.ReasonInvalidDependency, summary.FileErrors[0].Reason)
	assert.Equal(t, []string{f.planTask("d.md").ID}, f.dependencyIDs(f.planTask("c.md").ID))

	f.writePlan("b.md", planFile("queued", "B"))
	f.sync()
	assert.Empty(t, f.dependencyIDs(b.ID))
	blocked, _, err = f.taskSvc.DependencyGate(context.Background(), b.ID)
	require.NoError(t, err)
	assert.False(t, blocked)
}

// A blocked plan task moved into a step with agent auto-start is held by the
// gate the orchestrator consults before every automated launch, and no session
// is created for it.
func TestPlanFilesDependencies_BlockedPlanTaskMovedIntoAutoStartStepStartsNoAgent(t *testing.T) {
	f := newPlanSyncFixture(t).withDependencyStore()
	ctx := context.Background()
	f.writePlan("a.md", planFile("queued", "A"))
	f.writePlan("b.md", planFile("queued", "B", "depends_on: [a.md]"))
	f.sync()
	autoStep := "plan-step-auto"
	require.NoError(t, f.h.workflowSvc.CreateStep(ctx, &wfmodels.WorkflowStep{
		ID: autoStep, WorkflowID: planBoardID, Name: "Auto", Position: 20,
		Events: wfmodels.StepEvents{OnEnter: []wfmodels.OnEnterAction{{Type: wfmodels.OnEnterAutoStartAgent}}},
	}))
	spy := &gateSpy{Service: f.taskSvc}
	log, err := logger.NewLogger(logger.LoggingConfig{Level: "error", Format: "json", OutputPath: "stdout"})
	require.NoError(t, err)
	orch := orchestrator.NewService(orchestrator.DefaultServiceConfig(), f.h.eventBus, nil,
		&taskRepositoryAdapter{repo: f.h.repo, svc: f.taskSvc}, f.h.repo, nil, nil, nil, log)
	orch.SetTaskDependencyReader(spy)
	orch.SetWorkflowStepGetter(&orchestratorWorkflowStepGetterAdapter{svc: f.h.workflowSvc})
	orch.SetTurnService(newTurnServiceAdapter(f.taskSvc))
	require.NoError(t, orch.Start(ctx))
	t.Cleanup(func() { _ = orch.Stop() })

	b := f.planTask("b.md")
	_, err = f.taskSvc.MoveTask(ctx, b.ID, planBoardID, autoStep, 0)
	require.NoError(t, err)

	sessions, err := f.taskSvc.ListTaskSessions(ctx, b.ID)
	require.NoError(t, err)
	assert.Empty(t, sessions)
	verdicts := spy.verdictsFor(b.ID)
	require.NotEmpty(t, verdicts, "the auto-start gate must be consulted")
	for _, blocked := range verdicts {
		assert.True(t, blocked)
	}
	moved, err := f.taskSvc.GetTask(ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, v1.TaskStateCreated, moved.State)
}
