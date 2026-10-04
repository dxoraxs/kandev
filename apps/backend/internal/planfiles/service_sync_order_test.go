package planfiles

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/planfiles/format"
)

// @covers AC-TASKS-PLAN-FILES-002.5
func TestSync_OrdersStepByOrderThenPathAmongPlanTasks(t *testing.T) {
	h := newSyncHarness(t)
	h.tasks.seedTask("other", "wf-1", h.step(format.BoardQueued), "")
	h.writeFile("docs/plans/a.md", planDoc("queued", "A", "order: 20"))
	h.writeFile("docs/plans/b.md", planDoc("queued", "B", "order: 10"))
	h.writeFile("docs/plans/c.md", planDoc("queued", "C"))
	h.writeFile("docs/plans/d.md", planDoc("queued", "D"))

	h.sync()

	require.Len(t, h.tasks.reorders, 1)
	ids := func(rels ...string) []string {
		out := []string{"other"}
		for _, rel := range rels {
			out = append(out, h.taskFor(rel).ID)
		}
		return out
	}
	assert.Equal(t, ids("docs/plans/b.md", "docs/plans/a.md", "docs/plans/c.md", "docs/plans/d.md"), h.tasks.reorders[0])
	before := h.tasks.writeCount()
	h.sync()
	assert.Equal(t, before, h.tasks.writeCount(), "an already ordered step is not reordered again")
}

// @covers AC-TASKS-PLAN-FILES-002.5
func TestSync_ReorderKeepsNonPlanTasksInTheirSlots(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A", "order: 20"))
	h.writeFile("docs/plans/b.md", planDoc("queued", "B", "order: 10"))
	h.sync()
	a, b := h.taskFor("docs/plans/a.md"), h.taskFor("docs/plans/b.md")
	h.tasks.seedTask("x", "wf-1", h.step(format.BoardQueued), "")
	h.tasks.tasks[b.ID].Position, h.tasks.tasks["x"].Position, h.tasks.tasks[a.ID].Position = 0, 1, 2
	h.tasks.reorders = nil

	h.writeFile("docs/plans/b.md", planDoc("queued", "B", "order: 30"))
	h.sync()

	require.Len(t, h.tasks.reorders, 1)
	assert.Equal(t, []string{a.ID, "x", b.ID}, h.tasks.reorders[0], "the non-plan task keeps the middle slot")
}

// @covers AC-TASKS-PLAN-FILES-002.5
func TestSync_OrderingIsPerStep(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A", "order: 2"))
	h.writeFile("docs/plans/b.md", planDoc("done", "B", "order: 1"))
	h.writeFile("docs/plans/c.md", planDoc("queued", "C", "order: 1"))

	h.sync()

	require.Len(t, h.tasks.reorders, 1, "only the queued step has two plan tasks out of order")
	assert.Equal(t, []string{h.taskFor("docs/plans/c.md").ID, h.taskFor("docs/plans/a.md").ID}, h.tasks.reorders[0])
}

// @covers AC-TASKS-PLAN-FILES-002.3
func TestSync_PriorityFollowsTheFile(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A", "priority: low"))
	h.sync()

	h.writeFile("docs/plans/a.md", planDoc("queued", "A", "priority: critical"))
	h.sync()

	assert.Equal(t, "critical", h.taskFor("docs/plans/a.md").Priority)
}

// @covers AC-TASKS-PLAN-FILES-002.3
func TestSync_LongTitleIsTruncatedWithEllipsis(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", strings.Repeat("W", 100)))

	h.sync()

	title := h.taskFor("docs/plans/a.md").Title
	assert.Equal(t, 60, len([]rune(title)))
	assert.True(t, strings.HasSuffix(title, "…"))
}
