package backendapp

import (
	"context"
	"expvar"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/common/logger"
	"github.com/kandev/kandev/internal/planfiles"
	"github.com/kandev/kandev/internal/planfiles/format"
	taskservice "github.com/kandev/kandev/internal/task/service"
)

func (f *planSyncFixture) startWriteBack() {
	f.t.Helper()
	log, err := logger.NewLogger(logger.LoggingConfig{Level: "error", Format: "json", OutputPath: "stdout"})
	require.NoError(f.t, err)
	w := planfiles.NewWriteBackSubscriber(f.svc, f.h.eventBus, log)
	w.Start(context.Background())
	f.t.Cleanup(w.Stop)
}

func (f *planSyncFixture) readPlan(name string) string {
	f.t.Helper()
	data, err := os.ReadFile(filepath.Join(f.repoRoot, "docs", "plans", name))
	require.NoError(f.t, err)
	return string(data)
}

func (f *planSyncFixture) planMtime(name string) int64 {
	f.t.Helper()
	info, err := os.Stat(filepath.Join(f.repoRoot, "docs", "plans", name))
	require.NoError(f.t, err)
	return info.ModTime().UnixNano()
}

func (f *planSyncFixture) moveTask(name, stepID string) {
	f.t.Helper()
	_, err := f.taskSvc.MoveTaskWithOptions(context.Background(), f.planTask(name).ID, planBoardID, stepID, 0,
		taskservice.MoveTaskOptions{})
	require.NoError(f.t, err)
}

func writebackWritten() int64 {
	counters, ok := expvar.Get("plan_files_writeback_total").(*expvar.Map)
	if !ok {
		return 0
	}
	if v, ok := counters.Get("outcome=written").(*expvar.Int); ok {
		return v.Value()
	}
	return 0
}

// @covers AC-TASKS-PLAN-FILES-003.1
// @covers AC-TASKS-PLAN-FILES-003.2
// @covers AC-TASKS-PLAN-FILES-003.3
func TestPlanFilesWriteBack_RealTaskServiceMoveReorderAndPriority(t *testing.T) {
	f := newPlanSyncFixture(t)
	ctx := context.Background()
	f.writePlan("a.md", planFile("queued", "Alpha", "order: 10"))
	f.writePlan("b.md", planFile("queued", "Beta", "order: 20"))
	f.writePlan("c.md", planFile("queued", "Gamma", "order: 30"))
	f.sync()
	f.startWriteBack()
	queued := f.steps[format.BoardQueued]

	f.moveTask("a.md", f.steps[format.BoardDone])
	require.Eventually(t, func() bool { return strings.Contains(f.readPlan("a.md"), "board: done") },
		5*time.Second, 10*time.Millisecond, "a board move is written to the file")

	_, err := f.taskSvc.UpdateTask(ctx, f.planTask("b.md").ID, &taskservice.UpdateTaskRequest{Priority: strPtr("high")})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return strings.Contains(f.readPlan("b.md"), "priority: high") },
		5*time.Second, 10*time.Millisecond, "a priority change is written to the file")

	_, err = f.taskSvc.ReorderStepTasks(ctx, queued, "admitted", []string{f.planTask("c.md").ID, f.planTask("b.md").ID})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return strings.Contains(f.readPlan("c.md"), "order: 19") },
		5*time.Second, 10*time.Millisecond, "a reorder is written to the moved file only")
	assert.Equal(t, 0, f.sync().Counts.Failed)
	assert.Equal(t, []string{"Gamma", "Beta"}, f.stepOrder(queued))
	assert.Equal(t, planfiles.PassCounts{}, f.sync().Counts)
}

func strPtr(s string) *string { return &s }

// @covers AC-TASKS-PLAN-FILES-003.1
func TestPlanFilesWriteBack_PassEventsWriteNoFiles(t *testing.T) {
	f := newPlanSyncFixture(t)
	f.startWriteBack()
	f.writePlan("a.md", planFile("queued", "Alpha", "order: 20"))
	f.writePlan("b.md", planFile("queued", "Beta", "order: 10"))
	f.writePlan("c.md", planFile("in_progress", "Gamma", "priority: high"))
	f.sync()
	f.writePlan("b.md", planFile("done", "Beta", "order: 10"))
	f.writePlan("a.md", planFile("queued", "Alpha", "order: 5"))
	f.writePlan("c.md", planFile("queued", "Gamma", "priority: low", "order: 7"))
	names := []string{"a.md", "b.md", "c.md"}
	contents, mtimes := map[string]string{}, map[string]int64{}
	for _, name := range names {
		contents[name], mtimes[name] = f.readPlan(name), f.planMtime(name)
	}
	written := writebackWritten()

	summary := f.sync()

	assert.Equal(t, 2, summary.Counts.Moved)
	// A deliberate board move afterwards proves the subscriber has drained the
	// events of the pass: its queue is first in, first out.
	f.moveTask("a.md", f.steps[format.BoardWaitingOwner])
	require.Eventually(t, func() bool { return strings.Contains(f.readPlan("a.md"), "board: waiting_owner") },
		5*time.Second, 10*time.Millisecond)
	for _, name := range []string{"b.md", "c.md"} {
		assert.Equal(t, contents[name], f.readPlan(name), name)
		assert.Equal(t, mtimes[name], f.planMtime(name), "%s was rewritten", name)
	}
	assert.Equal(t, written+1, writebackWritten(), "only the deliberate move was written")
}
