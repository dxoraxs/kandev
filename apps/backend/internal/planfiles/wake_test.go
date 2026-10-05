package planfiles

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/common/logger"
	"github.com/kandev/kandev/internal/planfiles/format"
)

const wakePlanRel = "docs/plans/w.md"

type dateCall struct {
	workspaceID string
	title       string
	relPath     string
}

// fakeDateNotifier records every NotifyPlanDateReached call.
type fakeDateNotifier struct {
	mu    sync.Mutex
	calls []dateCall
	err   error
}

func (f *fakeDateNotifier) NotifyPlanDateReached(_ context.Context, workspaceID, title, relPath string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, dateCall{workspaceID: workspaceID, title: title, relPath: relPath})
	return f.err
}

func (f *fakeDateNotifier) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func wakeCount() int64 { return wakeTotal.Value() }

func localDay(year int, month time.Month, day, hour, min, sec int) time.Time {
	return time.Date(year, month, day, hour, min, sec, 0, time.Local)
}

// wakeHarness is a harness with a notifier and a plan file synced the day
// before its date.
func wakeHarness(t *testing.T, board string, extra ...string) (*syncHarness, *fakeDateNotifier) {
	t.Helper()
	h := newSyncHarness(t)
	notifier := &fakeDateNotifier{}
	h.svc.SetDateNotifier(notifier)
	h.setNow(localDay(2026, 8, 1, 12, 0, 0))
	h.writeFile(wakePlanRel, planDoc(board, "Plan", extra...))
	h.sync()
	require.Equal(t, 0, notifier.count())
	return h, notifier
}

func (h *syncHarness) woken() string {
	return planDoc("waiting_owner", "Plan", "date: 2026-10-06") + "\n## Owner notes\n\n"
}

// @covers AC-TASKS-PLAN-BOARD-OPS-003.1
// @covers AC-TASKS-PLAN-BOARD-OPS-003.3
func TestWake_WaitingStatusesComeBackToTheOwnerOnTheirDate(t *testing.T) {
	for _, status := range []string{"waiting_external", "deferred"} {
		t.Run(status, func(t *testing.T) {
			h, notifier := wakeHarness(t, status, "date: 2026-10-06")
			h.setNow(localDay(2026, 10, 6, 12, 0, 0))
			h.tasks.moves = nil
			before := wakeCount()

			summary := h.sync()

			assert.Equal(t, OutcomeOK, summary.Outcome, "%+v", summary.FileErrors)
			assert.Equal(t, h.woken()+"- 2026-10-06 date reached: was "+status+"\n", h.read(wakePlanRel))
			assert.Equal(t, h.step(format.BoardWaitingOwner), h.taskFor(wakePlanRel).WorkflowStepID)
			require.Len(t, h.tasks.moves, 1)
			assert.Equal(t, h.step(format.BoardWaitingOwner), h.tasks.moves[0].StepID)
			assert.Equal(t, []dateCall{{workspaceID: testWorkspace, title: "Plan", relPath: wakePlanRel}}, notifier.calls)
			assert.Equal(t, before+1, wakeCount())

			// A second pass, also after a restart, changes and sends nothing.
			content, writes, mtime := h.read(wakePlanRel), h.tasks.writeCount(), h.mtime(wakePlanRel)
			h.sync()
			assert.Equal(t, content, h.read(wakePlanRel))
			assert.Equal(t, mtime, h.mtime(wakePlanRel))
			assert.Equal(t, writes, h.tasks.writeCount())
			assert.Equal(t, 1, notifier.count())
		})
	}
}

// @covers AC-TASKS-PLAN-BOARD-OPS-003.1
func TestWake_AWakeUpIsOnceEvenAfterARestart(t *testing.T) {
	h, notifier := wakeHarness(t, "waiting_external", "date: 2026-10-06")
	h.setNow(localDay(2026, 10, 6, 12, 0, 0))
	h.sync()
	require.Equal(t, 1, notifier.count())

	restarted := NewService(h.svc.store, h.wf, h.wf, logger.Default())
	restarted.SetSyncDeps(h.tasks, h.tasks)
	restarted.SetClock(func() time.Time { return localDay(2026, 10, 6, 12, 5, 0) })
	again := &fakeDateNotifier{}
	restarted.SetDateNotifier(again)
	content, writes := h.read(wakePlanRel), h.tasks.writeCount()

	_, err := restarted.SyncWorkspace(context.Background(), testWorkspace)

	require.NoError(t, err)
	assert.Equal(t, 0, again.count())
	assert.Equal(t, content, h.read(wakePlanRel))
	assert.Equal(t, writes, h.tasks.writeCount())
}

// @covers AC-TASKS-PLAN-BOARD-OPS-003.1
func TestWake_APastDateWakesWithTodayInTheNote(t *testing.T) {
	h, notifier := wakeHarness(t, "waiting_external", "date: 2026-09-01")
	h.setNow(localDay(2026, 10, 6, 12, 0, 0))

	h.sync()

	assert.Contains(t, h.read(wakePlanRel), "- 2026-10-06 date reached: was waiting_external\n")
	assert.Equal(t, 1, notifier.count())
}

// @covers AC-TASKS-PLAN-BOARD-OPS-003.1
func TestWake_AFutureDateDoesNothing(t *testing.T) {
	h, notifier := wakeHarness(t, "waiting_external", "date: 2026-10-07")
	h.setNow(localDay(2026, 10, 6, 23, 59, 0))
	content, writes := h.read(wakePlanRel), h.tasks.writeCount()

	h.sync()

	assert.Equal(t, content, h.read(wakePlanRel))
	assert.Equal(t, writes, h.tasks.writeCount())
	assert.Equal(t, 0, notifier.count())
}

// @covers AC-TASKS-PLAN-BOARD-OPS-003.1
func TestWake_TodayIsTheServerLocalDay(t *testing.T) {
	h, notifier := wakeHarness(t, "waiting_external", "date: 2026-10-06")

	h.setNow(localDay(2026, 10, 5, 23, 59, 59))
	h.sync()
	assert.Equal(t, 0, notifier.count(), "the day before the date, one second before local midnight")
	assert.Contains(t, h.read(wakePlanRel), "board: waiting_external")

	h.setNow(localDay(2026, 10, 6, 0, 0, 0))
	h.sync()
	assert.Equal(t, 1, notifier.count(), "local midnight starts the date")
	assert.Contains(t, h.read(wakePlanRel), "- 2026-10-06 date reached: was waiting_external\n")
}

// @covers AC-TASKS-PLAN-BOARD-OPS-003.2
func TestWake_OnlyWaitingStatusesAreChangedWhateverTheirDate(t *testing.T) {
	for _, status := range []string{"queued", "in_progress", "waiting_owner", "done"} {
		t.Run(status, func(t *testing.T) {
			h, notifier := wakeHarness(t, status, "date: 2026-09-01")
			h.setNow(localDay(2026, 10, 6, 12, 0, 0))
			content, writes := h.read(wakePlanRel), h.tasks.writeCount()

			h.sync()

			assert.Equal(t, content, h.read(wakePlanRel))
			assert.Equal(t, writes, h.tasks.writeCount())
			assert.Equal(t, 0, notifier.count())
		})
	}
}

// @covers AC-TASKS-PLAN-BOARD-OPS-003.2
func TestWake_AnInvalidDateIsNeverReached(t *testing.T) {
	h, notifier := wakeHarness(t, "waiting_external", "date: 2026-13-45")
	h.setNow(localDay(2026, 10, 6, 12, 0, 0))
	content := h.read(wakePlanRel)

	h.sync()

	assert.Equal(t, content, h.read(wakePlanRel))
	assert.Equal(t, 0, notifier.count())
}

// @covers AC-TASKS-PLAN-BOARD-OPS-003.1
func TestWake_TurnedOffChangesNoFile(t *testing.T) {
	h, notifier := wakeHarness(t, "waiting_external", "date: 2026-10-06")
	off := false
	req := validRequest()
	req.WakeOnDate = &off
	_, err := h.svc.PutConfig(context.Background(), testWorkspace, req)
	require.NoError(t, err)
	h.setNow(localDay(2026, 10, 7, 12, 0, 0))
	content := h.read(wakePlanRel)

	h.sync()

	assert.Equal(t, content, h.read(wakePlanRel))
	assert.Equal(t, 0, notifier.count())
	assert.Equal(t, h.step(format.BoardWaitingExternal), h.taskFor(wakePlanRel).WorkflowStepID)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-003.1
func TestWake_TheTaskCardAndTheIndexShowTheNewStatusInTheSamePass(t *testing.T) {
	h := newSyncHarness(t)
	h.setIndexFile(indexName)
	h.setNow(localDay(2026, 10, 5, 12, 0, 0))
	h.writeFile(wakePlanRel, planDoc("deferred", "Plan", "date: 2026-10-06"))
	h.sync()
	require.Equal(t, "deferred", h.card(wakePlanRel)["date_kind"])
	require.Contains(t, h.read(indexRel), "## deferred")

	h.setNow(localDay(2026, 10, 6, 12, 0, 0))
	h.sync()

	assert.Equal(t, h.step(format.BoardWaitingOwner), h.taskFor(wakePlanRel).WorkflowStepID)
	assert.Equal(t, "waiting", h.card(wakePlanRel)["date_kind"])
	index := h.read(indexRel)
	assert.Contains(t, index, "## waiting_owner")
	assert.NotContains(t, index, "## deferred")
}

// @covers AC-TASKS-PLAN-BOARD-OPS-003.1
func TestWake_AFileChangedBetweenReadAndWriteFailsTheWakeUpAndIsRetried(t *testing.T) {
	h, notifier := wakeHarness(t, "waiting_external", "date: 2026-10-06")
	h.setNow(localDay(2026, 10, 6, 12, 0, 0))
	full := filepath.Join(h.root, filepath.FromSlash(wakePlanRel))
	edited := planDoc("waiting_external", "Plan", "date: 2026-10-06") + "\nAn edit made during the pass.\n"
	h.tasks.onGetTask = func(string) {
		h.tasks.onGetTask = nil
		require.NoError(t, os.WriteFile(full, []byte(edited), 0o644))
	}
	before := wakeCount()

	summary := h.sync()

	assert.Equal(t, OutcomePartial, summary.Outcome)
	require.Len(t, summary.FileErrors, 1)
	assert.Equal(t, ReasonWakeFailed, summary.FileErrors[0].Reason)
	assert.Equal(t, wakePlanRel, summary.FileErrors[0].RelPath)
	assert.Equal(t, edited, h.read(wakePlanRel), "the concurrent edit is never overwritten")
	assert.Equal(t, 0, notifier.count())
	assert.Equal(t, before, wakeCount())

	summary = h.sync()

	assert.Equal(t, OutcomeOK, summary.Outcome, "%+v", summary.FileErrors)
	assert.Contains(t, h.read(wakePlanRel), "An edit made during the pass.")
	assert.Contains(t, h.read(wakePlanRel), "- 2026-10-06 date reached: was waiting_external\n")
	assert.Equal(t, 1, notifier.count())
}

// @covers AC-TASKS-PLAN-BOARD-OPS-003.3
func TestWake_ANotifierErrorDoesNotUndoTheWakeUpOrFailThePass(t *testing.T) {
	h, notifier := wakeHarness(t, "waiting_external", "date: 2026-10-06")
	notifier.err = errors.New("provider down")
	h.setNow(localDay(2026, 10, 6, 12, 0, 0))

	summary := h.sync()

	assert.Equal(t, OutcomeOK, summary.Outcome, "%+v", summary.FileErrors)
	assert.Contains(t, h.read(wakePlanRel), "board: waiting_owner")
	assert.Equal(t, h.step(format.BoardWaitingOwner), h.taskFor(wakePlanRel).WorkflowStepID)
	assert.Equal(t, 1, notifier.count())
}

// @covers AC-TASKS-PLAN-BOARD-OPS-003.3
func TestWake_WithoutANotifierTheWakeUpStillHappens(t *testing.T) {
	h := newSyncHarness(t)
	h.setNow(localDay(2026, 10, 5, 12, 0, 0))
	h.writeFile(wakePlanRel, planDoc("deferred", "Plan", "date: 2026-10-06"))
	h.sync()
	h.setNow(localDay(2026, 10, 6, 12, 0, 0))

	summary := h.sync()

	assert.Equal(t, OutcomeOK, summary.Outcome, "%+v", summary.FileErrors)
	assert.Contains(t, h.read(wakePlanRel), "- 2026-10-06 date reached: was deferred\n")
}

func TestWake_NotesHeadingComesFromTheConfig(t *testing.T) {
	h, _ := wakeHarness(t, "waiting_external", "date: 2026-10-06")
	heading := "Journal"
	req := validRequest()
	req.NotesHeading = &heading
	_, err := h.svc.PutConfig(context.Background(), testWorkspace, req)
	require.NoError(t, err)
	h.setNow(localDay(2026, 10, 6, 12, 0, 0))

	h.sync()

	content := h.read(wakePlanRel)
	assert.Contains(t, content, "\n## Journal\n\n- 2026-10-06 date reached: was waiting_external\n")
	assert.False(t, strings.Contains(content, emDash), "the note uses no long dash")
}

// @covers AC-TASKS-PLAN-BOARD-OPS-003.1
func TestWake_AWrittenPlanShowsAsUncommittedInTheSamePass(t *testing.T) {
	h := newSyncHarness(t)
	h.initGit()
	h.setNow(localDay(2026, 10, 5, 12, 0, 0))
	h.writeFile(wakePlanRel, planDoc("waiting_external", "Plan", "date: 2026-10-06"))
	h.git("add", "-A")
	h.git("commit", "-q", "-m", "plan")
	h.sync()
	require.Empty(t, h.flagsOf(wakePlanRel))

	h.setNow(localDay(2026, 10, 6, 12, 0, 0))
	h.sync()

	assert.Equal(t, []any{"uncommitted"}, h.flagsOf(wakePlanRel))
}
