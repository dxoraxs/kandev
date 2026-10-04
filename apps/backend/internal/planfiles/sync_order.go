package planfiles

import (
	"context"
	"errors"
	"sort"

	"go.uber.org/zap"

	taskmodels "github.com/kandev/kandev/internal/task/models"
	"github.com/kandev/kandev/internal/task/repository/repoerrors"
)

const admittedBand = "admitted"

// archiveRow archives the task of a plan file that is gone or hidden. A nil
// row means the file never had a task.
func (p *pass) archiveRow(ctx context.Context, row *TaskRow) {
	if row == nil {
		return
	}
	if err := p.archiveTask(ctx, row.TaskID); err != nil {
		p.fail(row.RepositoryID, row.RelPath, err)
	}
}

func (p *pass) archiveTask(ctx context.Context, taskID string) error {
	task, err := p.svc.tasks.GetTask(ctx, taskID)
	switch {
	case isTaskGone(err):
		return p.svc.store.DeleteTaskRow(ctx, taskID)
	case err != nil:
		return err
	case task.ArchivedAt != nil:
		return nil
	}
	if busy, err := p.turnInFlight(ctx, taskID); err != nil || busy {
		return err
	}
	if _, err := p.svc.archiver.ArchiveTaskTree(ctx, taskID, false); err != nil && !isPostCommit(err) {
		return err
	}
	p.counts.Archived++
	return nil
}

// archiveMissing archives the task of every row whose plan file was not found.
func (p *pass) archiveMissing(ctx context.Context) {
	for _, row := range p.rows {
		key := pathKey(row.RepositoryID, row.RelPath)
		if _, seen := p.seen[key]; seen {
			continue
		}
		if _, claimed := p.claimed[row.TaskID]; claimed || p.isProtected(row) {
			continue
		}
		p.archiveRow(ctx, row)
	}
}

// reorder makes the board order of the plan tasks of each step follow their
// desired order keys. Non-plan tasks keep their relative positions: plan tasks
// only trade places among the slots plan tasks already occupy.
func (p *pass) reorder(ctx context.Context) {
	if len(p.keys) < 2 {
		return
	}
	tasks, err := p.svc.tasks.ListTasks(ctx, p.cfg.WorkflowID)
	if err != nil {
		p.degraded = true
		p.svc.logger.Warn("plan file reorder could not list tasks", zap.Error(err))
		return
	}
	bands := admittedBands(tasks)
	for _, stepID := range sortedKeys(bands) {
		p.reorderStep(ctx, stepID, bands[stepID])
	}
}

// admittedBands groups the tasks of each step that are in the admitted band,
// each group in board order.
func admittedBands(tasks []*taskmodels.Task) map[string][]*taskmodels.Task {
	bands := map[string][]*taskmodels.Task{}
	for _, task := range tasks {
		if !task.WIPAdmitted && task.QueuedForStepID == task.WorkflowStepID {
			continue
		}
		bands[task.WorkflowStepID] = append(bands[task.WorkflowStepID], task)
	}
	for _, band := range bands {
		sort.SliceStable(band, func(i, j int) bool { return taskmodels.StepOrderLess(band[i], band[j]) })
	}
	return bands
}

func (p *pass) reorderStep(ctx context.Context, stepID string, band []*taskmodels.Task) {
	var slots []int
	var current []*taskmodels.Task
	for i, task := range band {
		if _, ok := p.keys[task.ID]; ok {
			slots = append(slots, i)
			current = append(current, task)
		}
	}
	if len(current) < 2 {
		return
	}
	desired := append([]*taskmodels.Task(nil), current...)
	sort.SliceStable(desired, func(i, j int) bool { return p.keys[desired[i].ID].less(p.keys[desired[j].ID]) })
	if sameOrder(current, desired) {
		return
	}
	if ids := taskIDs(current); p.boardOrderTrusted(ids) {
		p.writeBoardOrder(ctx, ids)
		return
	}
	ordered := make([]string, len(band))
	for i, task := range band {
		ordered[i] = task.ID
	}
	for n, slot := range slots {
		ordered[slot] = desired[n].ID
	}
	if _, err := p.svc.tasks.ReorderStepTasks(ctx, stepID, admittedBand, ordered); err != nil {
		p.degraded = true
		if !errors.Is(err, repoerrors.ErrStepChanged) {
			p.svc.logger.Warn("plan file reorder failed", zap.Error(err))
		}
	}
}

func sameOrder(a, b []*taskmodels.Task) bool {
	for i := range a {
		if a[i].ID != b[i].ID {
			return false
		}
	}
	return true
}

func taskIDs(tasks []*taskmodels.Task) []string {
	ids := make([]string, len(tasks))
	for i, task := range tasks {
		ids[i] = task.ID
	}
	return ids
}
