package planfiles

import (
	"context"
	"errors"
	"fmt"
	"path"
	"time"

	"go.uber.org/zap"

	"github.com/kandev/kandev/internal/planfiles/format"
	"github.com/kandev/kandev/internal/planfiles/scan"
)

// wakeNotifyTimeout bounds one delivery of a wake-up notification.
const wakeNotifyTimeout = 30 * time.Second

// DateNotifier delivers the notification of a plan whose date arrived.
// Satisfied by the notification service.
type DateNotifier interface {
	NotifyPlanDateReached(ctx context.Context, workspaceID, title, relPath string) error
}

// SetDateNotifier installs the notifier of date wake-ups. A nil notifier
// leaves wake-ups silent.
func (s *Service) SetDateNotifier(n DateNotifier) {
	s.dateNotifier = n
}

// wakeNotice is one written wake-up waiting to be announced.
type wakeNotice struct {
	workspaceID, title, relPath string
}

// dayOf is the calendar day of t in the server's local time zone, as the
// `date` key and owner notes spell it.
func dayOf(t time.Time) string {
	return t.In(time.Local).Format(noteDateLayout)
}

// wakesOn reports that a plan waiting for a date has reached it. Dates are
// valid YYYY-MM-DD values, so comparing them as text compares the days.
func wakesOn(pf format.PlanFile, today string) bool {
	if pf.Board != format.BoardWaitingExternal && pf.Board != format.BoardDeferred {
		return false
	}
	return pf.Date != "" && pf.Date <= today
}

// wakeDue returns every waiting plan whose date arrived to the owner: it
// writes `board: waiting_owner` and an owner note, and replaces the plan's
// parsed file so the rest of the pass moves the task, projects the card, and
// lists the plan under the new status. A write that cannot be made is a file
// error and is retried by the next pass.
func (p *pass) wakeDue(ctx context.Context, plans []tracked) {
	if !p.cfg.WakeOnDate {
		return
	}
	today := dayOf(p.now)
	for i := range plans {
		tr := &plans[i]
		if !wakesOn(tr.entry.file, today) {
			continue
		}
		// A board edit nobody wrote back yet is settled first; the file would
		// otherwise read as changed under it.
		if tr.row != nil && boardDiverged(p.cfg, tr.row, tr.task) {
			continue
		}
		p.wake(tr, today)
	}
}

func (p *pass) wake(tr *tracked, today string) {
	e := tr.entry
	updated, err := p.writeWake(e, today)
	if err != nil {
		p.counts.Failed++
		p.addError(e.repo.ID, e.relPath, ReasonWakeFailed)
		p.svc.logger.Warn("plan file wake-up failed",
			zap.String("repository_id", e.repo.ID), zap.String("rel_path", e.relPath), zap.Error(err))
		return
	}
	incWake(p.svc.logger)
	p.markDirty(e)
	tr.entry.file = updated
	p.svc.queueWake(wakeNotice{workspaceID: p.cfg.WorkspaceID, title: projectTitle(updated), relPath: e.relPath})
}

// writeWake rewrites the file of e once, guarded by a compare-and-swap on the
// hash the pass read, and returns the parsed result.
func (p *pass) writeWake(e planEntry, today string) (format.PlanFile, error) {
	root := p.roots[e.repo.ID]
	if root == "" {
		return format.PlanFile{}, fmt.Errorf("repository %s has no local path", e.repo.ID)
	}
	current, err := scan.ReadFile(root, e.relPath)
	if err != nil {
		return format.PlanFile{}, err
	}
	if format.ContentHash(current.Content) != e.file.Hash {
		return format.PlanFile{}, scan.ErrHashMismatch
	}
	note := "- " + today + " date reached: was " + string(e.file.Board)
	out, err := composeEdit(current.Content, map[string]string{keyBoard: string(format.BoardWaitingOwner)},
		notesHeading(p.cfg), note)
	if err != nil {
		return format.PlanFile{}, fmt.Errorf("compose wake-up: %w", err)
	}
	if err := scan.WriteFile(root, e.relPath, out, e.file.Hash, format.ContentHash); err != nil {
		return format.PlanFile{}, err
	}
	updated, ok := format.Parse(path.Base(e.relPath), out)
	if !ok {
		return format.PlanFile{}, errors.New("the woken file is no longer a plan file")
	}
	return updated, nil
}

// markDirty records that the pass changed a plan file of a git working tree,
// so the card shows it as uncommitted in the same pass.
func (p *pass) markDirty(e planEntry) {
	set, tracked := p.dirty[e.repo.ID]
	if !tracked {
		return
	}
	if set == nil {
		set = map[string]struct{}{}
		p.dirty[e.repo.ID] = set
	}
	set[e.relPath] = struct{}{}
}

// queueWake keeps a written wake-up until the workspace lock is released.
func (s *Service) queueWake(n wakeNotice) {
	s.wakeMu.Lock()
	defer s.wakeMu.Unlock()
	s.pendingWakes = append(s.pendingWakes, n)
}

// flushWakeNotices announces the queued wake-ups. It runs after the workspace
// lock is released, so a slow provider never holds up board writes. A failed
// delivery is logged and never undoes the wake-up.
func (s *Service) flushWakeNotices() {
	s.wakeMu.Lock()
	pending := s.pendingWakes
	s.pendingWakes = nil
	s.wakeMu.Unlock()
	if s.dateNotifier == nil {
		return
	}
	for _, n := range pending {
		ctx, cancel := context.WithTimeout(context.Background(), wakeNotifyTimeout)
		err := s.dateNotifier.NotifyPlanDateReached(ctx, n.workspaceID, n.title, n.relPath)
		cancel()
		if err != nil {
			s.logger.Warn("plan date notification failed",
				zap.String("workspace_id", n.workspaceID), zap.String("rel_path", n.relPath), zap.Error(err))
		}
	}
}
