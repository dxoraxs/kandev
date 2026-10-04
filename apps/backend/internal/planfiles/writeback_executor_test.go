package planfiles

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/planfiles/format"
	wfmodels "github.com/kandev/kandev/internal/workflow/models"
)

const (
	codexStep = "extra"
	claudeStp = "extra2"
	plainStep = "extra3"
)

// useExecutorSteps gives the plan board an executor step per name and one
// handoff step without a name.
func (h *syncHarness) useExecutorSteps(names map[string]string) {
	h.t.Helper()
	h.wf.addWorkflow("wf-1", testWorkspace, "q", "ip", "wo", "we", "df", "dn", codexStep, claudeStp, plainStep)
	req := validRequest()
	req.ExecutorSteps = ptr(names)
	_, err := h.svc.PutConfig(context.Background(), testWorkspace, req)
	require.NoError(h.t, err)
}

func (h *syncHarness) useDefaultExecutorSteps() {
	h.t.Helper()
	h.useExecutorSteps(map[string]string{codexStep: "Codex", claudeStp: "Claude"})
}

func (h *syncHarness) writeBackTask(rel string) {
	h.t.Helper()
	h.writeBack().handleTask(context.Background(), h.taskFor(rel).ID)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-001.2
func TestWriteBack_QueuedPlanMovedToExecutorStepWritesOnlyExecutor(t *testing.T) {
	h := newSyncHarness(t)
	h.useDefaultExecutorSteps()
	h.writeFile("docs/plans/a.md", commentedPlan)
	h.sync()
	h.boardMove("docs/plans/a.md", codexStep)

	h.writeBackTask("docs/plans/a.md")

	assert.Equal(t, strings.Replace(commentedPlan, "owner: me\n", "owner: me\nexecutor: Codex\n", 1), h.read("docs/plans/a.md"))
	row := h.row("docs/plans/a.md")
	assert.Equal(t, codexStep, row.SyncedStepID)
	assert.Equal(t, format.ContentHash([]byte(h.read("docs/plans/a.md"))), row.ContentHash)
	h.tasks.moves = nil
	snap := h.snapshot("docs/plans/a.md")

	// The first pass only projects the new executor into the task title.
	summary := h.sync()

	assert.Equal(t, 0, summary.Counts.Moved)
	assert.Empty(t, h.tasks.moves)
	assert.Equal(t, "[Codex] Alpha plan", h.taskFor("docs/plans/a.md").Title)
	assert.Equal(t, codexStep, h.taskFor("docs/plans/a.md").WorkflowStepID)
	h.assertUnchanged(snap)
	before := h.tasks.writeCount()
	assert.Equal(t, PassCounts{}, h.sync().Counts)
	assert.Equal(t, before, h.tasks.writeCount())
	h.assertUnchanged(snap)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-001.2
func TestWriteBack_ExecutorAlreadyNamedWritesNothing(t *testing.T) {
	h := newSyncHarness(t)
	h.useDefaultExecutorSteps()
	h.writeFile("docs/plans/a.md", planDoc("queued", "A", "executor:   codex  "))
	h.sync()
	h.boardMove("docs/plans/a.md", codexStep)
	snap := h.snapshot("docs/plans/a.md")
	written := writebackCount(writebackWritten)

	h.writeBackTask("docs/plans/a.md")

	h.assertUnchanged(snap)
	assert.Equal(t, written, writebackCount(writebackWritten))
	assert.Equal(t, codexStep, h.row("docs/plans/a.md").SyncedStepID)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-001.2
func TestWriteBack_InProgressPlanWithoutExecutorGetsTheName(t *testing.T) {
	h := newSyncHarness(t)
	h.useDefaultExecutorSteps()
	h.writeFile("docs/plans/a.md", planDoc("in_progress", "A"))
	h.sync()
	h.boardMove("docs/plans/a.md", claudeStp)

	h.writeBackTask("docs/plans/a.md")

	assert.Contains(t, h.read("docs/plans/a.md"), "board: in_progress\nexecutor: Claude\n")
	assert.Equal(t, claudeStp, h.taskFor("docs/plans/a.md").WorkflowStepID)
	assert.Equal(t, 0, h.sync().Counts.Moved)
	assert.Equal(t, claudeStp, h.taskFor("docs/plans/a.md").WorkflowStepID)
	assert.Equal(t, PassCounts{}, h.sync().Counts)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-001.3
func TestWriteBack_ClaimConflictWritesNothingAndMovesBackWithNotice(t *testing.T) {
	h := newSyncHarness(t)
	h.useDefaultExecutorSteps()
	h.writeFile("docs/plans/a.md", planDoc("in_progress", "A", "executor: Claude"))
	h.sync()
	h.boardMove("docs/plans/a.md", codexStep)
	snap := h.snapshot("docs/plans/a.md")
	h.tasks.moves = nil

	h.writeBackTask("docs/plans/a.md")

	h.assertUnchanged(snap)
	require.Len(t, h.tasks.moves, 1)
	assert.Equal(t, h.step(format.BoardInProgress), h.tasks.moves[0].StepID)
	assert.Equal(t, wfmodels.StepTransitionActorSystem, h.tasks.moves[0].Actor)
	row := h.row("docs/plans/a.md")
	assert.Equal(t, h.step(format.BoardInProgress), row.SyncedStepID)
	assert.Equal(t, "plan is already taken by Claude", row.Notice)

	// The event of the move back is not a board edit.
	h.writeBackTask("docs/plans/a.md")
	assert.Len(t, h.tasks.moves, 1)

	// The next pass shows the notice and then settles.
	summary := h.sync()
	assert.Equal(t, 0, summary.Counts.Moved)
	assert.Contains(t, h.taskFor("docs/plans/a.md").Description, "plan is already taken by Claude")
	before := h.tasks.writeCount()
	assert.Equal(t, PassCounts{}, h.sync().Counts)
	assert.Equal(t, before, h.tasks.writeCount())
	assert.Equal(t, "plan is already taken by Claude", h.row("docs/plans/a.md").Notice)
	h.assertUnchanged(snap)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-001.3
func TestWriteBack_ClaimNoticeClearsWhenTheFileChanges(t *testing.T) {
	h := newSyncHarness(t)
	h.useDefaultExecutorSteps()
	h.writeFile("docs/plans/a.md", planDoc("in_progress", "A", "executor: Claude"))
	h.sync()
	h.boardMove("docs/plans/a.md", codexStep)
	h.writeBackTask("docs/plans/a.md")
	h.sync()

	h.writeFile("docs/plans/a.md", planDoc("in_progress", "A", "executor: Claude", "priority: high"))
	h.sync()

	assert.NotContains(t, h.taskFor("docs/plans/a.md").Description, "plan is already taken")
}

func TestWriteBack_ExecutorStepEnteredWhileDoneWritesNothing(t *testing.T) {
	h := newSyncHarness(t)
	h.useDefaultExecutorSteps()
	h.writeFile("docs/plans/a.md", planDoc("done", "A"))
	h.sync()
	h.boardMove("docs/plans/a.md", codexStep)
	snap := h.snapshot("docs/plans/a.md")

	h.writeBackTask("docs/plans/a.md")

	h.assertUnchanged(snap)
	assert.Equal(t, codexStep, h.row("docs/plans/a.md").SyncedStepID)
	h.sync()
	assert.Equal(t, h.step(format.BoardDone), h.taskFor("docs/plans/a.md").WorkflowStepID, "a done plan leaves the executor step")
}

// @covers AC-TASKS-PLAN-BOARD-OPS-001.2
func TestWriteBack_ExecutorNameThatIsNotAPlainScalarReadsBackExactly(t *testing.T) {
	h := newSyncHarness(t)
	h.useExecutorSteps(map[string]string{codexStep: "Team: A #1"})
	h.writeFile("docs/plans/a.md", commentedPlan)
	h.sync()
	h.boardMove("docs/plans/a.md", codexStep)

	h.writeBackTask("docs/plans/a.md")

	pf, ok := format.Parse("a.md", []byte(h.read("docs/plans/a.md")))
	require.True(t, ok)
	assert.Equal(t, "Team: A #1", pf.Executor)
	assert.Empty(t, pf.ParseErrors)
	assert.Equal(t, 0, h.sync().Counts.Moved, "the pass keeps the task in the step")
	assert.Equal(t, codexStep, h.taskFor("docs/plans/a.md").WorkflowStepID)
	assert.Equal(t, PassCounts{}, h.sync().Counts)
}

func TestExecutorScalar_RoundTripsThroughSetKeysAndParse(t *testing.T) {
	names := []string{
		"Codex", "Claude Code", "gpt-5", "a.b", "Team: A", "a: b", "a #b", "a#b", "- x", "-x", "? x", "[x]", "{x}",
		"&x", "*x", "!x", "|x", ">x", "'x'", `"x"`, "%x", "@x", "`x`", "123", "1.5", "0x1F", "true", "null", "~",
		"yes", "2026-10-05", "x\ty", "x\\y", "ünï 🚀", "a,b", "a'b", `a"b`, "end:", "# c",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			doc := []byte(planDoc("queued", "T", "executor: old # note"))
			out, err := format.SetKeys(doc, map[string]string{keyExecutor: executorScalar(name)})
			require.NoError(t, err)
			pf, ok := format.Parse("t.md", out)
			require.True(t, ok)
			assert.Empty(t, pf.ParseErrors)
			assert.Equal(t, name, pf.Executor)
			assert.Equal(t, "queued", string(pf.Board))
		})
	}
}

func TestExecutorScalar_LeavesPlainNamesUnquoted(t *testing.T) {
	assert.Equal(t, "Codex", executorScalar("Codex"))
	assert.Equal(t, "Claude Code", executorScalar("Claude Code"))
	assert.Equal(t, `"Team: A"`, executorScalar("Team: A"))
	assert.Equal(t, `"123"`, executorScalar("123"))
}

func TestPutConfig_WorkflowChangeResetsExecutorStepsThatWereNotSent(t *testing.T) {
	svc, fake := newTestService(t)
	ctx := context.Background()
	fake.addWorkflow("wf-2", testWorkspace, "q2", "ip2", "wo2", "we2", "df2", "dn2", "x2")
	req := validRequest()
	req.ExecutorSteps = ptr(map[string]string{"extra": "Claude"})
	_, err := svc.PutConfig(ctx, testWorkspace, req)
	require.NoError(t, err)
	move := validRequest()
	move.WorkflowID = "wf-2"
	move.StatusSteps = map[format.BoardStatus]string{
		format.BoardQueued: "q2", format.BoardInProgress: "ip2", format.BoardWaitingOwner: "wo2",
		format.BoardWaitingExternal: "we2", format.BoardDeferred: "df2", format.BoardDone: "dn2",
	}

	saved, err := svc.PutConfig(ctx, testWorkspace, move)

	require.NoError(t, err)
	assert.Equal(t, "wf-2", saved.WorkflowID)
	assert.Empty(t, saved.ExecutorSteps)

	move.ExecutorSteps = ptr(map[string]string{"x2": "Codex"})
	saved, err = svc.PutConfig(ctx, testWorkspace, move)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"x2": "Codex"}, saved.ExecutorSteps)
}
