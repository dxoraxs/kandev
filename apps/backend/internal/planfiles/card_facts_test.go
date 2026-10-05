package planfiles

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/planfiles/format"
	taskmodels "github.com/kandev/kandev/internal/task/models"
)

func datedPlan(board format.BoardStatus, date, executor string) format.PlanFile {
	return format.PlanFile{Board: board, Title: "Plan", Priority: "medium", Date: date, Executor: executor}
}

// @covers AC-TASKS-PLAN-CARD-001.2
func TestProjectCardFacts_DateKindForEveryBoardStatus(t *testing.T) {
	cases := []struct {
		board format.BoardStatus
		want  map[string]any
	}{
		{format.BoardQueued, map[string]any{"date": "2026-10-12", "date_kind": "due"}},
		{format.BoardInProgress, map[string]any{"date": "2026-10-12", "date_kind": "due"}},
		{format.BoardWaitingOwner, map[string]any{"date": "2026-10-12", "date_kind": "waiting"}},
		{format.BoardWaitingExternal, map[string]any{"date": "2026-10-12", "date_kind": "waiting"}},
		{format.BoardDeferred, map[string]any{"date": "2026-10-12", "date_kind": "deferred"}},
		{format.BoardDone, nil},
		{format.BoardHidden, map[string]any{"date": "2026-10-12", "date_kind": "due"}},
		{"", map[string]any{"date": "2026-10-12", "date_kind": "due"}},
	}
	for _, c := range cases {
		t.Run(string(c.board), func(t *testing.T) {
			assert.Equal(t, c.want, projectCardFacts(datedPlan(c.board, "2026-10-12", "")))
		})
	}
}

// @covers AC-TASKS-PLAN-CARD-001.2
func TestProjectCardFacts_NoDateNoDateFact(t *testing.T) {
	assert.Nil(t, projectCardFacts(datedPlan(format.BoardWaitingOwner, "", "")))
}

// @covers AC-TASKS-PLAN-CARD-001.3
func TestProjectCardFacts_ExecutorFact(t *testing.T) {
	assert.Equal(t,
		map[string]any{"executor": map[string]any{"name": "Claude", "kind": "agent"}},
		projectCardFacts(datedPlan(format.BoardInProgress, "", "  Claude ")))
	assert.Equal(t,
		map[string]any{
			"date": "2026-10-12", "date_kind": "due",
			"executor": map[string]any{"name": "codex", "kind": "agent"},
		},
		projectCardFacts(datedPlan(format.BoardQueued, "2026-10-12", "codex")))
	assert.Nil(t, projectCardFacts(datedPlan(format.BoardDone, "2026-10-12", "codex")), "done plans carry no facts")
}

// @covers AC-TASKS-PLAN-CARD-001.3
func TestProjectTitle_NeverCarriesTheExecutor(t *testing.T) {
	assert.Equal(t, "Build it", projectTitle(filePlan(format.BoardQueued, "Build it", "codex")))
	assert.Equal(t, "Build it", projectTitle(filePlan(format.BoardDone, "Build it", "codex")))
	assert.Equal(t, "Build it", projectTitle(filePlan("", "Build it", "codex")))
}

func (h *syncHarness) cardOf(rel string) any {
	h.t.Helper()
	return h.taskFor(rel).Metadata["card_display"]
}

// @covers AC-TASKS-PLAN-CARD-001.2
// @covers AC-TASKS-PLAN-CARD-001.3
func TestSync_CreationWritesCardFactsWithoutTitlePrefix(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("waiting_external", "Alpha", "date: 2026-10-12", "executor: Claude"))
	h.writeFile("docs/plans/b.md", planDoc("queued", "Beta"))

	h.sync()

	assert.Equal(t, "Alpha", h.taskFor("docs/plans/a.md").Title)
	assert.Equal(t, map[string]any{
		"date": "2026-10-12", "date_kind": "waiting",
		"executor": map[string]any{"name": "Claude", "kind": "agent"},
	}, h.cardOf("docs/plans/a.md"))
	assert.Nil(t, h.taskFor("docs/plans/b.md").Metadata, "a plan without facts is created without metadata")
	for _, req := range h.tasks.creates {
		if req.ExternalID == defaultExternalID(testRepoID, "docs/plans/b.md") {
			assert.Nil(t, req.Metadata)
		}
	}
}

// @covers AC-TASKS-PLAN-CARD-001.1
func TestSync_InvalidDateIsReportedAndWritesNoDateFact(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "Alpha", "date: 2026-02-30", "executor: codex"))

	h.sync()

	task := h.taskFor("docs/plans/a.md")
	assert.Contains(t, task.Description, "> Parse error: date: invalid_date")
	assert.Equal(t, map[string]any{"executor": map[string]any{"name": "codex", "kind": "agent"}}, h.cardOf("docs/plans/a.md"))
}

// @covers AC-TASKS-PLAN-CARD-001.4
func TestSync_UnchangedFactsProduceNoUpdate(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("waiting_owner", "Alpha", "date: 2026-10-12", "executor: Claude"))
	h.sync()
	updates, writes := len(h.tasks.updates), h.tasks.writeCount()

	first := h.sync()
	second := h.sync()

	assert.Equal(t, PassCounts{}, first.Counts)
	assert.Equal(t, PassCounts{}, second.Counts)
	assert.Len(t, h.tasks.updates, updates, "no UpdateTask call over two passes")
	assert.Equal(t, writes, h.tasks.writeCount())
}

// @covers AC-TASKS-PLAN-CARD-001.4
func TestSync_ChangedFactsUpdateOnceAndKeepOtherMetadata(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("waiting_owner", "Alpha", "date: 2026-10-12"))
	h.sync()
	id := h.taskFor("docs/plans/a.md").ID
	h.tasks.tasks[id].Metadata["other"] = map[string]any{"keep": "me"}
	h.tasks.tasks[id].Metadata["flag"] = true

	h.writeFile("docs/plans/a.md", planDoc("waiting_owner", "Alpha", "date: 2026-10-20"))
	before := len(h.tasks.updates)
	h.sync()

	require.Len(t, h.tasks.updates, before+1)
	got := h.taskFor("docs/plans/a.md").Metadata
	assert.Equal(t, map[string]any{"date": "2026-10-20", "date_kind": "waiting"}, got["card_display"])
	assert.Equal(t, map[string]any{"keep": "me"}, got["other"])
	assert.Equal(t, true, got["flag"])
	h.sync()
	assert.Len(t, h.tasks.updates, before+1)
}

// @covers AC-TASKS-PLAN-CARD-001.4
func TestSync_NoFactsRemovesCardDisplayAndKeepsEveryOtherKey(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "Alpha", "date: 2026-10-12", "executor: codex"))
	h.sync()
	id := h.taskFor("docs/plans/a.md").ID
	h.tasks.tasks[id].Metadata["other"] = "kept"

	h.writeFile("docs/plans/a.md", planDoc("done", "Alpha", "date: 2026-10-12", "executor: codex"))
	h.sync()

	got := h.taskFor("docs/plans/a.md").Metadata
	assert.NotContains(t, got, "card_display")
	assert.Equal(t, "kept", got["other"])
	updates := len(h.tasks.updates)
	h.sync()
	assert.Len(t, h.tasks.updates, updates)
}

// @covers AC-TASKS-PLAN-CARD-001.4
func TestSync_ForeignCardDisplayKeysAreReplacedByTheProjection(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "Alpha", "date: 2026-10-12"))
	h.sync()
	id := h.taskFor("docs/plans/a.md").ID
	card := h.tasks.tasks[id].Metadata["card_display"].(map[string]any)
	card["stale"] = "x"

	h.sync()

	assert.Equal(t, map[string]any{"date": "2026-10-12", "date_kind": "due"}, h.cardOf("docs/plans/a.md"))
}

// @covers AC-TASKS-PLAN-CARD-001.2
func TestWaitingOwner_ReturnsWhatThePassWrote(t *testing.T) {
	h := newSyncHarness(t)
	h.svc.SetWorkspaceLister(fakeWorkspaceLister{workspaces: []*taskmodels.Workspace{{ID: testWorkspace, Name: "kandev"}}})
	h.writeFile("docs/plans/a.md", planDoc("waiting_owner", "Alpha", "date: 2026-10-12", "executor: Claude"))
	h.writeFile("docs/plans/b.md", planDoc("waiting_owner", "Beta"))
	h.sync()

	out, err := h.svc.WaitingOwner(context.Background())

	require.NoError(t, err)
	require.Len(t, out.Items, 2)
	assert.Equal(t, "Alpha", out.Items[0].Title)
	assert.Equal(t, "2026-10-12", out.Items[0].Date)
	assert.Equal(t, "Claude", out.Items[0].Executor)
	assert.Equal(t, "Beta", out.Items[1].Title)
	assert.Empty(t, out.Items[1].Date)
	assert.Empty(t, out.Items[1].Executor)
}
