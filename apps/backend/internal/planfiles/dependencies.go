package planfiles

import (
	"context"
	"fmt"
	"path"
	"strings"

	"go.uber.org/zap"
)

// depWork is a plan task whose dependencies are reconciled after every task of
// the pass has been resolved and applied.
type depWork struct {
	entry planEntry
	row   *TaskRow
}

// syncDependencies makes each plan task depend on the plan tasks its
// `depends_on` list names. The tasks it created are tracked in
// SyncedDependsOn, so a dependency a person added is never removed.
func (p *pass) syncDependencies(ctx context.Context, work []depWork) {
	for _, w := range work {
		p.syncTaskDependencies(ctx, w)
	}
}

func (p *pass) syncTaskDependencies(ctx context.Context, w depWork) {
	desired, ok := p.resolveDependencies(w)
	if !ok {
		return
	}
	old := nonNilSorted(w.row.SyncedDependsOn)
	if sameStrings(old, desired) {
		return
	}
	applied, failed := p.applyDependencyDiff(ctx, w.row.TaskID, old, desired)
	if failed {
		p.addErrorOnce(w.entry, ReasonInvalidDependency)
	}
	if sameStrings(old, applied) {
		return
	}
	next := *w.row
	next.SyncedDependsOn = applied
	next.LastSeenAt = p.now
	if err := p.svc.store.UpsertTaskRow(ctx, &next); err != nil {
		p.fail(w.entry.repo.ID, w.entry.relPath, fmt.Errorf("save dependencies: %w", err))
	}
}

// resolveDependencies maps the entries of `depends_on` to the sorted task IDs
// of plan files in the same directory. Unresolvable entries are reported and
// left out. ok is false when a named file could not be read this pass: the
// dependencies of the plan are then left as they are.
func (p *pass) resolveDependencies(w depWork) ([]string, bool) {
	e := w.entry
	ids := map[string]struct{}{}
	for _, name := range e.file.DependsOn {
		depPath := dependencyPlanPath(e.relPath, name)
		if depPath == "" {
			p.addErrorOnce(e, ReasonInvalidDependency)
			continue
		}
		if p.pathUnreadable(e.repo.ID, depPath) {
			return nil, false
		}
		taskID, found := p.pathTask[pathKey(e.repo.ID, depPath)]
		switch {
		case !found:
			p.addErrorOnce(e, ReasonUnknownDependency)
		case taskID == w.row.TaskID:
			p.addErrorOnce(e, ReasonInvalidDependency)
		default:
			ids[taskID] = struct{}{}
		}
	}
	return sortedKeys(ids), true
}

// dependencyPlanPath is the path of the plan file a `depends_on` entry names,
// or "" when the entry is not the name of a Markdown file in the plan's own
// directory.
func dependencyPlanPath(planRelPath, name string) string {
	depPath := dependencyPath(planRelPath, name)
	if depPath == "" || !strings.EqualFold(path.Ext(depPath), ".md") {
		return ""
	}
	return depPath
}

// applyDependencyDiff removes the dependencies only the old list holds and
// adds those only the desired list holds. It returns the list of dependencies
// this feature now holds: a removal or an addition the dependency service
// refused leaves that entry as it was, so the next pass retries it.
func (p *pass) applyDependencyDiff(ctx context.Context, taskID string, old, desired []string) ([]string, bool) {
	held := make(map[string]struct{}, len(old))
	for _, id := range old {
		held[id] = struct{}{}
	}
	want := make(map[string]struct{}, len(desired))
	for _, id := range desired {
		want[id] = struct{}{}
	}
	failed := false
	for _, id := range old {
		if _, keep := want[id]; keep {
			continue
		}
		if err := p.svc.tasks.RemoveDependency(ctx, taskID, id); err != nil && !isTaskGone(err) {
			p.logDependencyError("remove", taskID, id, err)
			failed = true
			continue
		}
		delete(held, id)
	}
	for _, id := range desired {
		if _, have := held[id]; have {
			continue
		}
		if err := p.svc.tasks.AddDependency(ctx, taskID, id); err != nil {
			p.logDependencyError("add", taskID, id, err)
			failed = true
			continue
		}
		held[id] = struct{}{}
	}
	return sortedKeys(held), failed
}

func (p *pass) logDependencyError(op, taskID, dependsOnID string, err error) {
	p.svc.logger.Warn("plan file dependency "+op+" failed",
		zap.String("task_id", taskID), zap.String("depends_on_task_id", dependsOnID), zap.Error(err))
}

// addErrorOnce reports a reason for a plan file at most once per pass.
func (p *pass) addErrorOnce(e planEntry, reason string) {
	for _, fe := range p.errs {
		if fe.RepositoryID == e.repo.ID && fe.RelPath == e.relPath && fe.Reason == reason {
			return
		}
	}
	p.addError(e.repo.ID, e.relPath, reason)
}

func nonNilSorted(ids []string) []string {
	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return sortedKeys(set)
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
