package planfiles

import (
	"bytes"
	"encoding/json"
	"maps"
	"time"

	"github.com/kandev/kandev/internal/planfiles/format"
)

// Date kinds of the `card_display.date_kind` fact.
const (
	dateKindWaiting  = "waiting"
	dateKindDeferred = "deferred"
	dateKindDue      = "due"

	executorKindAgent = "agent"
)

// Card flags of the `card_display.flags` fact, in the order they are listed.
const (
	flagStale       = "stale"
	flagOpenItems   = "open_items"
	flagUncommitted = "uncommitted"
)

// cardInputs are the per-pass facts about a plan that its file alone does not
// hold. The zero value yields no progress and no flag.
type cardInputs struct {
	// Done and Total count the task-list items of the plan body plus its
	// tracked files and work orders.
	Done, Total int
	// Newest is the latest modification time among the plan file and its
	// tracked items; the zero time means unknown.
	Newest time.Time
	// Uncommitted reports that the plan file has uncommitted changes.
	Uncommitted bool
	// Now and StaleAfter decide staleness; StaleAfter 0 turns the flag off.
	Now        time.Time
	StaleAfter time.Duration
	// TurnRunning reports that an agent turn is starting or running.
	TurnRunning bool
}

// projectCardFacts is the `card_display` object a plan file projects into its
// task metadata, or nil when the file yields no facts. The sync pass owns the
// whole object: keys it does not project are removed. A done plan carries no
// date, executor, or progress, only flags. Every fact is added by its own
// helper so a new fact is one more entry here.
func projectCardFacts(pf format.PlanFile, in cardInputs) map[string]any {
	facts := map[string]any{}
	for _, add := range []func(map[string]any, format.PlanFile, cardInputs){
		addDateFacts, addExecutorFact, addProgressFact, addFlagsFact,
	} {
		add(facts, pf, in)
	}
	if len(facts) == 0 {
		return nil
	}
	return facts
}

func addDateFacts(facts map[string]any, pf format.PlanFile, _ cardInputs) {
	if pf.Date == "" || pf.Board == format.BoardDone {
		return
	}
	facts["date"] = pf.Date
	facts["date_kind"] = dateKindOf(pf.Board)
}

func addExecutorFact(facts map[string]any, pf format.PlanFile, _ cardInputs) {
	if pf.Board == format.BoardDone {
		return
	}
	if name := oneLine(pf.Executor); name != "" {
		facts["executor"] = map[string]any{"name": name, "kind": executorKindAgent}
	}
}

func addProgressFact(facts map[string]any, pf format.PlanFile, in cardInputs) {
	if pf.Board == format.BoardDone || in.Total <= 0 {
		return
	}
	facts["progress"] = map[string]any{"done": in.Done, "total": in.Total}
}

func addFlagsFact(facts map[string]any, pf format.PlanFile, in cardInputs) {
	var flags []string
	if staleByTime(pf, in) && !in.TurnRunning {
		flags = append(flags, flagStale)
	}
	if pf.Board == format.BoardDone && in.Done < in.Total {
		flags = append(flags, flagOpenItems)
	}
	if in.Uncommitted {
		flags = append(flags, flagUncommitted)
	}
	if len(flags) > 0 {
		facts["flags"] = flags
	}
}

// staleByTime reports that an in-progress plan and all its tracked items are
// older than the threshold, whatever its sessions are doing.
func staleByTime(pf format.PlanFile, in cardInputs) bool {
	return pf.Board == format.BoardInProgress && in.StaleAfter > 0 && !in.Newest.IsZero() &&
		in.Now.Sub(in.Newest) > in.StaleAfter
}

func dateKindOf(board format.BoardStatus) string {
	switch board {
	case format.BoardWaitingOwner, format.BoardWaitingExternal:
		return dateKindWaiting
	case format.BoardDeferred:
		return dateKindDeferred
	}
	return dateKindDue
}

// cardMetadata is the metadata map to create a task with: the facts alone, or
// nil when there are none.
func cardMetadata(facts map[string]any) map[string]any {
	if facts == nil {
		return nil
	}
	return map[string]any{cardDisplayKey: facts}
}

// mergeCardFacts returns the task metadata with `card_display` set to facts
// (removed when facts is nil) and every other key kept, and whether that
// differs from the current metadata. The current map is never modified.
func mergeCardFacts(current, facts map[string]any) (map[string]any, bool) {
	existing, has := current[cardDisplayKey]
	if !has && facts == nil {
		return current, false
	}
	if has && facts != nil && sameJSON(existing, facts) {
		return current, false
	}
	next := maps.Clone(current)
	if next == nil {
		next = map[string]any{}
	}
	if facts == nil {
		delete(next, cardDisplayKey)
	} else {
		next[cardDisplayKey] = facts
	}
	return next, true
}

// sameJSON compares two values by their canonical JSON form, which sorts map
// keys and so ignores the in-memory shape a store round trip produces.
func sameJSON(a, b any) bool {
	ja, errA := json.Marshal(a)
	jb, errB := json.Marshal(b)
	return errA == nil && errB == nil && bytes.Equal(ja, jb)
}
