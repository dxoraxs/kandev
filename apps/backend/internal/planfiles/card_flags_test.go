package planfiles

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kandev/kandev/internal/planfiles/format"
)

var flagsNow = time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)

const flagsWeek = 7 * 24 * time.Hour

func planOn(board format.BoardStatus) format.PlanFile {
	return format.PlanFile{Board: board, Title: "Plan", Priority: "medium"}
}

func staleInputs(age time.Duration) cardInputs {
	return cardInputs{Now: flagsNow, StaleAfter: flagsWeek, Newest: flagsNow.Add(-age)}
}

func withTurn(in cardInputs) cardInputs {
	in.TurnRunning = true
	return in
}

// @covers AC-TASKS-PLAN-BOARD-OPS-005.2
func TestProjectCardFacts_ProgressCountsBodyPlusTracked(t *testing.T) {
	facts := projectCardFacts(planOn(format.BoardInProgress), cardInputs{Done: 3, Total: 8})

	assert.Equal(t, map[string]any{"progress": map[string]any{"done": 3, "total": 8}}, facts)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-005.2
func TestProjectCardFacts_NoItemsNoProgress(t *testing.T) {
	assert.Nil(t, projectCardFacts(planOn(format.BoardInProgress), cardInputs{}))
}

// @covers AC-TASKS-PLAN-BOARD-OPS-005.2
// @covers AC-TASKS-PLAN-BOARD-OPS-005.3
func TestProjectCardFacts_DonePlanHasNoProgressAndTheOpenItemsFlag(t *testing.T) {
	facts := projectCardFacts(planOn(format.BoardDone), cardInputs{Done: 3, Total: 8})

	assert.Equal(t, map[string]any{"flags": []string{"open_items"}}, facts)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-005.3
func TestProjectCardFacts_DonePlanWithEveryItemDoneHasNoFacts(t *testing.T) {
	assert.Nil(t, projectCardFacts(planOn(format.BoardDone), cardInputs{Done: 8, Total: 8}))
	assert.Nil(t, projectCardFacts(planOn(format.BoardDone), cardInputs{}))
}

// @covers AC-TASKS-PLAN-BOARD-OPS-005.3
func TestProjectCardFacts_OpenItemsOnlyForDonePlans(t *testing.T) {
	assert.Nil(t, projectCardFacts(planOn(format.BoardQueued), cardInputs{}))
	facts := projectCardFacts(planOn(format.BoardQueued), cardInputs{Done: 1, Total: 2})
	assert.NotContains(t, facts, "flags")
}

// @covers AC-TASKS-PLAN-BOARD-OPS-005.4
func TestProjectCardFacts_StaleOnlyForInProgressPlansOlderThanTheThreshold(t *testing.T) {
	ancient := 30 * 24 * time.Hour
	cases := []struct {
		name  string
		board format.BoardStatus
		in    cardInputs
		stale bool
	}{
		{"older than the threshold", format.BoardInProgress, staleInputs(flagsWeek + time.Second), true},
		{"exactly the threshold", format.BoardInProgress, staleInputs(flagsWeek), false},
		{"younger", format.BoardInProgress, staleInputs(time.Hour), false},
		{"queued", format.BoardQueued, staleInputs(ancient), false},
		{"waiting", format.BoardWaitingOwner, staleInputs(ancient), false},
		{"done", format.BoardDone, staleInputs(ancient), false},
		{"threshold off", format.BoardInProgress, cardInputs{Now: flagsNow, Newest: flagsNow.Add(-ancient)}, false},
		{"turn running", format.BoardInProgress, withTurn(staleInputs(ancient)), false},
		{"unknown time", format.BoardInProgress, cardInputs{Now: flagsNow, StaleAfter: flagsWeek}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			facts := projectCardFacts(planOn(c.board), c.in)
			if c.stale {
				assert.Equal(t, map[string]any{"flags": []string{"stale"}}, facts)
				return
			}
			assert.NotContains(t, facts, "flags")
		})
	}
}

// @covers AC-TASKS-PLAN-BOARD-OPS-008.1
func TestProjectCardFacts_UncommittedFlag(t *testing.T) {
	facts := projectCardFacts(planOn(format.BoardQueued), cardInputs{Uncommitted: true})

	assert.Equal(t, map[string]any{"flags": []string{"uncommitted"}}, facts)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-005.4
func TestProjectCardFacts_FlagsKeepAFixedOrder(t *testing.T) {
	in := staleInputs(30 * 24 * time.Hour)
	in.Uncommitted = true
	in.Done, in.Total = 1, 4

	facts := projectCardFacts(planOn(format.BoardInProgress), in)

	assert.Equal(t, []string{"stale", "uncommitted"}, facts["flags"])
	assert.Equal(t, map[string]any{"done": 1, "total": 4}, facts["progress"])

	done := projectCardFacts(planOn(format.BoardDone), cardInputs{Done: 1, Total: 4, Uncommitted: true})
	assert.Equal(t, map[string]any{"flags": []string{"open_items", "uncommitted"}}, done)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-005.2
func TestProjectCardFacts_ProgressAndFlagsSitBesideDateAndExecutor(t *testing.T) {
	pf := planOn(format.BoardInProgress)
	pf.Date, pf.Executor = "2026-10-12", "Claude"

	facts := projectCardFacts(pf, cardInputs{Done: 1, Total: 2, Uncommitted: true})

	assert.Equal(t, map[string]any{
		"date": "2026-10-12", "date_kind": "due",
		"executor": map[string]any{"name": "Claude", "kind": "agent"},
		"progress": map[string]any{"done": 1, "total": 2},
		"flags":    []string{"uncommitted"},
	}, facts)
}
