package planfiles

import (
	"context"
	"path"
	"sync"

	"go.uber.org/zap"

	"github.com/kandev/kandev/internal/common/logger"
	"github.com/kandev/kandev/internal/events"
	"github.com/kandev/kandev/internal/events/bus"
	"github.com/kandev/kandev/internal/planfiles/format"
	"github.com/kandev/kandev/internal/planfiles/scan"
	taskmodels "github.com/kandev/kandev/internal/task/models"
)

// eventSubscriber is the part of the event bus the write-back needs.
type eventSubscriber interface {
	Subscribe(subject string, handler bus.EventHandler) (bus.Subscription, error)
}

// workItem names what to re-examine: a task, or the board order of one step.
// Events only locate the work; the state it acts on is always reloaded.
type workItem struct {
	step        bool
	workspaceID string
	id          string
}

// WriteBackSubscriber writes board edits (a moved, reordered, or reprioritised
// plan task) back to the plan file. Event handlers only queue work: the bus
// delivers synchronously to the publisher, and a sync pass publishes while it
// holds the workspace lock the write-back needs. One owned goroutine drains
// the queue.
type WriteBackSubscriber struct {
	svc    *Service
	events eventSubscriber
	logger *logger.Logger

	mu      sync.Mutex
	started bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	subs    []bus.Subscription

	queueMu sync.Mutex
	queue   []workItem
	queued  map[workItem]struct{}
	wake    chan struct{}
}

// NewWriteBackSubscriber creates a write-back subscriber for svc on the given
// event bus.
func NewWriteBackSubscriber(svc *Service, eventBus eventSubscriber, log *logger.Logger) *WriteBackSubscriber {
	return &WriteBackSubscriber{
		svc:    svc,
		events: eventBus,
		logger: log.WithFields(zap.String("component", "planfiles-writeback")),
		queued: map[workItem]struct{}{},
		wake:   make(chan struct{}, 1),
	}
}

// Start subscribes to the task events and launches the worker. Idempotent.
func (w *WriteBackSubscriber) Start(ctx context.Context) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.started {
		return
	}
	handlers := map[string]bus.EventHandler{
		events.TaskMoved:     w.onTaskEvent,
		events.TaskUpdated:   w.onTaskEvent,
		events.TaskReordered: w.onReorderEvent,
	}
	for subject, handler := range handlers {
		sub, err := w.events.Subscribe(subject, handler)
		if err != nil {
			w.logger.Error("plan file write-back could not subscribe", zap.String("subject", subject), zap.Error(err))
			w.unsubscribe()
			return
		}
		w.subs = append(w.subs, sub)
	}
	w.started = true
	ctx, w.cancel = context.WithCancel(ctx)
	w.wg.Add(1)
	go w.loop(ctx)
}

// Stop unsubscribes and waits for the worker to drain. Idempotent. The mutex is
// held through the wait so a concurrent Start cannot register a second worker
// mid-shutdown; the worker never takes it.
func (w *WriteBackSubscriber) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.started {
		return
	}
	w.started = false
	w.unsubscribe()
	w.cancel()
	w.wg.Wait()
}

func (w *WriteBackSubscriber) unsubscribe() {
	for _, sub := range w.subs {
		_ = sub.Unsubscribe()
	}
	w.subs = nil
}

func (w *WriteBackSubscriber) onTaskEvent(_ context.Context, event *bus.Event) error {
	if id := eventField(event, "task_id"); id != "" {
		w.enqueue(workItem{id: id})
	}
	return nil
}

func (w *WriteBackSubscriber) onReorderEvent(_ context.Context, event *bus.Event) error {
	stepID, workspaceID := eventField(event, "workflow_step_id"), eventField(event, "workspace_id")
	if stepID != "" && workspaceID != "" {
		w.enqueue(workItem{step: true, workspaceID: workspaceID, id: stepID})
	}
	return nil
}

func eventField(event *bus.Event, key string) string {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return ""
	}
	value, _ := data[key].(string)
	return value
}

func (w *WriteBackSubscriber) enqueue(item workItem) {
	w.queueMu.Lock()
	if _, pending := w.queued[item]; !pending {
		w.queued[item] = struct{}{}
		w.queue = append(w.queue, item)
	}
	w.queueMu.Unlock()
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

func (w *WriteBackSubscriber) next() (workItem, bool) {
	w.queueMu.Lock()
	defer w.queueMu.Unlock()
	if len(w.queue) == 0 {
		return workItem{}, false
	}
	item := w.queue[0]
	w.queue = w.queue[1:]
	delete(w.queued, item)
	return item, true
}

func (w *WriteBackSubscriber) loop(ctx context.Context) {
	defer w.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.wake:
		}
		for ctx.Err() == nil {
			item, ok := w.next()
			if !ok {
				break
			}
			if item.step {
				w.handleStep(ctx, item.workspaceID, item.id)
			} else {
				w.handleTask(ctx, item.id)
			}
		}
	}
}

// handleTask writes back the board edit of one task, if it has a plan row.
func (w *WriteBackSubscriber) handleTask(ctx context.Context, taskID string) {
	row, err := w.svc.store.GetTaskRow(ctx, taskID)
	if err != nil || row == nil {
		return
	}
	cfg := w.activeConfig(ctx, row.WorkspaceID)
	if cfg == nil {
		return
	}
	unlock := w.svc.LockWorkspace(row.WorkspaceID)
	defer unlock()
	w.writeTask(ctx, cfg, taskID)
}

// handleStep writes the board order of one step back to the plan files.
func (w *WriteBackSubscriber) handleStep(ctx context.Context, workspaceID, stepID string) {
	cfg := w.activeConfig(ctx, workspaceID)
	if cfg == nil {
		return
	}
	unlock := w.svc.LockWorkspace(workspaceID)
	defer unlock()
	tasks, err := w.svc.tasks.ListTasks(ctx, cfg.WorkflowID)
	if err != nil {
		w.logger.Warn("plan file order write-back could not list tasks", zap.Error(err))
		return
	}
	band := admittedBands(tasks)[stepID]
	for _, task := range band {
		w.writeTask(ctx, cfg, task.ID)
	}
	items, ok := w.stepSequence(ctx, workspaceID, band)
	if !ok {
		return
	}
	if values := computeOrders(items); len(values) > 0 {
		w.svc.writeOrders(ctx, workspaceID, values)
	}
}

// activeConfig returns the workspace's config while plan files are enabled and
// the task system is wired.
func (w *WriteBackSubscriber) activeConfig(ctx context.Context, workspaceID string) *Config {
	if w.svc.tasks == nil {
		return nil
	}
	cfg, err := w.svc.store.GetConfig(ctx, workspaceID)
	if err != nil || cfg == nil || !cfg.Enabled {
		return nil
	}
	return cfg
}

// writeTask writes back the board edit of one task. The caller holds the
// workspace lock.
func (w *WriteBackSubscriber) writeTask(ctx context.Context, cfg *Config, taskID string) {
	row, err := w.svc.store.GetTaskRow(ctx, taskID)
	if err != nil || row == nil {
		return
	}
	task, err := w.svc.tasks.GetTask(ctx, taskID)
	if err != nil || !boardDiverged(cfg, row, task) {
		return
	}
	roots, err := w.svc.repoRoots(ctx, cfg.WorkspaceID)
	if err != nil || roots[row.RepositoryID] == "" {
		return
	}
	w.svc.writeBoardEdit(ctx, cfg, roots[row.RepositoryID], row, task)
}

// stepSequence reads the order key of every plan task of a step band, in board
// order. It reports false when a file cannot be read as a plan file, because a
// partial sequence would write wrong values.
func (w *WriteBackSubscriber) stepSequence(
	ctx context.Context, workspaceID string, band []*taskmodels.Task,
) ([]orderItem, bool) {
	roots, err := w.svc.repoRoots(ctx, workspaceID)
	if err != nil {
		return nil, false
	}
	var items []orderItem
	for _, task := range band {
		row, err := w.svc.store.GetTaskRow(ctx, task.ID)
		if err != nil {
			return nil, false
		}
		if row == nil {
			continue
		}
		file, err := scan.ReadFile(roots[row.RepositoryID], row.RelPath)
		if err != nil {
			return nil, false
		}
		pf, ok := format.Parse(path.Base(row.RelPath), file.Content)
		if !ok {
			return nil, false
		}
		items = append(items, orderItem{ID: task.ID, Key: newOrderKey(pf.Order, row.RelPath, row.RepositoryID)})
	}
	return items, true
}
