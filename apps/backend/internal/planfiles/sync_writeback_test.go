package planfiles

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/planfiles/format"
)

// snapshot records the bytes and modification time of plan files.
type fileSnapshot map[string]struct {
	content string
	mtime   int64
}

func (h *syncHarness) snapshot(rels ...string) fileSnapshot {
	snap := fileSnapshot{}
	for _, rel := range rels {
		snap[rel] = struct {
			content string
			mtime   int64
		}{h.read(rel), h.mtime(rel)}
	}
	return snap
}

func (h *syncHarness) assertUnchanged(snap fileSnapshot) {
	h.t.Helper()
	for rel, before := range snap {
		assert.Equal(h.t, before.content, h.read(rel), rel)
		assert.Equal(h.t, before.mtime, h.mtime(rel), "%s was rewritten", rel)
	}
}

func totalWritebacks() int64 {
	return writebackCount(writebackWritten) + writebackCount(writebackConflict) +
		writebackCount(writebackFailed) + writebackCount(writebackHandoffSkipped)
}

// @covers AC-TASKS-PLAN-FILES-003.5
func TestSyncPass_BoardMoveSeenBeforeTheSubscriberIsWrittenNotReverted(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", commentedPlan)
	h.sync()
	h.boardMove("docs/plans/a.md", h.step(format.BoardDone))

	summary := h.sync()

	assert.Equal(t, h.step(format.BoardDone), h.taskFor("docs/plans/a.md").WorkflowStepID, "the move is kept")
	assert.Equal(t, strings.Replace(commentedPlan, "board: queued   # trailing", "board: done   # trailing", 1),
		h.read("docs/plans/a.md"))
	assert.Equal(t, 0, summary.Counts.Moved)
	assert.Equal(t, 0, summary.Counts.Failed)
	assert.Equal(t, h.step(format.BoardDone), h.row("docs/plans/a.md").SyncedStepID)
	before := h.tasks.writeCount()
	assert.Equal(t, PassCounts{}, h.sync().Counts)
	assert.Equal(t, before, h.tasks.writeCount())
}

// @covers AC-TASKS-PLAN-FILES-003.5
func TestSyncPass_PriorityChangeSeenBeforeTheSubscriberIsWritten(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", commentedPlan)
	h.sync()
	h.boardPriority("docs/plans/a.md", "critical")

	h.sync()

	assert.Equal(t, "critical", h.taskFor("docs/plans/a.md").Priority)
	assert.Contains(t, h.read("docs/plans/a.md"), "priority: critical\n")
}

// @covers AC-TASKS-PLAN-FILES-003.5
// @covers AC-TASKS-PLAN-FILES-003.2
func TestSyncPass_ReorderSeenBeforeTheSubscriberIsWrittenNotReverted(t *testing.T) {
	h := newSyncHarness(t)
	for rel, order := range map[string]string{"a": "10", "b": "20", "c": "30"} {
		h.writeFile("docs/plans/"+rel+".md", planDoc("queued", strings.ToUpper(rel), "order: "+order))
	}
	h.sync()
	h.tasks.reorders = nil
	h.boardReorder("docs/plans/c.md", "docs/plans/a.md", "docs/plans/b.md")
	untouched := h.snapshot("docs/plans/a.md", "docs/plans/b.md")

	summary := h.sync()

	assert.Empty(t, h.tasks.reorders, "the board is not reordered back")
	assert.Equal(t, []string{"docs/plans/c.md", "docs/plans/a.md", "docs/plans/b.md"}, h.boardOrder(h.step(format.BoardQueued)))
	assert.Contains(t, h.read("docs/plans/c.md"), "order: 9\n")
	h.assertUnchanged(untouched)
	assert.Equal(t, 0, summary.Counts.Failed)
	before := h.tasks.writeCount()
	assert.Equal(t, PassCounts{}, h.sync().Counts)
	assert.Equal(t, before, h.tasks.writeCount())
}

// @covers AC-TASKS-PLAN-FILES-003.5
func TestSyncPass_BoardEditToAChangedFileLosesAndShowsTheNotice(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", commentedPlan)
	h.sync()
	edited := strings.Replace(commentedPlan, "Body text.", "Body text, edited.", 1)
	h.writeFile("docs/plans/a.md", edited)
	h.boardMove("docs/plans/a.md", h.step(format.BoardDone))
	h.boardPriority("docs/plans/a.md", "critical")
	before := totalWritebacks()

	h.sync()

	task := h.taskFor("docs/plans/a.md")
	assert.Equal(t, h.step(format.BoardQueued), task.WorkflowStepID)
	assert.Equal(t, "medium", task.Priority)
	assert.Contains(t, task.Description, "Notice: board edit not saved because the file changed")
	assert.Equal(t, edited, h.read("docs/plans/a.md"))
	assert.Equal(t, before, totalWritebacks(), "no write was attempted")
	h.sync()
	assert.Contains(t, h.taskFor("docs/plans/a.md").Description, "Notice:", "the notice stays while the file is unchanged")
}

// @covers AC-TASKS-PLAN-FILES-003.2
func TestSyncPass_MoveIntoAStepIsReorderedByTheFileNotWrittenBack(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A", "order: 10"))
	h.writeFile("docs/plans/b.md", planDoc("queued", "B", "order: 20"))
	h.writeFile("docs/plans/c.md", planDoc("done", "C", "order: 5"))
	h.sync()
	h.boardMove("docs/plans/c.md", h.step(format.BoardQueued))
	untouched := h.snapshot("docs/plans/a.md", "docs/plans/b.md")

	h.sync()

	assert.Contains(t, h.read("docs/plans/c.md"), "board: queued")
	h.assertUnchanged(untouched)
	assert.Equal(t, []string{"docs/plans/c.md", "docs/plans/a.md", "docs/plans/b.md"}, h.boardOrder(h.step(format.BoardQueued)))
	assert.NotEmpty(t, h.tasks.reorders, "the pass reorders the board to the file orders")
}

// @covers AC-TASKS-PLAN-FILES-003.1
func TestSyncPass_OwnMovesAndReordersTriggerNoFileWrites(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A", "order: 20"))
	h.writeFile("docs/plans/b.md", planDoc("queued", "B", "order: 10"))
	h.writeFile("docs/plans/c.md", planDoc("in_progress", "C", "priority: high"))
	h.writeFile("docs/plans/d.md", planDoc("queued", "D"))
	h.sync()
	h.writeFile("docs/plans/b.md", planDoc("done", "B", "order: 10"))
	h.writeFile("docs/plans/a.md", planDoc("queued", "A", "order: 5"))
	h.writeFile("docs/plans/c.md", planDoc("queued", "C", "priority: low", "order: 7"))
	rels := []string{"docs/plans/a.md", "docs/plans/b.md", "docs/plans/c.md", "docs/plans/d.md"}
	snap := h.snapshot(rels...)
	counted := totalWritebacks()
	h.tasks.moves, h.tasks.reorders = nil, nil

	summary := h.sync()

	assert.Equal(t, 2, summary.Counts.Moved)
	assert.NotEmpty(t, h.tasks.reorders, "the pass reorders the board")
	h.assertUnchanged(snap)
	assert.Equal(t, counted, totalWritebacks(), "the pass writes no file")

	wb := h.writeBack()
	ctx := context.Background()
	for _, rel := range rels {
		wb.handleTask(ctx, h.taskFor(rel).ID)
	}
	wb.handleStep(ctx, testWorkspace, h.step(format.BoardQueued))
	wb.handleStep(ctx, testWorkspace, h.step(format.BoardDone))
	h.assertUnchanged(snap)
	assert.Equal(t, counted, totalWritebacks(), "the events of the pass's own changes write nothing")
	require.Equal(t, PassCounts{}, h.sync().Counts)
}
