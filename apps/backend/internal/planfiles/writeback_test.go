package planfiles

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/events"
	"github.com/kandev/kandev/internal/events/bus"
	"github.com/kandev/kandev/internal/planfiles/format"
)

const commentedPlan = "---\n# kept comment\nboard: queued   # trailing\ntitle: Alpha plan\nowner: me\n---\n\n# Alpha\n\nBody text.\n"

// @covers AC-TASKS-PLAN-FILES-003.1
// @covers AC-TASKS-PLAN-FILES-003.4
func TestWriteBack_MoveToMappedStepWritesOnlyBoard(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", commentedPlan)
	h.sync()
	h.boardMove("docs/plans/a.md", h.step(format.BoardDone))
	written := writebackCount(writebackWritten)

	h.writeBack().handleTask(context.Background(), h.taskFor("docs/plans/a.md").ID)

	assert.Equal(t, strings.Replace(commentedPlan, "board: queued   # trailing", "board: done   # trailing", 1),
		h.read("docs/plans/a.md"))
	assert.Equal(t, written+1, writebackCount(writebackWritten))
	row := h.row("docs/plans/a.md")
	assert.Equal(t, h.step(format.BoardDone), row.SyncedStepID)
	assert.Equal(t, format.ContentHash([]byte(h.read("docs/plans/a.md"))), row.ContentHash)
	before := h.tasks.writeCount()

	summary := h.sync()

	assert.Equal(t, PassCounts{}, summary.Counts)
	assert.Equal(t, before, h.tasks.writeCount(), "the next pass makes no task change")
	assert.Equal(t, h.step(format.BoardDone), h.taskFor("docs/plans/a.md").WorkflowStepID)
}

// @covers AC-TASKS-PLAN-FILES-003.6
func TestWriteBack_MoveToHandoffStepWritesNothingAndRecordsStep(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", commentedPlan)
	h.sync()
	h.boardMove("docs/plans/a.md", handoffStep)
	skipped := writebackCount(writebackHandoffSkipped)

	h.writeBack().handleTask(context.Background(), h.taskFor("docs/plans/a.md").ID)

	assert.Equal(t, commentedPlan, h.read("docs/plans/a.md"))
	assert.Equal(t, handoffStep, h.row("docs/plans/a.md").SyncedStepID)
	assert.Equal(t, skipped+1, writebackCount(writebackHandoffSkipped))
	h.sync()
	assert.Equal(t, handoffStep, h.taskFor("docs/plans/a.md").WorkflowStepID, "a queued file leaves the task in the handoff step")
}

// @covers AC-TASKS-PLAN-FILES-003.3
func TestWriteBack_PriorityChangeWritesPriority(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", commentedPlan)
	h.sync()
	h.boardPriority("docs/plans/a.md", "high")

	h.writeBack().handleTask(context.Background(), h.taskFor("docs/plans/a.md").ID)

	assert.Equal(t, strings.Replace(commentedPlan, "owner: me\n", "owner: me\npriority: high\n", 1),
		h.read("docs/plans/a.md"))
	assert.Equal(t, "high", h.row("docs/plans/a.md").SyncedPriority)
	before := h.tasks.writeCount()
	h.sync()
	assert.Equal(t, before, h.tasks.writeCount())
	assert.Equal(t, "high", h.taskFor("docs/plans/a.md").Priority)
}

// @covers AC-TASKS-PLAN-FILES-003.2
func TestWriteBack_ReorderBetweenOrderedNeighboursChangesOnlyTheMovedFile(t *testing.T) {
	h := newSyncHarness(t)
	for _, rel := range []string{"a", "b", "c", "d"} {
		order := map[string]string{"a": "10", "b": "20", "c": "30", "d": "40"}[rel]
		h.writeFile("docs/plans/"+rel+".md", planDoc("queued", strings.ToUpper(rel), "order: "+order))
	}
	h.sync()
	files := map[string]string{}
	mtimes := map[string]int64{}
	for _, rel := range []string{"a", "b", "c", "d"} {
		files[rel], mtimes[rel] = h.read("docs/plans/"+rel+".md"), h.mtime("docs/plans/"+rel+".md")
	}
	h.boardReorder("docs/plans/a.md", "docs/plans/c.md", "docs/plans/b.md", "docs/plans/d.md")

	h.writeBack().handleStep(context.Background(), testWorkspace, h.step(format.BoardQueued))

	changed := 0
	for _, rel := range []string{"a", "b", "c", "d"} {
		if h.read("docs/plans/"+rel+".md") != files[rel] {
			changed++
			assert.NotEqual(t, mtimes[rel], h.mtime("docs/plans/"+rel+".md"))
		}
	}
	assert.Equal(t, 1, changed, "only the moved file changes")
	h.tasks.reorders = nil
	before := h.tasks.writeCount()

	summary := h.sync()

	assert.Equal(t, PassCounts{}, summary.Counts)
	assert.Equal(t, before, h.tasks.writeCount(), "the pass reproduces the board order without a reorder")
	assert.Equal(t, []string{"docs/plans/a.md", "docs/plans/c.md", "docs/plans/b.md", "docs/plans/d.md"},
		h.boardOrder(h.step(format.BoardQueued)))
}

// @covers AC-TASKS-PLAN-FILES-003.2
func TestWriteBack_ReorderBelowUnorderedNeighboursNumbersFromTop(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/m.md", planDoc("queued", "M", "order: 5"))
	h.writeFile("docs/plans/u1.md", planDoc("queued", "U1"))
	h.writeFile("docs/plans/u2.md", planDoc("queued", "U2"))
	h.writeFile("docs/plans/u3.md", planDoc("queued", "U3"))
	h.writeFile("docs/plans/z.md", planDoc("done", "Z"))
	h.sync()
	u3, z := h.read("docs/plans/u3.md"), h.read("docs/plans/z.md")
	h.boardReorder("docs/plans/u1.md", "docs/plans/u2.md", "docs/plans/m.md", "docs/plans/u3.md")

	h.writeBack().handleStep(context.Background(), testWorkspace, h.step(format.BoardQueued))

	assert.Contains(t, h.read("docs/plans/u1.md"), "order: 10\n")
	assert.Contains(t, h.read("docs/plans/u2.md"), "order: 20\n")
	assert.Contains(t, h.read("docs/plans/m.md"), "order: 30\n")
	assert.Equal(t, u3, h.read("docs/plans/u3.md"), "tasks below the moved one are not written")
	assert.Equal(t, z, h.read("docs/plans/z.md"), "other steps are not written")
	h.tasks.reorders = nil
	h.sync()
	assert.Empty(t, h.tasks.reorders)
	assert.Equal(t, []string{"docs/plans/u1.md", "docs/plans/u2.md", "docs/plans/m.md", "docs/plans/u3.md"},
		h.boardOrder(h.step(format.BoardQueued)))
}

// @covers AC-TASKS-PLAN-FILES-003.2
func TestWriteBack_ReorderLeavesAnAlreadySortedStepAlone(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A", "order: 10"))
	h.writeFile("docs/plans/b.md", planDoc("queued", "B", "order: 20"))
	h.sync()
	before, written := h.read("docs/plans/a.md"), writebackCount(writebackWritten)

	h.writeBack().handleStep(context.Background(), testWorkspace, h.step(format.BoardQueued))

	assert.Equal(t, before, h.read("docs/plans/a.md"))
	assert.Equal(t, written, writebackCount(writebackWritten))
}

// @covers AC-TASKS-PLAN-FILES-003.5
func TestWriteBack_FileChangedAfterLastReadIsNotOverwritten(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", commentedPlan)
	h.sync()
	edited := strings.Replace(commentedPlan, "Body text.", "Body text, edited by a person.", 1)
	h.writeFile("docs/plans/a.md", edited)
	h.boardMove("docs/plans/a.md", h.step(format.BoardDone))
	conflicts := writebackCount(writebackConflict)

	h.writeBack().handleTask(context.Background(), h.taskFor("docs/plans/a.md").ID)

	assert.Equal(t, edited, h.read("docs/plans/a.md"), "the file is not overwritten")
	assert.Equal(t, conflicts+1, writebackCount(writebackConflict))
	assert.NotEmpty(t, h.row("docs/plans/a.md").Notice)

	summary := h.sync()

	task := h.taskFor("docs/plans/a.md")
	assert.Equal(t, h.step(format.BoardQueued), task.WorkflowStepID, "the task returns to the file's state")
	assert.Contains(t, task.Description, "Notice: board edit not saved because the file changed")
	assert.Equal(t, edited, h.read("docs/plans/a.md"))
	assert.Equal(t, 0, summary.Counts.Failed)

	h.writeFile("docs/plans/a.md", edited+"more\n")
	h.sync()
	assert.NotContains(t, h.taskFor("docs/plans/a.md").Description, "Notice:", "the notice clears when the file next changes")
}

// @covers AC-TASKS-PLAN-FILES-003.5
func TestWriteBack_UnsupportedValueShapeFailsWithoutTouchingTheFile(t *testing.T) {
	h := newSyncHarness(t)
	content := "---\nboard: |\n  queued\n---\n\n# A\n"
	h.writeFile("docs/plans/a.md", content)
	h.sync()
	require.Equal(t, h.step(format.BoardQueued), h.taskFor("docs/plans/a.md").WorkflowStepID)
	h.boardMove("docs/plans/a.md", h.step(format.BoardDone))
	failed := writebackCount(writebackFailed)

	h.writeBack().handleTask(context.Background(), h.taskFor("docs/plans/a.md").ID)

	assert.Equal(t, content, h.read("docs/plans/a.md"))
	assert.Equal(t, failed+1, writebackCount(writebackFailed))

	summary := h.sync()

	assert.Equal(t, h.step(format.BoardQueued), h.taskFor("docs/plans/a.md").WorkflowStepID, "the next pass restores the task")
	assert.Equal(t, content, h.read("docs/plans/a.md"))
	require.Len(t, summary.FileErrors, 1)
	assert.Equal(t, ReasonWriteFailed, summary.FileErrors[0].Reason)
}

func TestWriteBack_IgnoresTasksWithoutAPlanRow(t *testing.T) {
	h := newSyncHarness(t)
	h.tasks.seedTask("other", "wf-1", h.step(format.BoardQueued), "")
	written := writebackCount(writebackWritten)

	h.writeBack().handleTask(context.Background(), "other")
	h.writeBack().handleTask(context.Background(), "missing")

	assert.Equal(t, written, writebackCount(writebackWritten))
}

func newBusWriteBack(t *testing.T, h *syncHarness) (*WriteBackSubscriber, bus.EventBus) {
	t.Helper()
	log := h.svc.logger
	eventBus := bus.NewMemoryEventBus(log)
	w := NewWriteBackSubscriber(h.svc, eventBus, log)
	w.Start(context.Background())
	t.Cleanup(w.Stop)
	return w, eventBus
}

// @covers AC-TASKS-PLAN-FILES-003.1
func TestWriteBack_SubscriberWritesAfterATaskMovedEvent(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", commentedPlan)
	h.sync()
	_, eventBus := newBusWriteBack(t, h)
	h.boardMove("docs/plans/a.md", h.step(format.BoardDone))

	require.NoError(t, eventBus.Publish(context.Background(), events.TaskMoved,
		bus.NewEvent(events.TaskMoved, "test", map[string]interface{}{"task_id": h.taskFor("docs/plans/a.md").ID})))

	require.Eventually(t, func() bool {
		return strings.Contains(h.read("docs/plans/a.md"), "board: done")
	}, 5*time.Second, 5*time.Millisecond)
}

// @covers AC-TASKS-PLAN-FILES-003.2
func TestWriteBack_SubscriberHandlesAReorderedEvent(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A", "order: 10"))
	h.writeFile("docs/plans/b.md", planDoc("queued", "B", "order: 20"))
	h.sync()
	_, eventBus := newBusWriteBack(t, h)
	h.boardReorder("docs/plans/b.md", "docs/plans/a.md")

	require.NoError(t, eventBus.Publish(context.Background(), events.TaskReordered,
		bus.NewEvent(events.TaskReordered, "test", map[string]interface{}{
			"workspace_id": testWorkspace, "workflow_step_id": h.step(format.BoardQueued),
		})))

	require.Eventually(t, func() bool {
		return strings.Contains(h.read("docs/plans/b.md"), "order: 9\n")
	}, 5*time.Second, 5*time.Millisecond)
}

// The memory bus delivers events synchronously from the publisher, which a
// sync pass is while it moves tasks; the handler must therefore never wait for
// the workspace lock.
func TestWriteBack_HandlerDoesNotWaitForTheWorkspaceLock(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", commentedPlan)
	h.sync()
	_, eventBus := newBusWriteBack(t, h)
	unlock := h.svc.LockWorkspace(testWorkspace)
	released := false
	defer func() {
		if !released {
			unlock()
		}
	}()
	h.boardMove("docs/plans/a.md", h.step(format.BoardDone))
	done := make(chan struct{})

	go func() {
		defer close(done)
		_ = eventBus.Publish(context.Background(), events.TaskUpdated,
			bus.NewEvent(events.TaskUpdated, "test", map[string]interface{}{"task_id": h.taskFor("docs/plans/a.md").ID}))
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the event handler blocked on the workspace lock")
	}
	unlock()
	released = true
	require.Eventually(t, func() bool {
		return strings.Contains(h.read("docs/plans/a.md"), "board: done")
	}, 5*time.Second, 5*time.Millisecond)
}

func TestWriteBack_StartStopAreIdempotentAndRestartable(t *testing.T) {
	h := newSyncHarness(t)
	w := NewWriteBackSubscriber(h.svc, bus.NewMemoryEventBus(h.svc.logger), h.svc.logger)

	w.Start(context.Background())
	w.Start(context.Background())
	w.Stop()
	w.Stop()
	w.Start(context.Background())
	w.Stop()
}
