package planfiles

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	taskmodels "github.com/kandev/kandev/internal/task/models"
	"github.com/kandev/kandev/internal/task/repository/repoerrors"
	taskservice "github.com/kandev/kandev/internal/task/service"
	wfmodels "github.com/kandev/kandev/internal/workflow/models"
	v1 "github.com/kandev/kandev/pkg/api/v1"
)

// movedRecord is one MoveTaskWithOptions call that reached the store.
type movedRecord struct {
	TaskID     string
	WorkflowID string
	StepID     string
	Actor      wfmodels.StepTransitionActor
}

// fakeTaskSystem mimics the documented semantics of the task service that the
// sync pass relies on: external-id idempotency that also finds archived tasks,
// UpdateTask metadata replacement, moves that refuse archived tasks and tasks
// with a starting or running session, and the whole-band reorder contract.
// Every write is appended to writes, so a test can assert that a pass wrote
// nothing.
type fakeTaskSystem struct {
	mu       sync.Mutex
	repos    []*taskmodels.Repository
	tasks    map[string]*taskmodels.Task
	sessions map[string][]*taskmodels.TaskSession
	stepWF   map[string]string
	nextID   int
	clock    time.Time

	writes     []string
	moves      []movedRecord
	creates    []*taskservice.CreateTaskRequest
	updates    []*taskservice.UpdateTaskRequest
	reorders   [][]string
	failCreate map[string]error
	listCalls  int
	// onGetTask runs at the start of every GetTask call, before any lock.
	onGetTask func(id string)

	// blockers maps a task to the tasks it depends on. depCalls logs every
	// dependency call, so a test can assert that a pass made none.
	blockers map[string]map[string]bool
	depCalls []string
}

func newFakeTaskSystem() *fakeTaskSystem {
	return &fakeTaskSystem{
		tasks:      map[string]*taskmodels.Task{},
		sessions:   map[string][]*taskmodels.TaskSession{},
		stepWF:     map[string]string{},
		failCreate: map[string]error{},
		blockers:   map[string]map[string]bool{},
		clock:      time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC),
	}
}

func (f *fakeTaskSystem) writeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.writes)
}

func (f *fakeTaskSystem) logWrite(format string, args ...any) {
	f.writes = append(f.writes, fmt.Sprintf(format, args...))
}

func (f *fakeTaskSystem) tick() time.Time {
	f.clock = f.clock.Add(time.Second)
	return f.clock
}

func cloneTask(t *taskmodels.Task) *taskmodels.Task {
	c := *t
	c.Metadata = cloneMetadata(t.Metadata)
	return &c
}

// cloneMetadata deep-copies nested maps, as a store round trip would.
func cloneMetadata(in map[string]interface{}) map[string]interface{} {
	if in == nil {
		return nil
	}
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		if nested, ok := v.(map[string]interface{}); ok {
			v = cloneMetadata(nested)
		}
		out[k] = v
	}
	return out
}

// seedTask adds a task that was created outside the plan-file feature.
func (f *fakeTaskSystem) seedTask(id, workflowID, stepID, externalID string) *taskmodels.Task {
	f.mu.Lock()
	defer f.mu.Unlock()
	task := &taskmodels.Task{
		ID: id, WorkspaceID: testWorkspace, WorkflowID: workflowID, WorkflowStepID: stepID,
		Title: id, Priority: taskmodels.TaskPriorityMedium, State: v1.TaskStateTODO,
		ExternalID: externalID, WIPAdmitted: true, Position: f.nextPositionLocked(stepID),
		CreatedAt: f.tick(),
	}
	f.tasks[id] = task
	f.stepWF[stepID] = workflowID
	return cloneTask(task)
}

func (f *fakeTaskSystem) nextPositionLocked(stepID string) int {
	next := 0
	for _, t := range f.tasks {
		if t.WorkflowStepID == stepID && t.ArchivedAt == nil && t.Position >= next {
			next = t.Position + 1
		}
	}
	return next
}

func (f *fakeTaskSystem) setSession(taskID string, state taskmodels.TaskSessionState) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sessions[taskID] = []*taskmodels.TaskSession{{ID: "sess-" + taskID, TaskID: taskID, State: state, IsPrimary: true}}
}

func (f *fakeTaskSystem) task(id string) *taskmodels.Task {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t, ok := f.tasks[id]; ok {
		return cloneTask(t)
	}
	return nil
}

func (f *fakeTaskSystem) byExternalID(ext string) *taskmodels.Task {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, t := range f.tasks {
		if t.ExternalID == ext {
			return cloneTask(t)
		}
	}
	return nil
}

func (f *fakeTaskSystem) ListRepositories(_ context.Context, _ string) ([]*taskmodels.Repository, error) {
	return f.repos, nil
}

func (f *fakeTaskSystem) GetTask(_ context.Context, id string) (*taskmodels.Task, error) {
	if f.onGetTask != nil {
		f.onGetTask(id)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tasks[id]
	if !ok {
		return nil, repoerrors.ErrTaskNotFound
	}
	return cloneTask(t), nil
}

func (f *fakeTaskSystem) GetTaskByExternalID(_ context.Context, ws, ext string) (*taskmodels.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, t := range f.tasks {
		if t.WorkspaceID == ws && t.ExternalID == ext {
			return cloneTask(t), nil
		}
	}
	return nil, repoerrors.ErrTaskNotFound
}

func (f *fakeTaskSystem) CreateTask(_ context.Context, req *taskservice.CreateTaskRequest) (taskservice.CreateTaskResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.failCreate[req.ExternalID]; err != nil {
		return taskservice.CreateTaskResult{}, err
	}
	for _, t := range f.tasks {
		if t.WorkspaceID == req.WorkspaceID && t.ExternalID == req.ExternalID {
			return taskservice.CreateTaskResult{Task: cloneTask(t), Outcome: taskservice.CreateTaskOutcomeFoundSettled}, nil
		}
	}
	f.nextID++
	id := fmt.Sprintf("task-%d", f.nextID)
	task := &taskmodels.Task{
		ID: id, WorkspaceID: req.WorkspaceID, WorkflowID: req.WorkflowID, WorkflowStepID: req.WorkflowStepID,
		Title: req.Title, Description: req.Description, Priority: req.Priority, State: v1.TaskStateTODO,
		ExternalID: req.ExternalID, WIPAdmitted: true, Position: f.nextPositionLocked(req.WorkflowStepID),
		CreatedAt: f.tick(), Metadata: cloneMetadata(req.Metadata),
	}
	f.tasks[id] = task
	f.creates = append(f.creates, req)
	f.logWrite("create:%s", req.ExternalID)
	return taskservice.CreateTaskResult{Task: cloneTask(task), Outcome: taskservice.CreateTaskOutcomeCreated}, nil
}

func (f *fakeTaskSystem) SettleExternalID(_ context.Context, taskID, _ string) (bool, *taskmodels.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.logWrite("settle:%s", taskID)
	return true, nil, nil
}

func (f *fakeTaskSystem) UpdateTask(_ context.Context, id string, req *taskservice.UpdateTaskRequest) (*taskmodels.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tasks[id]
	if !ok {
		return nil, repoerrors.ErrTaskNotFound
	}
	if req.Title != nil {
		t.Title = *req.Title
	}
	if req.Description != nil {
		t.Description = *req.Description
	}
	if req.Priority != nil {
		t.Priority = *req.Priority
	}
	if req.Metadata != nil {
		t.Metadata = cloneMetadata(req.Metadata) // replaces, as the real service does
	}
	f.updates = append(f.updates, req)
	f.logWrite("update:%s", id)
	return cloneTask(t), nil
}

func (f *fakeTaskSystem) MoveTaskWithOptions(
	_ context.Context, id, workflowID, stepID string, _ int, opts taskservice.MoveTaskOptions,
) (*taskservice.MoveTaskResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tasks[id]
	if !ok {
		return nil, repoerrors.ErrTaskNotFound
	}
	if t.ArchivedAt != nil {
		return nil, fmt.Errorf("archived tasks cannot be moved")
	}
	for _, s := range f.sessions[id] {
		if s.State == taskmodels.TaskSessionStateStarting || s.State == taskmodels.TaskSessionStateRunning {
			return nil, fmt.Errorf("task has an active session (%s)", s.State)
		}
	}
	t.WorkflowID, t.WorkflowStepID = workflowID, stepID
	t.Position = f.nextPositionLocked(stepID)
	f.stepWF[stepID] = workflowID
	f.moves = append(f.moves, movedRecord{TaskID: id, WorkflowID: workflowID, StepID: stepID, Actor: opts.StepHistoryActor})
	f.logWrite("move:%s->%s", id, stepID)
	return &taskservice.MoveTaskResult{Task: cloneTask(t), Transitioned: true}, nil
}

func (f *fakeTaskSystem) ReorderStepTasks(
	_ context.Context, stepID, band string, ordered []string,
) (*taskservice.ReorderStepTasksResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if band != "admitted" || len(ordered) == 0 {
		return nil, repoerrors.ErrInvalidReorder
	}
	var admitted []*taskmodels.Task
	for _, t := range f.tasks {
		if t.WorkflowStepID == stepID && t.ArchivedAt == nil && (t.WIPAdmitted || t.QueuedForStepID != stepID) {
			admitted = append(admitted, t)
		}
	}
	if len(admitted) != len(ordered) {
		return nil, repoerrors.ErrStepChanged
	}
	for i, id := range ordered {
		t, ok := f.tasks[id]
		if !ok || t.WorkflowStepID != stepID || t.ArchivedAt != nil {
			return nil, repoerrors.ErrStepChanged
		}
		t.Position = i
	}
	f.reorders = append(f.reorders, append([]string(nil), ordered...))
	f.logWrite("reorder:%s:%s", stepID, strings.Join(ordered, ","))
	return &taskservice.ReorderStepTasksResult{WorkflowStepID: stepID}, nil
}

func (f *fakeTaskSystem) ListTasks(_ context.Context, workflowID string) ([]*taskmodels.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listCalls++
	var out []*taskmodels.Task
	for _, t := range f.tasks {
		if t.WorkflowID == workflowID && t.ArchivedAt == nil {
			out = append(out, cloneTask(t))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *fakeTaskSystem) ListTaskSessions(_ context.Context, taskID string) ([]*taskmodels.TaskSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*taskmodels.TaskSession(nil), f.sessions[taskID]...), nil
}

func (f *fakeTaskSystem) ArchiveTaskTree(_ context.Context, id string, cascade bool) (*taskservice.CascadeOutcome, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tasks[id]
	if !ok {
		return nil, repoerrors.ErrTaskNotFound
	}
	now := f.tick()
	t.ArchivedAt = &now
	f.logWrite("archive:%s cascade=%v", id, cascade)
	return &taskservice.CascadeOutcome{}, nil
}

func (f *fakeTaskSystem) UnarchiveTaskTree(_ context.Context, id string) (*taskservice.CascadeOutcome, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tasks[id]
	if !ok {
		return nil, repoerrors.ErrTaskNotFound
	}
	t.ArchivedAt = nil
	f.logWrite("unarchive:%s", id)
	return &taskservice.CascadeOutcome{}, nil
}

func (f *fakeTaskSystem) byExternalIDLocked(ext string) string {
	for id, t := range f.tasks {
		if t.ExternalID == ext {
			return id
		}
	}
	return ""
}

func updatePriority(priority string) *taskservice.UpdateTaskRequest {
	return &taskservice.UpdateTaskRequest{Priority: &priority}
}

// AddDependency mimics the task service: self-edges, unknown tasks, and cycles
// are rejected; an existing edge is a successful replay.
func (f *fakeTaskSystem) AddDependency(_ context.Context, taskID, dependsOnID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.depCalls = append(f.depCalls, "add:"+taskID+"->"+dependsOnID)
	if taskID == dependsOnID {
		return fmt.Errorf("a task cannot depend on itself")
	}
	if f.tasks[taskID] == nil || f.tasks[dependsOnID] == nil {
		return repoerrors.ErrTaskNotFound
	}
	if f.reachesLocked(dependsOnID, taskID) {
		return fmt.Errorf("dependency cycle")
	}
	if f.blockers[taskID] == nil {
		f.blockers[taskID] = map[string]bool{}
	}
	f.blockers[taskID][dependsOnID] = true
	f.logWrite("dep-add:%s->%s", taskID, dependsOnID)
	return nil
}

// RemoveDependency removes an edge; an absent edge is a no-op and an unknown
// task is not found.
func (f *fakeTaskSystem) RemoveDependency(_ context.Context, taskID, dependsOnID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.depCalls = append(f.depCalls, "remove:"+taskID+"->"+dependsOnID)
	if f.tasks[taskID] == nil || f.tasks[dependsOnID] == nil {
		return repoerrors.ErrTaskNotFound
	}
	delete(f.blockers[taskID], dependsOnID)
	f.logWrite("dep-remove:%s->%s", taskID, dependsOnID)
	return nil
}

// reachesLocked reports whether the target is reachable from the start task along dependency edges.
func (f *fakeTaskSystem) reachesLocked(from, to string) bool {
	seen := map[string]bool{}
	stack := []string{from}
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if id == to {
			return true
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		for next := range f.blockers[id] {
			stack = append(stack, next)
		}
	}
	return false
}

// seedDependency adds an edge the way a person would, bypassing the call log.
func (f *fakeTaskSystem) seedDependency(taskID, dependsOnID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.blockers[taskID] == nil {
		f.blockers[taskID] = map[string]bool{}
	}
	f.blockers[taskID][dependsOnID] = true
}

// dependenciesOf returns the sorted IDs the task depends on.
func (f *fakeTaskSystem) dependenciesOf(taskID string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := []string{}
	for id := range f.blockers[taskID] {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (f *fakeTaskSystem) dependencyCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.depCalls)
}
