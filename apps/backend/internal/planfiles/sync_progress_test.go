package planfiles

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	taskmodels "github.com/kandev/kandev/internal/task/models"
)

const (
	progressPlanRel = "docs/plans/p.md"
	workOrdersDir   = "docs/wo"
	cardKey         = "card_display"
)

var progressNow = time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)

// itemsPlan is a plan with 2 of 5 body items done. extra holds frontmatter lines.
func itemsPlan(board string, extra ...string) string {
	body := "- [x] a\n- [x] b\n- [ ] c\n- [ ] d\n- [ ] e\n"
	var b strings.Builder
	b.WriteString("---\nboard: " + board + "\n")
	for _, line := range extra {
		b.WriteString(line + "\n")
	}
	b.WriteString("---\n\n# Plan\n\n" + body)
	return b.String()
}

// writeWorkOrders writes total task files under dir, the first done of them
// with status done.
func (h *syncHarness) writeWorkOrders(dir string, done, total int) {
	h.t.Helper()
	for i := 1; i <= total; i++ {
		status := "pending"
		if i <= done {
			status = "done"
		}
		h.writeFile(fmt.Sprintf("%s/task-%02d-x.md", dir, i), "---\nstatus: "+status+"\n---\n# Work order\n")
	}
}

// age sets the modification time of a repository file.
func (h *syncHarness) age(rel string, mtime time.Time) {
	h.t.Helper()
	require.NoError(h.t, os.Chtimes(filepath.Join(h.root, filepath.FromSlash(rel)), mtime, mtime))
}

func (h *syncHarness) setNow(now time.Time) {
	h.svc.SetClock(func() time.Time { return now })
}

// card is the card_display object of a plan task as a store round trip returns
// it: JSON decoded, so numbers are float64 and lists are []any.
func (h *syncHarness) card(rel string) map[string]any {
	h.t.Helper()
	raw, err := json.Marshal(h.taskFor(rel).Metadata[cardKey])
	require.NoError(h.t, err)
	var card map[string]any
	require.NoError(h.t, json.Unmarshal(raw, &card))
	return card
}

func (h *syncHarness) flagsOf(rel string) []any {
	flags, _ := h.card(rel)["flags"].([]any)
	return flags
}

func (h *syncHarness) progressOf(rel string) any {
	return h.card(rel)["progress"]
}

func (h *syncHarness) errorReasons(summary PassSummary) []string {
	reasons := []string{}
	for _, fe := range summary.FileErrors {
		reasons = append(reasons, fe.Reason)
	}
	return reasons
}

// @covers AC-TASKS-PLAN-BOARD-OPS-005.2
// @covers AC-TASKS-PLAN-BOARD-OPS-005.3
func TestSync_ProgressCountsBodyItemsAndTrackedWorkOrders(t *testing.T) {
	h := newSyncHarness(t)
	h.writeWorkOrders(workOrdersDir, 1, 3)
	h.writeFile(progressPlanRel, itemsPlan("in_progress", "tracks:", "  - "+workOrdersDir))

	h.sync()

	assert.Equal(t, map[string]any{"done": float64(3), "total": float64(8)}, h.progressOf(progressPlanRel))
	assert.Empty(t, h.flagsOf(progressPlanRel))

	h.writeFile(progressPlanRel, itemsPlan("done", "tracks:", "  - "+workOrdersDir))
	h.sync()

	assert.Nil(t, h.progressOf(progressPlanRel), "a done plan shows no progress")
	assert.Equal(t, []any{"open_items"}, h.flagsOf(progressPlanRel))
}

// @covers AC-TASKS-PLAN-BOARD-OPS-005.2
func TestSync_TrackedFileItemsCountAndFrontmatterDoesNot(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/notes.md", "---\nlist:\n  - [x]\n---\n- [x] one\n- [ ] two\n")
	h.writeFile(progressPlanRel, planDoc("queued", "Plan", "tracks:", "  - docs/notes.md"))

	h.sync()

	assert.Equal(t, map[string]any{"done": float64(1), "total": float64(2)}, h.progressOf(progressPlanRel))
}

// @covers AC-TASKS-PLAN-BOARD-OPS-005.3
func TestSync_DonePlanWithAllItemsDoneHasNoFlag(t *testing.T) {
	h := newSyncHarness(t)
	h.writeWorkOrders(workOrdersDir, 3, 3)
	h.writeFile(progressPlanRel, planDoc("done", "Plan", "tracks:", "  - "+workOrdersDir))

	h.sync()

	assert.Nil(t, h.taskFor(progressPlanRel).Metadata[cardKey])
}

// @covers AC-TASKS-PLAN-BOARD-OPS-005.1
func TestSync_UnsafeTracksAreFileErrorsAndCountAsZero(t *testing.T) {
	h := newSyncHarness(t)
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "x.md"), []byte("- [ ] secret\n"), 0o644))
	require.NoError(t, os.Symlink(outside, filepath.Join(h.root, "linked")))
	require.NoError(t, os.Symlink(filepath.Join(outside, "x.md"), filepath.Join(h.root, "linked.md")))
	h.writeWorkOrders(workOrdersDir, 1, 2)
	h.writeFile(progressPlanRel, planDoc("queued", "Plan", "tracks:",
		"  - ../escape.md", "  - /etc/hosts", "  - linked", "  - linked.md", "  - docs/missing.md",
		"  - "+workOrdersDir))

	summary := h.sync()

	assert.Equal(t, OutcomePartial, summary.Outcome)
	assert.Equal(t, []string{"invalid_track", "invalid_track", "invalid_track", "invalid_track", "invalid_track"},
		h.errorReasons(summary))
	for _, fe := range summary.FileErrors {
		assert.Equal(t, progressPlanRel, fe.RelPath)
	}
	assert.Equal(t, map[string]any{"done": float64(1), "total": float64(2)}, h.progressOf(progressPlanRel))
	assert.NotNil(t, h.taskFor(progressPlanRel), "the plan still syncs")
}

// @covers AC-TASKS-PLAN-BOARD-OPS-005.1
func TestSync_TrackedDirectoryIsCappedAtTwoHundredWorkOrders(t *testing.T) {
	h := newSyncHarness(t)
	h.writeWorkOrders(workOrdersDir, 0, 203)
	h.writeFile(progressPlanRel, planDoc("queued", "Plan", "tracks:", "  - "+workOrdersDir))

	summary := h.sync()

	assert.Equal(t, []string{"truncated"}, h.errorReasons(summary))
	assert.Equal(t, map[string]any{"done": float64(0), "total": float64(200)}, h.progressOf(progressPlanRel))
}

// @covers AC-TASKS-PLAN-BOARD-OPS-005.4
func TestSync_StaleConsidersThePlanAndTheTrackedItems(t *testing.T) {
	h := newSyncHarness(t)
	h.setNow(progressNow)
	old := progressNow.Add(-10 * 24 * time.Hour)
	h.writeWorkOrders(workOrdersDir, 1, 2)
	h.writeFile(progressPlanRel, planDoc("in_progress", "Plan", "tracks:", "  - "+workOrdersDir))
	for _, rel := range []string{progressPlanRel, workOrdersDir + "/task-01-x.md", workOrdersDir + "/task-02-x.md", workOrdersDir} {
		h.age(rel, old)
	}

	h.sync()
	assert.Equal(t, []any{"stale"}, h.flagsOf(progressPlanRel))

	h.age(workOrdersDir+"/task-02-x.md", progressNow.Add(-time.Hour))
	h.sync()
	assert.Empty(t, h.flagsOf(progressPlanRel), "touching a tracked file removes the flag")

	h.age(workOrdersDir+"/task-02-x.md", old)
	h.age(workOrdersDir, old)
	h.sync()
	assert.Equal(t, []any{"stale"}, h.flagsOf(progressPlanRel))

	h.tasks.setSession(h.taskFor(progressPlanRel).ID, taskmodels.TaskSessionStateRunning)
	h.sync()
	assert.Empty(t, h.flagsOf(progressPlanRel), "a running turn removes the flag")
}

// @covers AC-TASKS-PLAN-BOARD-OPS-005.4
func TestSync_StaleOnlyForInProgressPlansAndOnlyWhenEnabled(t *testing.T) {
	h := newSyncHarness(t)
	h.setNow(progressNow)
	old := progressNow.Add(-30 * 24 * time.Hour)
	h.writeFile("docs/plans/q.md", planDoc("queued", "Queued"))
	h.writeFile("docs/plans/w.md", planDoc("waiting_owner", "Waiting"))
	h.writeFile("docs/plans/i.md", planDoc("in_progress", "Active"))
	for _, name := range []string{"q", "w", "i"} {
		h.age("docs/plans/"+name+".md", old)
	}

	h.sync()

	assert.Empty(t, h.flagsOf("docs/plans/q.md"))
	assert.Empty(t, h.flagsOf("docs/plans/w.md"))
	assert.Equal(t, []any{"stale"}, h.flagsOf("docs/plans/i.md"))

	req := validRequest()
	off := 0
	req.StaleAfterDays = &off
	_, err := h.svc.PutConfig(t.Context(), testWorkspace, req)
	require.NoError(t, err)
	h.sync()
	assert.Empty(t, h.flagsOf("docs/plans/i.md"), "a zero threshold turns the flag off")
}

// @covers AC-TASKS-PLAN-BOARD-OPS-008.1
func TestSync_UncommittedFlagFollowsTheGitState(t *testing.T) {
	h := newSyncHarness(t)
	h.initGit()

	h.sync()
	assert.Empty(t, h.flagsOf(gitPlanQueued), "a committed plan has no flag")

	h.writeFile(gitPlanQueued, planDoc("queued", "Queued", "owner: me"))
	h.writeFile(gitPlanNew, planDoc("queued", "New"))
	h.sync()
	assert.Equal(t, []any{"uncommitted"}, h.flagsOf(gitPlanQueued))
	assert.Equal(t, []any{"uncommitted"}, h.flagsOf(gitPlanNew), "an untracked plan is flagged")

	h.git("add", "-A")
	h.git("commit", "-q", "-m", "plans")
	h.sync()
	assert.Empty(t, h.flagsOf(gitPlanQueued))
	assert.Empty(t, h.flagsOf(gitPlanNew))
}

// @covers AC-TASKS-PLAN-BOARD-OPS-008.1
func TestSync_NoGitWorkingTreeMeansNoFlagAndNoError(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile(progressPlanRel, planDoc("queued", "Plan"))

	summary := h.sync()

	assert.Equal(t, OutcomeOK, summary.Outcome)
	assert.Nil(t, h.taskFor(progressPlanRel).Metadata[cardKey])
}

// @covers AC-TASKS-PLAN-BOARD-OPS-008.1
func TestSync_FailedGitStatusKeepsThePreviousFlagAndReportsTheRepository(t *testing.T) {
	h := newSyncHarness(t)
	h.initGit()
	h.writeFile(gitPlanQueued, planDoc("queued", "Queued", "owner: me"))
	h.sync()
	require.Equal(t, []any{"uncommitted"}, h.flagsOf(gitPlanQueued))
	h.git("add", "-A")
	h.git("commit", "-q", "-m", "plans")
	require.NoError(t, os.WriteFile(filepath.Join(h.root, ".git", "index"), []byte("not an index"), 0o644))

	summary := h.sync()

	assert.Equal(t, []any{"uncommitted"}, h.flagsOf(gitPlanQueued), "the flag is unchanged while git status fails")
	assert.Equal(t, []string{"git_status_failed"}, h.errorReasons(summary))
	assert.Empty(t, summary.FileErrors[0].RelPath)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-005.4
func TestSync_UnchangedFilesAndGitStateSendNoUpdate(t *testing.T) {
	h := newSyncHarness(t)
	h.initGit()
	h.setNow(progressNow)
	h.writeWorkOrders(workOrdersDir, 1, 3)
	h.writeFile(progressPlanRel, itemsPlan("in_progress", "tracks:", "  - "+workOrdersDir))
	nearlyStale := progressNow.Add(-(7*24*time.Hour - time.Hour))
	h.age(progressPlanRel, nearlyStale)
	for _, entry := range []string{"task-01-x.md", "task-02-x.md", "task-03-x.md"} {
		h.age(workOrdersDir+"/"+entry, nearlyStale)
	}
	h.age(workOrdersDir, nearlyStale)
	h.sync()
	require.Equal(t, []any{"uncommitted"}, h.flagsOf(progressPlanRel))
	calls, writes := len(h.tasks.updates), h.tasks.writeCount()

	h.setNow(progressNow.Add(10 * time.Minute))
	first := h.sync()
	h.setNow(progressNow.Add(20 * time.Minute))
	second := h.sync()

	assert.Equal(t, PassCounts{}, first.Counts)
	assert.Equal(t, PassCounts{}, second.Counts)
	assert.Len(t, h.tasks.updates, calls, "no UpdateTask call over two unchanged passes")
	assert.Equal(t, writes, h.tasks.writeCount())
}

// @covers AC-TASKS-PLAN-BOARD-OPS-005.4
func TestSync_StaleFlippingByTimeAloneSendsExactlyOneUpdate(t *testing.T) {
	h := newSyncHarness(t)
	h.setNow(progressNow)
	h.writeFile(progressPlanRel, planDoc("in_progress", "Plan"))
	h.age(progressPlanRel, progressNow.Add(-(7*24*time.Hour - time.Hour)))
	h.sync()
	require.Empty(t, h.flagsOf(progressPlanRel))
	calls := len(h.tasks.updates)

	h.setNow(progressNow.Add(2 * time.Hour))
	h.sync()
	assert.Len(t, h.tasks.updates, calls+1)
	assert.Equal(t, []any{"stale"}, h.flagsOf(progressPlanRel))

	h.setNow(progressNow.Add(3 * time.Hour))
	h.sync()
	assert.Len(t, h.tasks.updates, calls+1, "the flag is already set")
}
