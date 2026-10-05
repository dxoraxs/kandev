package planfiles

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	depA = "docs/plans/a.md"
	depB = "docs/plans/b.md"
	depC = "docs/plans/c.md"
)

func (h *syncHarness) taskID(rel string) string { return h.taskFor(rel).ID }

func (h *syncHarness) reasonsFor(summary PassSummary, rel string) []string {
	var reasons []string
	for _, fe := range summary.FileErrors {
		if fe.RelPath == rel {
			reasons = append(reasons, fe.Reason)
		}
	}
	return reasons
}

func (h *syncHarness) storedDeps(rel string) []string {
	h.t.Helper()
	rows, err := h.svc.store.ListTaskRows(context.Background(), testWorkspace)
	require.NoError(h.t, err)
	for _, row := range rows {
		if row.RelPath == rel {
			return row.SyncedDependsOn
		}
	}
	h.t.Fatalf("no row for %s", rel)
	return nil
}

// @covers AC-TASKS-PLAN-BOARD-OPS-004.1
func TestDependencies_PassMakesTheTaskDependOnExactlyTheListedPlans(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile(depA, planDoc("queued", "A"))
	h.writeFile(depB, planDoc("queued", "B"))
	h.writeFile(depC, planDoc("queued", "C", "depends_on: [a.md, b.md]"))

	summary := h.sync()

	assert.Empty(t, summary.FileErrors)
	want := []string{h.taskID(depA), h.taskID(depB)}
	assert.ElementsMatch(t, want, h.tasks.dependenciesOf(h.taskID(depC)))
	assert.ElementsMatch(t, want, h.storedDeps(depC))
}

// A predecessor created in the same pass resolves in that pass: every task is
// resolved before any dependency is applied.
func TestDependencies_PredecessorCreatedInTheSamePassResolvesInThatPass(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile(depA, planDoc("queued", "A", "depends_on: [b.md]"))
	h.writeFile(depB, planDoc("queued", "B"))

	summary := h.sync()

	assert.Empty(t, summary.FileErrors)
	assert.Equal(t, []string{h.taskID(depB)}, h.tasks.dependenciesOf(h.taskID(depA)))
}

// @covers AC-TASKS-PLAN-BOARD-OPS-004.1
func TestDependencies_RemovingAnEntryRemovesOnlyTheDependencyTheFeatureCreated(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile(depA, planDoc("queued", "A"))
	h.writeFile(depB, planDoc("queued", "B"))
	h.writeFile(depC, planDoc("queued", "C", "depends_on: [a.md, b.md]"))
	h.sync()
	manual := h.tasks.seedTask("manual-task", "wf-1", h.step("queued"), "")
	h.tasks.seedDependency(h.taskID(depC), manual.ID)

	h.writeFile(depC, planDoc("queued", "C", "depends_on: [a.md]"))
	h.sync()

	assert.ElementsMatch(t, []string{h.taskID(depA), manual.ID}, h.tasks.dependenciesOf(h.taskID(depC)))
	assert.Equal(t, []string{h.taskID(depA)}, h.storedDeps(depC))
}

func TestDependencies_KeepsAManualDependencyOnAnotherPlanTaskNeverListed(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile(depA, planDoc("queued", "A"))
	h.writeFile(depB, planDoc("queued", "B"))
	h.writeFile(depC, planDoc("queued", "C", "depends_on: [a.md]"))
	h.sync()
	h.tasks.seedDependency(h.taskID(depC), h.taskID(depB))

	h.writeFile(depC, planDoc("queued", "C", "depends_on: []"))
	h.sync()

	assert.Equal(t, []string{h.taskID(depB)}, h.tasks.dependenciesOf(h.taskID(depC)))
	assert.Empty(t, h.storedDeps(depC))
}

// @covers AC-TASKS-PLAN-BOARD-OPS-004.2
func TestDependencies_CycleIsReportedForTheRejectedEdgeAndThePassCompletes(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile(depA, planDoc("queued", "A", "depends_on: [b.md]"))
	h.writeFile(depB, planDoc("queued", "B", "depends_on: [a.md]"))
	h.writeFile(depC, planDoc("queued", "C"))

	summary := h.sync()

	assert.Equal(t, OutcomePartial, summary.Outcome)
	assert.Equal(t, []string{ReasonInvalidDependency}, h.reasonsFor(summary, depB))
	assert.Empty(t, h.reasonsFor(summary, depA))
	assert.Equal(t, []string{h.taskID(depB)}, h.tasks.dependenciesOf(h.taskID(depA)))
	assert.Empty(t, h.tasks.dependenciesOf(h.taskID(depB)))
	assert.Empty(t, h.storedDeps(depB), "the rejected edge is not recorded")
	assert.NotNil(t, h.tasks.task(h.taskID(depC)))
}

func TestDependencies_RejectedEdgeIsRetriedOnTheNextPass(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile(depA, planDoc("queued", "A", "depends_on: [b.md]"))
	h.writeFile(depB, planDoc("queued", "B", "depends_on: [a.md]"))
	h.sync()

	h.writeFile(depA, planDoc("queued", "A"))
	summary := h.sync()

	assert.Empty(t, summary.FileErrors)
	assert.Equal(t, []string{h.taskID(depA)}, h.tasks.dependenciesOf(h.taskID(depB)))
}

func TestDependencies_SelfReferenceIsInvalidAndMakesNoCall(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile(depA, planDoc("queued", "A", "depends_on: [a.md]"))

	summary := h.sync()

	assert.Equal(t, []string{ReasonInvalidDependency}, h.reasonsFor(summary, depA))
	assert.Zero(t, h.tasks.dependencyCallCount())
}

// @covers AC-TASKS-PLAN-BOARD-OPS-004.4
func TestDependencies_UnchangedListMakesNoDependencyCalls(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile(depA, planDoc("queued", "A"))
	h.writeFile(depC, planDoc("queued", "C", "depends_on: [a.md]"))
	h.sync()
	before := h.tasks.dependencyCallCount()
	require.Positive(t, before)

	h.sync()
	h.sync()

	assert.Equal(t, before, h.tasks.dependencyCallCount())
}

func TestDependencies_PlansWithoutDependsOnMakeNoDependencyCalls(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile(depA, planDoc("queued", "A"))
	h.writeFile(depB, planDoc("done", "B"))

	h.sync()

	assert.Zero(t, h.tasks.dependencyCallCount())
}

// @covers AC-TASKS-PLAN-BOARD-OPS-004.2
func TestDependencies_EntryWithoutAPlanTaskIsUnknownAndOthersStillApply(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile(depA, planDoc("queued", "A"))
	h.writeFile("docs/plans/notes.md", "# Not a plan\n")
	h.writeFile(depC, planDoc("queued", "C", "depends_on: [missing.md, notes.md, a.md]"))

	summary := h.sync()

	assert.Equal(t, []string{ReasonUnknownDependency}, h.reasonsFor(summary, depC))
	assert.Equal(t, []string{h.taskID(depA)}, h.tasks.dependenciesOf(h.taskID(depC)))
}

func TestDependencies_EntryThatLeavesTheDirectoryOrIsNotAPlanFileNameIsInvalid(t *testing.T) {
	for _, entry := range []string{"../x.md", "sub/x.md", "/etc/passwd", "..", "."} {
		t.Run(entry, func(t *testing.T) {
			h := newSyncHarness(t)
			h.writeFile(depA, planDoc("queued", "A"))
			h.writeFile(depC, planDoc("queued", "C", `depends_on: ["`+entry+`"]`))

			summary := h.sync()

			assert.Equal(t, []string{ReasonInvalidDependency}, h.reasonsFor(summary, depC))
			assert.Empty(t, h.tasks.dependenciesOf(h.taskID(depC)))
			assert.Zero(t, h.tasks.dependencyCallCount())
		})
	}
}

func TestDependencies_NameWithoutMarkdownExtensionIsInvalid(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile(depC, planDoc("queued", "C", "depends_on: [a.txt]"))

	summary := h.sync()

	assert.Equal(t, []string{ReasonInvalidDependency}, h.reasonsFor(summary, depC))
}

func TestDependencies_PredecessorFileRemovalDropsTheEdgeAndReportsUnknown(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile(depA, planDoc("queued", "A"))
	h.writeFile(depC, planDoc("queued", "C", "depends_on: [a.md]"))
	h.sync()

	h.writeFile(depA, planDoc("hidden", "A"))
	summary := h.sync()

	assert.Equal(t, []string{ReasonUnknownDependency}, h.reasonsFor(summary, depC))
	assert.Empty(t, h.tasks.dependenciesOf(h.taskID(depC)))
	assert.Empty(t, h.storedDeps(depC))
}

func TestDependencies_RemovalOfAnAlreadyDeletedBlockerIsIgnored(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile(depA, planDoc("queued", "A"))
	h.writeFile(depC, planDoc("queued", "C", "depends_on: [a.md]"))
	h.sync()
	h.tasks.mu.Lock()
	delete(h.tasks.tasks, h.taskIDLocked(depA))
	h.tasks.mu.Unlock()

	h.writeFile(depC, planDoc("queued", "C"))
	summary := h.sync()

	for _, fe := range summary.FileErrors {
		assert.NotEqual(t, ReasonInvalidDependency, fe.Reason)
	}
	assert.Empty(t, h.storedDeps(depC))
}

func (h *syncHarness) taskIDLocked(rel string) string {
	for id, t := range h.tasks.tasks {
		if t.ExternalID == defaultExternalID(testRepoID, rel) {
			return id
		}
	}
	return ""
}
