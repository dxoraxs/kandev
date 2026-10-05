package planfiles

import (
	"bytes"
	"encoding/json"
	"maps"

	"github.com/kandev/kandev/internal/planfiles/format"
)

// Date kinds of the `card_display.date_kind` fact.
const (
	dateKindWaiting  = "waiting"
	dateKindDeferred = "deferred"
	dateKindDue      = "due"

	executorKindAgent = "agent"
)

// projectCardFacts is the `card_display` object a plan file projects into its
// task metadata, or nil when the file yields no facts. The sync pass owns the
// whole object: keys it does not project are removed. A done plan has no facts.
// Every fact is added by its own helper so a new fact is one more entry here.
func projectCardFacts(pf format.PlanFile) map[string]any {
	if pf.Board == format.BoardDone {
		return nil
	}
	facts := map[string]any{}
	for _, add := range []func(map[string]any, format.PlanFile){addDateFacts, addExecutorFact} {
		add(facts, pf)
	}
	if len(facts) == 0 {
		return nil
	}
	return facts
}

func addDateFacts(facts map[string]any, pf format.PlanFile) {
	if pf.Date == "" {
		return
	}
	facts["date"] = pf.Date
	facts["date_kind"] = dateKindOf(pf.Board)
}

func addExecutorFact(facts map[string]any, pf format.PlanFile) {
	if name := oneLine(pf.Executor); name != "" {
		facts["executor"] = map[string]any{"name": name, "kind": executorKindAgent}
	}
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
