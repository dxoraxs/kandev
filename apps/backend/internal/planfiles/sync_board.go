package planfiles

import "context"

// writeFailedNotice is shown on a task whose board edit could not be written.
// It is task description data, not interface copy.
const writeFailedNotice = "board edit not saved because the file could not be written"

// reconcileBoardEdit settles a task that differs from the state its last sync
// applied: a board edit nobody has written back yet. While the file is
// unchanged since it was last read, the edit is written to the file and kept.
// Otherwise the file wins: the task is left to be restored and carries a
// notice. It updates tr in place.
func (p *pass) reconcileBoardEdit(ctx context.Context, tr *tracked) {
	row := tr.row
	if row == nil || !boardDiverged(p.cfg, row, tr.task) {
		return
	}
	if row.ContentHash != tr.entry.file.Hash {
		tr.notice = conflictNotice
		return
	}
	root := p.roots[row.RepositoryID]
	if root == "" {
		return
	}
	res := p.svc.writeBoardEdit(ctx, p.cfg, root, row, tr.task)
	tr.row = res.row
	switch res.outcome {
	case writebackWritten:
		tr.entry.file, tr.notice = res.file, ""
	case writebackConflict:
		tr.notice = conflictNotice
	case writebackFailed:
		tr.notice = writeFailedNotice
		p.counts.Failed++
		p.addError(row.RepositoryID, row.RelPath, ReasonWriteFailed)
	}
}

// boardOrderTrusted reports that the board order of the listed plan tasks can
// only come from a person's reorder: no task of the step is new, changed on
// disk, or moved, so the file orders are what the board order should be
// written to.
func (p *pass) boardOrderTrusted(current []string) bool {
	for _, id := range current {
		if _, volatile := p.volatile[id]; volatile {
			return false
		}
	}
	return true
}

// writeBoardOrder writes the board order of a step to the plan files in place
// of reordering the board.
func (p *pass) writeBoardOrder(ctx context.Context, current []string) {
	items := make([]orderItem, len(current))
	for i, id := range current {
		items[i] = orderItem{ID: id, Key: p.keys[id]}
	}
	if values := computeOrders(items); len(values) > 0 {
		p.svc.writeOrders(ctx, p.cfg.WorkspaceID, values)
	}
}
