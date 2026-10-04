package planfiles

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/planfiles/format"
	wfmodels "github.com/kandev/kandev/internal/workflow/models"
)

// @covers AC-TASKS-PLAN-BOARD-OPS-001.2
func TestSyncPass_ExecutorStepEntrySeenByThePassWritesTheExecutor(t *testing.T) {
	h := newSyncHarness(t)
	h.useDefaultExecutorSteps()
	h.writeFile("docs/plans/a.md", commentedPlan)
	h.sync()
	h.boardMove("docs/plans/a.md", codexStep)
	h.tasks.moves = nil

	summary := h.sync()

	assert.Contains(t, h.read("docs/plans/a.md"), "executor: Codex\n")
	assert.Contains(t, h.read("docs/plans/a.md"), "board: queued   # trailing")
	assert.Equal(t, 0, summary.Counts.Moved)
	assert.Empty(t, h.tasks.moves)
	assert.Equal(t, codexStep, h.taskFor("docs/plans/a.md").WorkflowStepID)
	assert.Equal(t, codexStep, h.row("docs/plans/a.md").SyncedStepID)
	assert.Equal(t, PassCounts{}, h.sync().Counts)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-001.3
func TestSyncPass_ClaimConflictSeenByThePassMovesBackWithNotice(t *testing.T) {
	h := newSyncHarness(t)
	h.useDefaultExecutorSteps()
	h.writeFile("docs/plans/a.md", planDoc("in_progress", "A", "executor: Claude"))
	h.sync()
	h.boardMove("docs/plans/a.md", codexStep)
	snap := h.snapshot("docs/plans/a.md")
	h.tasks.moves = nil

	summary := h.sync()

	h.assertUnchanged(snap)
	assert.Equal(t, 1, summary.Counts.Moved)
	require.Len(t, h.tasks.moves, 1, "the task is moved back once")
	assert.Equal(t, h.step(format.BoardInProgress), h.tasks.moves[0].StepID)
	assert.Equal(t, wfmodels.StepTransitionActorSystem, h.tasks.moves[0].Actor)
	assert.Contains(t, h.taskFor("docs/plans/a.md").Description, "plan is already taken by Claude")
	row := h.row("docs/plans/a.md")
	assert.Equal(t, "plan is already taken by Claude", row.Notice)
	assert.Equal(t, h.step(format.BoardInProgress), row.SyncedStepID)

	before := h.tasks.writeCount()
	assert.Equal(t, PassCounts{}, h.sync().Counts)
	assert.Equal(t, before, h.tasks.writeCount())
	h.writeBackTask("docs/plans/a.md")
	assert.Equal(t, before, h.tasks.writeCount())
	h.assertUnchanged(snap)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-001.4
func TestSyncPass_TaskLeavesAnExecutorStepWhenTheExecutorNoLongerMatches(t *testing.T) {
	cases := []struct {
		name   string
		edited string
		want   format.BoardStatus
	}{
		{"another executor", planDoc("in_progress", "A", "executor: Claude"), format.BoardInProgress},
		{"executor removed", planDoc("queued", "A"), format.BoardQueued},
		{"plan waits for the owner", planDoc("waiting_owner", "A", "executor: Codex"), format.BoardWaitingOwner},
		{"plan done", planDoc("done", "A", "executor: Codex"), format.BoardDone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newSyncHarness(t)
			h.useDefaultExecutorSteps()
			h.writeFile("docs/plans/a.md", planDoc("queued", "A", "executor: Codex"))
			h.sync()
			h.boardMove("docs/plans/a.md", codexStep)
			h.writeBackTask("docs/plans/a.md")
			require.Equal(t, 0, h.sync().Counts.Moved)
			h.writeFile("docs/plans/a.md", tc.edited)
			h.tasks.moves = nil

			summary := h.sync()

			assert.Equal(t, 1, summary.Counts.Moved)
			assert.Equal(t, h.step(tc.want), h.taskFor("docs/plans/a.md").WorkflowStepID)
			assert.Equal(t, tc.edited, h.read("docs/plans/a.md"))
			assert.Equal(t, PassCounts{}, h.sync().Counts)
		})
	}
}

// @covers AC-TASKS-PLAN-BOARD-OPS-001.4
func TestSyncPass_NeverMovesATaskIntoAnExecutorStep(t *testing.T) {
	h := newSyncHarness(t)
	h.useDefaultExecutorSteps()
	h.writeFile("docs/plans/a.md", planDoc("queued", "A", "executor: Codex"))
	h.writeFile("docs/plans/b.md", planDoc("in_progress", "B", "executor: Claude"))

	h.sync()
	h.sync()

	assert.Equal(t, h.step(format.BoardQueued), h.taskFor("docs/plans/a.md").WorkflowStepID)
	assert.Equal(t, h.step(format.BoardInProgress), h.taskFor("docs/plans/b.md").WorkflowStepID)
	for _, move := range h.tasks.moves {
		assert.NotContains(t, []string{codexStep, claudeStp, plainStep}, move.StepID)
	}
}

// @covers AC-TASKS-PLAN-BOARD-OPS-001.5
func TestSyncPass_UnnamedHandoffStepKeepsAQueuedOrInProgressTask(t *testing.T) {
	h := newSyncHarness(t)
	h.useDefaultExecutorSteps()
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	h.writeFile("docs/plans/b.md", planDoc("in_progress", "B", "executor: Someone"))
	h.sync()
	h.boardMove("docs/plans/a.md", plainStep)
	h.boardMove("docs/plans/b.md", plainStep)
	h.writeBackTask("docs/plans/a.md")
	h.writeBackTask("docs/plans/b.md")

	summary := h.sync()

	assert.Equal(t, 0, summary.Counts.Moved)
	assert.Equal(t, plainStep, h.taskFor("docs/plans/a.md").WorkflowStepID)
	assert.Equal(t, plainStep, h.taskFor("docs/plans/b.md").WorkflowStepID)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-001.2
// @covers AC-TASKS-PLAN-BOARD-OPS-001.3
// @covers AC-TASKS-PLAN-BOARD-OPS-001.5
func TestSyncPass_SettledBoardWithParkedTasksWritesAndMovesNothing(t *testing.T) {
	h := newSyncHarness(t)
	h.useDefaultExecutorSteps()
	h.writeFile("docs/plans/a.md", planDoc("queued", "A", "executor: Codex"))
	h.writeFile("docs/plans/b.md", planDoc("in_progress", "B", "executor: Claude"))
	h.writeFile("docs/plans/c.md", planDoc("queued", "C"))
	h.writeFile("docs/plans/d.md", planDoc("done", "D"))
	h.sync()
	h.boardMove("docs/plans/a.md", codexStep)
	h.boardMove("docs/plans/b.md", claudeStp)
	h.boardMove("docs/plans/c.md", plainStep)
	rels := []string{"docs/plans/a.md", "docs/plans/b.md", "docs/plans/c.md", "docs/plans/d.md"}
	for _, rel := range rels {
		h.writeBackTask(rel)
	}
	h.sync()
	snap := h.snapshot(rels...)
	before := h.tasks.writeCount()
	counted := totalWritebacks()
	h.tasks.moves = nil

	summary := h.sync()

	assert.Equal(t, PassCounts{}, summary.Counts)
	assert.Equal(t, before, h.tasks.writeCount(), "no task write")
	assert.Empty(t, h.tasks.moves, "no move")
	assert.Equal(t, counted, totalWritebacks(), "no file write")
	h.assertUnchanged(snap)
	assert.Equal(t, codexStep, h.taskFor("docs/plans/a.md").WorkflowStepID)
	assert.Equal(t, claudeStp, h.taskFor("docs/plans/b.md").WorkflowStepID)
	assert.Equal(t, plainStep, h.taskFor("docs/plans/c.md").WorkflowStepID)
}
