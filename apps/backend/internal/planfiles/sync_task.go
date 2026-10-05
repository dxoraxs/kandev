package planfiles

import (
	"context"
	"errors"
	"fmt"

	"github.com/kandev/kandev/internal/planfiles/format"
	taskmodels "github.com/kandev/kandev/internal/task/models"
	taskservice "github.com/kandev/kandev/internal/task/service"
	wfmodels "github.com/kandev/kandev/internal/workflow/models"
)

// tracked is a visible plan file together with the task it resolved to.
type tracked struct {
	entry planEntry
	row   *TaskRow
	task  *taskmodels.Task
	// notice is the board-edit notice the task description carries.
	notice string
}

// resolveAll finds, adopts, or creates the task of every visible plan file and
// archives the task of every hidden one. It runs before any update so that
// dependency links can name tasks created in the same pass.
func (p *pass) resolveAll(ctx context.Context, entries []planEntry) []tracked {
	for _, e := range entries {
		if row := p.rowFor(e); row != nil && e.file.Board != format.BoardHidden {
			p.pathTask[e.key()] = row.TaskID
		}
	}
	result := make([]tracked, 0, len(entries))
	for _, e := range entries {
		p.seen[e.key()] = struct{}{}
		row := p.rowFor(e)
		if e.file.Board == format.BoardHidden {
			p.archiveRow(ctx, row)
			continue
		}
		task, row, err := p.resolveTask(ctx, e, row)
		if err != nil {
			p.fail(e.repo.ID, e.relPath, err)
			continue
		}
		p.claimed[task.ID] = struct{}{}
		p.pathTask[e.key()] = task.ID
		result = append(result, tracked{entry: e, row: row, task: task})
	}
	return result
}

// rowFor returns the sync row a plan file continues: the row of its path, else
// the row holding its external identifier (a renamed file that keeps its
// `external_id`).
func (p *pass) rowFor(e planEntry) *TaskRow {
	if row := p.rowsByPath[e.key()]; row != nil {
		return row
	}
	return p.rowsByExt[e.extID]
}

func (p *pass) resolveTask(ctx context.Context, e planEntry, row *TaskRow) (*taskmodels.Task, *TaskRow, error) {
	if row != nil {
		task, err := p.svc.tasks.GetTask(ctx, row.TaskID)
		switch {
		case err == nil:
			return p.checkClaim(task, row)
		case !isTaskGone(err):
			return nil, nil, err
		}
		// The task was deleted: forget its row so the file gets a new task.
		if err := p.svc.store.DeleteTaskRow(ctx, row.TaskID); err != nil {
			return nil, nil, err
		}
		row = nil
	}
	task, err := p.svc.tasks.GetTaskByExternalID(ctx, p.cfg.WorkspaceID, e.extID)
	switch {
	case err == nil:
		return p.checkClaim(task, nil)
	case !isTaskGone(err):
		return nil, nil, err
	}
	task, err = p.createTask(ctx, e)
	return task, row, err
}

func (p *pass) checkClaim(task *taskmodels.Task, row *TaskRow) (*taskmodels.Task, *TaskRow, error) {
	if _, taken := p.claimed[task.ID]; taken {
		return nil, nil, errIdentityConflict
	}
	return task, row, nil
}

func (p *pass) createTask(ctx context.Context, e planEntry) (*taskmodels.Task, error) {
	stepID, err := p.initialStep(e.file)
	if err != nil {
		return nil, err
	}
	result, err := p.svc.tasks.CreateTask(ctx, &taskservice.CreateTaskRequest{
		WorkspaceID:    p.cfg.WorkspaceID,
		WorkflowID:     p.cfg.WorkflowID,
		WorkflowStepID: stepID,
		Title:          projectTitle(e.file),
		Description:    p.describe(e, ""),
		Priority:       e.file.Priority,
		Repositories:   []taskservice.TaskRepositoryInput{{RepositoryID: e.repo.ID}},
		ExternalID:     e.extID,
	})
	if err != nil {
		return nil, err
	}
	if result.Outcome != taskservice.CreateTaskOutcomeCreated {
		return p.checkClaimOnly(result.Task)
	}
	p.counts.Created++
	settled, _, err := p.svc.tasks.SettleExternalID(ctx, result.Task.ID, e.extID)
	if err != nil {
		return nil, err
	}
	if !settled {
		return nil, errors.New("the external identifier was released while the task was being created")
	}
	return result.Task, nil
}

func (p *pass) checkClaimOnly(task *taskmodels.Task) (*taskmodels.Task, error) {
	task, _, err := p.checkClaim(task, nil)
	return task, err
}

// initialStep is the step of a new task: the one mapped to the file's status,
// or the queued step when the status value is unreadable.
func (p *pass) initialStep(pf format.PlanFile) (string, error) {
	if pf.Board == "" {
		return p.stepFor(format.BoardQueued)
	}
	return p.stepFor(pf.Board)
}

// desiredStep is where an existing task belongs. An unreadable status keeps
// the task where it is, or queues a task that is not on the plan board yet.
func (p *pass) desiredStep(pf format.PlanFile, task *taskmodels.Task) (string, error) {
	if pf.Board == "" {
		if task.WorkflowID == p.cfg.WorkflowID {
			return task.WorkflowStepID, nil
		}
		return p.stepFor(format.BoardQueued)
	}
	if p.staysInHandoff(pf, task) {
		return task.WorkflowStepID, nil
	}
	return p.stepFor(pf.Board)
}

// staysInHandoff reports that the task sits in a step of the plan board that no
// status maps to (a handoff step) while the file is still queued or in
// progress. Such a task is left there; any other status pulls it out.
func (p *pass) staysInHandoff(pf format.PlanFile, task *taskmodels.Task) bool {
	if pf.Board != format.BoardQueued && pf.Board != format.BoardInProgress {
		return false
	}
	if task.WorkflowID != p.cfg.WorkflowID {
		return false
	}
	for _, stepID := range p.cfg.StatusSteps {
		if stepID == task.WorkflowStepID {
			return false
		}
	}
	return handoffHolds(p.cfg, pf, task.WorkflowStepID)
}

// describe builds the task description of a plan file. notice is the
// board-edit notice to show.
func (p *pass) describe(e planEntry, notice string) string {
	in := descriptionInput{RepositoryName: e.repo.Name, RelPath: e.relPath, File: e.file, Notice: notice}
	for _, name := range e.file.DependsOn {
		depPath := dependencyPath(e.relPath, name)
		if depPath == "" {
			continue
		}
		if taskID, ok := p.pathTask[pathKey(e.repo.ID, depPath)]; ok {
			in.Dependencies = append(in.Dependencies, dependencyLink{Name: name, TaskID: taskID})
		}
	}
	return projectDescription(in)
}

// applyTracked brings one task in line with its file and stores the state it
// applied. It returns the stored row, or nil when the task could not be synced.
func (p *pass) applyTracked(ctx context.Context, tr tracked) *TaskRow {
	e := tr.entry
	row, err := p.applyOne(ctx, tr)
	if err != nil {
		p.fail(e.repo.ID, e.relPath, err)
		return nil
	}
	return row
}

func (p *pass) applyOne(ctx context.Context, tr tracked) (*TaskRow, error) {
	startStep := tr.task.WorkflowStepID
	volatile := p.startsVolatile(tr)
	task, err := p.unarchive(ctx, tr.task)
	if err != nil {
		return nil, err
	}
	tr.task = task
	tr.notice = carriedNotice(tr.row, tr.entry)
	p.reconcileBoardEdit(ctx, &tr)
	task = tr.task
	e := tr.entry
	stepID, err := p.desiredStep(e.file, task)
	if err != nil {
		return nil, err
	}
	task, err = p.moveIfNeeded(ctx, task, stepID)
	if err != nil {
		return nil, err
	}
	task, err = p.updateFields(ctx, e, tr.notice, task)
	if err != nil {
		return nil, err
	}
	if volatile || task.WorkflowStepID != startStep || task.WorkflowStepID != stepID {
		p.volatile[task.ID] = struct{}{}
	}
	p.keys[task.ID] = e.orderKey()
	p.finalStep[task.ID] = task.WorkflowStepID
	return p.saveRow(ctx, e, tr.row, task, tr.notice)
}

// startsVolatile reports that the position of a task on the board cannot be
// read as a person's reorder: the task is new, adopted, archived, changed on
// disk, or was moved to another step since the last pass.
func (p *pass) startsVolatile(tr tracked) bool {
	row := tr.row
	return row == nil || tr.task.ArchivedAt != nil || row.ContentHash != tr.entry.file.Hash ||
		tr.task.WorkflowStepID != row.SyncedStepID
}

// carriedNotice is the notice of the row while the file is the one it was
// recorded for. A changed file clears it.
func carriedNotice(row *TaskRow, e planEntry) string {
	if row != nil && row.ContentHash == e.file.Hash {
		return row.Notice
	}
	return ""
}

// unarchive restores an archived task: one whose file returned, or one adopted
// from the archive. The same task continues; no second task is created.
func (p *pass) unarchive(ctx context.Context, task *taskmodels.Task) (*taskmodels.Task, error) {
	if task.ArchivedAt == nil {
		return task, nil
	}
	if _, err := p.svc.archiver.UnarchiveTaskTree(ctx, task.ID); err != nil && !isPostCommit(err) {
		return nil, fmt.Errorf("unarchive task: %w", err)
	}
	restored, err := p.svc.tasks.GetTask(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	if restored.ArchivedAt != nil {
		return nil, errors.New("the task is still archived after unarchiving")
	}
	p.counts.Unarchived++
	return restored, nil
}

// isPostCommit reports housekeeping trouble after the archive state change
// already committed.
func isPostCommit(err error) bool {
	var postCommit *taskservice.CascadePostCommitError
	return errors.As(err, &postCommit)
}

func (p *pass) moveIfNeeded(ctx context.Context, task *taskmodels.Task, stepID string) (*taskmodels.Task, error) {
	if task.WorkflowID == p.cfg.WorkflowID && task.WorkflowStepID == stepID {
		return task, nil
	}
	if busy, err := p.turnInFlight(ctx, task.ID); err != nil || busy {
		return task, err
	}
	result, err := p.svc.tasks.MoveTaskWithOptions(ctx, task.ID, p.cfg.WorkflowID, stepID, 0,
		taskservice.MoveTaskOptions{StepHistoryActor: wfmodels.StepTransitionActorSystem})
	if err != nil {
		return nil, fmt.Errorf("move task: %w", err)
	}
	p.counts.Moved++
	if result != nil && result.Task != nil {
		return result.Task, nil
	}
	moved := *task
	moved.WorkflowID, moved.WorkflowStepID = p.cfg.WorkflowID, stepID
	return &moved, nil
}

// turnInFlight reports that a session of the task is starting or running an
// agent turn. Such a task is left where it is and moved by a later pass.
func (p *pass) turnInFlight(ctx context.Context, taskID string) (bool, error) {
	sessions, err := p.svc.tasks.ListTaskSessions(ctx, taskID)
	if err != nil {
		return false, fmt.Errorf("list task sessions: %w", err)
	}
	for _, session := range sessions {
		if session.State == taskmodels.TaskSessionStateStarting || session.State == taskmodels.TaskSessionStateRunning {
			return true, nil
		}
	}
	return false, nil
}

// updateFields writes the title, description, and priority a file projects.
// Metadata is never sent: UpdateTask replaces the whole map.
func (p *pass) updateFields(
	ctx context.Context, e planEntry, notice string, task *taskmodels.Task,
) (*taskmodels.Task, error) {
	title, desc, priority := projectTitle(e.file), p.describe(e, notice), e.file.Priority
	req := &taskservice.UpdateTaskRequest{}
	if task.Title != title {
		req.Title = &title
	}
	if task.Description != desc {
		req.Description = &desc
	}
	if task.Priority != priority {
		req.Priority = &priority
	}
	if req.Title == nil && req.Description == nil && req.Priority == nil {
		return task, nil
	}
	updated, err := p.svc.tasks.UpdateTask(ctx, task.ID, req)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	p.counts.Updated++
	if updated == nil {
		return task, nil
	}
	return updated, nil
}

// saveRow stores the state this pass applied, when it differs from the stored
// row. LastSeenAt therefore records the last change, not the last pass.
func (p *pass) saveRow(
	ctx context.Context, e planEntry, row *TaskRow, task *taskmodels.Task, notice string,
) (*TaskRow, error) {
	next := &TaskRow{
		TaskID: task.ID, WorkspaceID: p.cfg.WorkspaceID, RepositoryID: e.repo.ID, RelPath: e.relPath,
		ExternalID: e.extID, ContentHash: e.file.Hash, SyncedStepID: task.WorkflowStepID,
		SyncedPriority: task.Priority, SyncedOrderKey: e.orderKey().String(), Notice: notice, LastSeenAt: p.now,
	}
	if row != nil {
		next.ExternalID = row.ExternalID
		next.SyncedDependsOn = row.SyncedDependsOn
		if sameSyncState(row, next) {
			return row, nil
		}
	}
	if err := p.svc.store.UpsertTaskRow(ctx, next); err != nil {
		return nil, err
	}
	return next, nil
}

func sameSyncState(a, b *TaskRow) bool {
	return a.TaskID == b.TaskID && a.RepositoryID == b.RepositoryID && a.RelPath == b.RelPath &&
		a.ExternalID == b.ExternalID && a.ContentHash == b.ContentHash && a.SyncedStepID == b.SyncedStepID &&
		a.SyncedPriority == b.SyncedPriority && a.SyncedOrderKey == b.SyncedOrderKey && a.Notice == b.Notice
}
