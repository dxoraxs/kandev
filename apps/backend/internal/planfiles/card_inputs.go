package planfiles

import (
	"context"
	"time"

	"github.com/kandev/kandev/internal/planfiles/format"
	"github.com/kandev/kandev/internal/planfiles/scan"
	taskmodels "github.com/kandev/kandev/internal/task/models"
)

const hoursPerDay = 24

// planItems is what a plan file and its tracked entries contribute to the card:
// the item counts and the latest modification time.
type planItems struct {
	Done, Total int
	Newest      time.Time
}

func (it *planItems) touch(t time.Time) {
	if t.After(it.Newest) {
		it.Newest = t
	}
}

// addTrack folds a tracked file or directory into the counts. A tracked
// directory contributes one item per work order, done when its frontmatter
// status is done; a tracked file contributes its own task-list items.
func (it *planItems) addTrack(track scan.Track) {
	it.touch(track.ModTime)
	for _, f := range track.Files {
		it.touch(f.ModTime)
		if track.Dir {
			it.Total++
			if format.WorkOrderDone(f.Content) {
				it.Done++
			}
			continue
		}
		done, total := format.CountItems(format.BodyOf(f.Content))
		it.Done += done
		it.Total += total
	}
}

// readItems counts the items of a plan file and reads its tracked entries from
// the repository. An unreadable entry is a file error of the plan and
// contributes nothing. A hidden plan has no task and reads no tracks.
func (p *pass) readItems(repo *taskmodels.Repository, relPath string, pf format.PlanFile, modTime time.Time) planItems {
	items := planItems{Newest: modTime}
	items.Done, items.Total = format.CountItems(pf.Body)
	if pf.Board == format.BoardHidden {
		return items
	}
	seen := map[string]struct{}{}
	for _, rel := range pf.Tracks {
		if _, dup := seen[rel]; dup {
			continue
		}
		seen[rel] = struct{}{}
		track, err := scan.ReadTrack(repo.LocalPath, rel)
		if err != nil {
			p.addError(repo.ID, relPath, ReasonInvalidTrack)
			continue
		}
		if track.Truncated {
			p.addError(repo.ID, relPath, string(scan.ReasonTruncated))
		}
		items.addTrack(track)
	}
	return items
}

// cardInputs gathers the per-pass inputs of a plan's card facts. task is nil
// for a plan whose task does not exist yet.
func (p *pass) cardInputs(ctx context.Context, e planEntry, task *taskmodels.Task) (cardInputs, error) {
	in := cardInputs{
		Done: e.items.Done, Total: e.items.Total, Newest: e.items.Newest,
		Now: p.now, StaleAfter: time.Duration(p.cfg.StaleAfterDays) * hoursPerDay * time.Hour,
		Uncommitted: p.uncommitted(e, task),
	}
	if task != nil && staleByTime(e.file, in) {
		busy, err := p.turnInFlight(ctx, task.ID)
		if err != nil {
			return in, err
		}
		in.TurnRunning = busy
	}
	return in, nil
}

// uncommitted reports that the plan file is modified, staged, or untracked. A
// repository whose git state could not be read this pass keeps the flag its
// task already carries; a repository that is not a working tree never has it.
func (p *pass) uncommitted(e planEntry, task *taskmodels.Task) bool {
	if _, failed := p.dirtyFailed[e.repo.ID]; failed {
		return task != nil && hasCardFlag(task.Metadata, flagUncommitted)
	}
	_, dirty := p.dirty[e.repo.ID][e.relPath]
	return dirty
}

// hasCardFlag reports that the task metadata lists the flag, whether the list
// holds strings in memory or values decoded from JSON.
func hasCardFlag(metadata map[string]any, flag string) bool {
	card, _ := metadata[cardDisplayKey].(map[string]any)
	switch flags := card["flags"].(type) {
	case []string:
		for _, f := range flags {
			if f == flag {
				return true
			}
		}
	case []any:
		for _, f := range flags {
			if s, ok := f.(string); ok && s == flag {
				return true
			}
		}
	}
	return false
}
