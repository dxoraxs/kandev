package planfiles

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/planfiles/format"
	taskmodels "github.com/kandev/kandev/internal/task/models"
)

// @covers AC-TASKS-PLAN-FILES-002.1
// @covers AC-TASKS-PLAN-FILES-002.3
// @covers AC-TASKS-PLAN-FILES-004.1
func TestSync_CreatesOneTaskPerPlanFileInMappedStep(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "Alpha", "executor: codex", "priority: high"))
	h.writeFile("docs/plans/b.md", planDoc("done", "Beta", "executor: codex"))
	h.writeFile("docs/plans/notes.md", "# Just notes\n")
	h.writeFile("docs/plans/c.md", "---\ntitle: no board key\n---\nbody\n")

	summary := h.sync()

	assert.Equal(t, OutcomeOK, summary.Outcome)
	assert.Equal(t, 2, summary.Counts.Created)
	a := h.taskFor("docs/plans/a.md")
	assert.Equal(t, "Alpha", a.Title)
	assert.Equal(t, "high", a.Priority)
	assert.Equal(t, h.step(format.BoardQueued), a.WorkflowStepID)
	assert.Equal(t, "wf-1", a.WorkflowID)
	assert.Contains(t, a.Description, "> Plan file: `city_companion` `docs/plans/a.md`")
	assert.Contains(t, a.Description, "Body of Alpha.")
	b := h.taskFor("docs/plans/b.md")
	assert.Equal(t, "Beta", b.Title)
	assert.Equal(t, h.step(format.BoardDone), b.WorkflowStepID)
	assert.Len(t, h.tasks.creates, 2, "non-plan files create nothing")
	for _, req := range h.tasks.creates {
		assert.False(t, req.StartAgent)
		require.Len(t, req.Repositories, 1)
		assert.Equal(t, testRepoID, req.Repositories[0].RepositoryID)
	}
}

// @covers AC-TASKS-PLAN-FILES-004.1
func TestSync_FrontmatterExternalIDOverridesDefault(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "Alpha", "external_id: legacy-42"))

	h.sync()

	assert.NotNil(t, h.tasks.byExternalID("legacy-42"))
	assert.Nil(t, h.tasks.byExternalID(defaultExternalID(testRepoID, "docs/plans/a.md")))
}

// @covers AC-TASKS-PLAN-FILES-002.7
func TestSync_UnchangedSecondPassWritesNothing(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "Alpha", "executor: codex"))
	h.writeFile("docs/plans/b.md", planDoc("in_progress", "Beta", "order: 1"))
	h.sync()
	require.NotZero(t, h.tasks.writeCount(), "positive control: the first pass writes")
	before := h.tasks.writeCount()

	summary := h.sync()

	assert.Equal(t, before, h.tasks.writeCount(), "writes after second pass: %v", h.tasks.writes[before:])
	assert.Equal(t, PassCounts{}, summary.Counts)
	assert.Equal(t, OutcomeOK, summary.Outcome)
}

// @covers AC-TASKS-PLAN-FILES-002.2
func TestSync_FileChangeUpdatesAndMovesTheSameTask(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "Alpha"))
	h.sync()
	id := h.taskFor("docs/plans/a.md").ID

	h.writeFile("docs/plans/a.md", planDoc("waiting_owner", "Alpha renamed", "priority: critical"))
	summary := h.sync()

	got := h.tasks.task(id)
	assert.Equal(t, "Alpha renamed", got.Title)
	assert.Equal(t, "critical", got.Priority)
	assert.Equal(t, h.step(format.BoardWaitingOwner), got.WorkflowStepID)
	assert.Contains(t, got.Description, "Body of Alpha renamed.")
	assert.Equal(t, PassCounts{Updated: 1, Moved: 1}, summary.Counts)
	require.Len(t, h.tasks.moves, 1)
	assert.Equal(t, "system", string(h.tasks.moves[0].Actor))
	for _, upd := range h.tasks.updates {
		assert.Nil(t, upd.Metadata, "a plan without card facts never sends metadata")
	}
}

// @covers AC-TASKS-PLAN-FILES-002.6
func TestSync_DeletedOrHiddenFileArchivesAndReturnUnarchivesSameTask(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "Alpha"))
	h.writeFile("docs/plans/b.md", planDoc("queued", "Beta"))
	h.sync()
	aID, bID := h.taskFor("docs/plans/a.md").ID, h.taskFor("docs/plans/b.md").ID

	h.removeFile("docs/plans/a.md")
	h.writeFile("docs/plans/b.md", planDoc("hidden", "Beta"))
	summary := h.sync()

	assert.Equal(t, 2, summary.Counts.Archived)
	assert.NotNil(t, h.tasks.task(aID).ArchivedAt)
	assert.NotNil(t, h.tasks.task(bID).ArchivedAt)

	h.writeFile("docs/plans/a.md", planDoc("queued", "Alpha"))
	h.writeFile("docs/plans/b.md", planDoc("done", "Beta"))
	summary = h.sync()

	assert.Equal(t, 2, summary.Counts.Unarchived)
	assert.Nil(t, h.tasks.task(aID).ArchivedAt)
	assert.Nil(t, h.tasks.task(bID).ArchivedAt)
	assert.Equal(t, aID, h.taskFor("docs/plans/a.md").ID, "no second task is created")
	assert.Equal(t, h.step(format.BoardDone), h.tasks.task(bID).WorkflowStepID)
	assert.Len(t, h.tasks.creates, 2)
}

// @covers AC-TASKS-PLAN-FILES-002.6
func TestSync_HiddenFileWithoutTaskCreatesNothing(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("hidden", "Alpha"))

	summary := h.sync()

	assert.Equal(t, PassCounts{}, summary.Counts)
	assert.Zero(t, h.tasks.writeCount())
}

// @covers AC-TASKS-PLAN-FILES-004.2
func TestSync_AdoptsArchivedTaskFromAnotherWorkflow(t *testing.T) {
	h := newSyncHarness(t)
	seeded := h.tasks.seedTask("legacy-task", "wf-other", "other-step", "legacy-42")
	_, err := h.tasks.ArchiveTaskTree(t.Context(), seeded.ID, false)
	require.NoError(t, err)
	h.tasks.writes = nil
	h.writeFile("docs/plans/a.md", planDoc("in_progress", "Alpha", "external_id: legacy-42"))

	summary := h.sync()

	got := h.tasks.task("legacy-task")
	assert.Nil(t, got.ArchivedAt, "an archived task is unarchived on adoption")
	assert.Equal(t, "wf-1", got.WorkflowID)
	assert.Equal(t, h.step(format.BoardInProgress), got.WorkflowStepID)
	assert.Equal(t, "Alpha", got.Title)
	assert.Empty(t, h.tasks.creates, "the adopted task is reused, not duplicated")
	assert.Equal(t, 1, summary.Counts.Unarchived)
	assert.Equal(t, 1, summary.Counts.Moved)
	require.Len(t, h.tasks.moves, 1)
	assert.Equal(t, "system", string(h.tasks.moves[0].Actor))
	row, err := h.svc.Store().GetTaskRow(t.Context(), "legacy-task")
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "legacy-42", row.ExternalID)
}

// @covers AC-TASKS-PLAN-FILES-004.3
func TestSync_DuplicateExternalIDFailsBothFiles(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "Alpha", "external_id: same"))
	h.writeFile("docs/plans/b.md", planDoc("queued", "Beta", "external_id: same"))
	h.writeFile("docs/plans/c.md", planDoc("queued", "Gamma"))

	summary := h.sync()

	assert.Equal(t, OutcomePartial, summary.Outcome)
	assert.Nil(t, h.tasks.byExternalID("same"), "neither duplicate is synced")
	assert.NotNil(t, h.taskFor("docs/plans/c.md"), "other files still sync")
	var dupPaths []string
	for _, fe := range summary.FileErrors {
		if fe.Reason == ReasonDuplicateExternal {
			dupPaths = append(dupPaths, fe.RelPath)
		}
	}
	assert.ElementsMatch(t, []string{"docs/plans/a.md", "docs/plans/b.md"}, dupPaths)
}

// @covers AC-TASKS-PLAN-FILES-004.3
func TestSync_DuplicateOfDefaultAndExplicitIDIsDetected(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "Alpha"))
	h.writeFile("docs/plans/b.md", planDoc("queued", "Beta",
		"external_id: plan-file:"+testRepoID+":docs/plans/a.md"))

	summary := h.sync()

	assert.Empty(t, h.tasks.creates)
	assert.Len(t, summary.FileErrors, 2)
}

// @covers AC-TASKS-PLAN-FILES-002.8
func TestSync_RunningSessionBlocksMoveUntilItSettles(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "Alpha"))
	h.sync()
	id := h.taskFor("docs/plans/a.md").ID
	h.tasks.setSession(id, taskmodels.TaskSessionStateRunning)
	h.writeFile("docs/plans/a.md", planDoc("done", "Alpha"))

	summary := h.sync()

	assert.Equal(t, h.step(format.BoardQueued), h.tasks.task(id).WorkflowStepID, "no move during a running turn")
	assert.Empty(t, h.tasks.moves)
	assert.Equal(t, 0, summary.Counts.Moved)
	assert.Equal(t, 0, summary.Counts.Failed, "a deferred move is not a failure")

	h.tasks.setSession(id, taskmodels.TaskSessionStateWaitingForInput)
	summary = h.sync()

	assert.Equal(t, h.step(format.BoardDone), h.tasks.task(id).WorkflowStepID)
	assert.Equal(t, 1, summary.Counts.Moved)
}

// @covers AC-TASKS-PLAN-FILES-002.8
func TestSync_StartingSessionAlsoBlocksMove(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "Alpha"))
	h.sync()
	id := h.taskFor("docs/plans/a.md").ID
	h.tasks.setSession(id, taskmodels.TaskSessionStateStarting)
	h.writeFile("docs/plans/a.md", planDoc("done", "Alpha"))

	summary := h.sync()

	assert.Empty(t, h.tasks.moves)
	assert.Equal(t, 0, summary.Counts.Failed)
}

// @covers AC-TASKS-PLAN-FILES-003.6
func TestSync_TaskInHandoffStepStaysWhileQueuedOrInProgress(t *testing.T) {
	for _, board := range []string{"queued", "in_progress"} {
		t.Run(board, func(t *testing.T) {
			h := newSyncHarness(t)
			h.writeFile("docs/plans/a.md", planDoc("queued", "Alpha"))
			h.sync()
			id := h.taskFor("docs/plans/a.md").ID
			_, err := h.tasks.MoveTaskWithOptions(t.Context(), id, "wf-1", handoffStep, 0, noMoveOptions())
			require.NoError(t, err)
			h.tasks.moves = nil
			h.writeFile("docs/plans/a.md", planDoc(board, "Alpha"))

			h.sync()
			h.sync()

			assert.Equal(t, handoffStep, h.tasks.task(id).WorkflowStepID)
			assert.Empty(t, h.tasks.moves, "the pass must not pull the task out of the handoff step")
		})
	}
}

// @covers AC-TASKS-PLAN-FILES-003.6
func TestSync_TaskInHandoffStepMovesForAnyOtherStatus(t *testing.T) {
	for _, board := range []format.BoardStatus{
		format.BoardWaitingOwner, format.BoardWaitingExternal, format.BoardDeferred, format.BoardDone,
	} {
		t.Run(string(board), func(t *testing.T) {
			h := newSyncHarness(t)
			h.writeFile("docs/plans/a.md", planDoc("queued", "Alpha"))
			h.sync()
			id := h.taskFor("docs/plans/a.md").ID
			_, err := h.tasks.MoveTaskWithOptions(t.Context(), id, "wf-1", handoffStep, 0, noMoveOptions())
			require.NoError(t, err)
			h.writeFile("docs/plans/a.md", planDoc(string(board), "Alpha"))

			h.sync()

			assert.Equal(t, h.step(board), h.tasks.task(id).WorkflowStepID)
		})
	}
}

// @covers AC-TASKS-PLAN-FILES-001.4
func TestSync_InvalidBoardValueKeepsStepAndShowsParseError(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/new.md", planDoc("someday", "New"))
	h.writeFile("docs/plans/old.md", planDoc("waiting_owner", "Old"))
	h.sync()
	oldID := h.taskFor("docs/plans/old.md").ID

	h.writeFile("docs/plans/old.md", planDoc("someday", "Old"))
	summary := h.sync()

	fresh := h.taskFor("docs/plans/new.md")
	assert.Equal(t, h.step(format.BoardQueued), fresh.WorkflowStepID, "a new task starts in the queued step")
	assert.Contains(t, fresh.Description, "> Parse error: board:")
	assert.Equal(t, h.step(format.BoardWaitingOwner), h.tasks.task(oldID).WorkflowStepID, "an existing task keeps its step")
	assert.Contains(t, h.tasks.task(oldID).Description, "> Parse error: board:")
	assert.Equal(t, 0, summary.Counts.Moved)
	assert.Equal(t, OutcomePartial, summary.Outcome)
}

// @covers AC-TASKS-PLAN-FILES-002.4
func TestSync_DescriptionLinksDependenciesThatExist(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "Alpha", "depends_on: [b.md, missing.md]"))
	h.writeFile("docs/plans/b.md", planDoc("queued", "Beta"))

	h.sync()

	bID := h.taskFor("docs/plans/b.md").ID
	desc := h.taskFor("docs/plans/a.md").Description
	assert.Contains(t, desc, "> Depends on: [b.md](/t/"+bID+")")
	assert.NotContains(t, desc, "missing.md")
	before := h.tasks.writeCount()
	h.sync()
	assert.Equal(t, before, h.tasks.writeCount(), "the corrected description is stable")
}
