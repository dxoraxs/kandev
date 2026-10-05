package service

import (
	"context"
	"testing"
	"time"

	"github.com/kandev/kandev/internal/task/models"
)

type maintenanceFixture struct {
	svc  *Service
	repo interface {
		CreateTask(ctx context.Context, task *models.Task) error
		CreateTaskSession(ctx context.Context, session *models.TaskSession) error
		CreateTaskRepository(ctx context.Context, taskRepo *models.TaskRepository) error
	}
}

func newMaintenanceFixture(t *testing.T) *maintenanceFixture {
	t.Helper()
	svc, _, repo := createTestService(t)
	ctx := context.Background()
	if err := repo.CreateWorkspace(ctx, &models.Workspace{ID: "ws-1", Name: "Workspace"}); err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}
	for _, id := range []string{"repo-1", "repo-2"} {
		if err := repo.CreateRepository(ctx, &models.Repository{
			ID: id, WorkspaceID: "ws-1", Name: id, SourceType: "local", LocalPath: "/tmp/" + id,
		}); err != nil {
			t.Fatalf("CreateRepository: %v", err)
		}
	}
	return &maintenanceFixture{svc: svc, repo: repo}
}

func (f *maintenanceFixture) workflow(t *testing.T, id string, sortOrder int, hidden bool) {
	t.Helper()
	repo, ok := f.repo.(interface {
		CreateWorkflow(ctx context.Context, wf *models.Workflow) error
	})
	if !ok {
		t.Fatal("repository cannot create workflows")
	}
	if err := repo.CreateWorkflow(context.Background(), &models.Workflow{
		ID: id, WorkspaceID: "ws-1", Name: id, SortOrder: sortOrder, Hidden: hidden,
	}); err != nil {
		t.Fatalf("CreateWorkflow %s: %v", id, err)
	}
}

func (f *maintenanceFixture) task(
	t *testing.T, id, repoID, kind string, state models.TaskSessionState, withSession bool,
) {
	t.Helper()
	ctx := context.Background()
	meta := map[string]interface{}{}
	if kind != "" {
		meta[MetaKeyMaintenanceKind] = kind
		meta[MetaKeyMaintenanceRepositoryID] = repoID
	}
	if err := f.repo.CreateTask(ctx, &models.Task{
		ID: id, WorkspaceID: "ws-1", WorkflowID: "wf-1", WorkflowStepID: "step-1",
		Title: id, Priority: "medium", Metadata: meta,
	}); err != nil {
		t.Fatalf("CreateTask %s: %v", id, err)
	}
	if err := f.repo.CreateTaskRepository(ctx, &models.TaskRepository{
		ID: "tr-" + id, TaskID: id, RepositoryID: repoID, BaseBranch: "main",
	}); err != nil {
		t.Fatalf("CreateTaskRepository %s: %v", id, err)
	}
	if !withSession {
		return
	}
	if err := f.repo.CreateTaskSession(ctx, &models.TaskSession{
		ID: "sess-" + id, TaskID: id, State: state, IsPrimary: true,
		StartedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateTaskSession %s: %v", id, err)
	}
}

func TestFindActiveMaintenanceTask_ReturnsMatchingActiveTask(t *testing.T) {
	f := newMaintenanceFixture(t)
	f.workflow(t, "wf-1", 0, false)
	f.task(t, "task-a", "repo-1", "plan_adaptation", models.TaskSessionStateRunning, true)

	task, sessionID, err := f.svc.FindActiveMaintenanceTask(context.Background(), "ws-1", "repo-1")
	if err != nil {
		t.Fatalf("FindActiveMaintenanceTask: %v", err)
	}
	if task == nil || task.ID != "task-a" || sessionID != "sess-task-a" {
		t.Fatalf("got task=%v session=%q, want task-a / sess-task-a", task, sessionID)
	}
}

func TestFindActiveMaintenanceTask_MatchesAnyKindButIgnoresOtherRepositoryAndPlainTasks(t *testing.T) {
	f := newMaintenanceFixture(t)
	f.workflow(t, "wf-1", 0, false)
	f.task(t, "task-repo", "repo-2", "plan_adaptation", models.TaskSessionStateRunning, true)
	f.task(t, "task-plain", "repo-1", "", models.TaskSessionStateRunning, true)

	task, _, err := f.svc.FindActiveMaintenanceTask(context.Background(), "ws-1", "repo-1")
	if err != nil {
		t.Fatalf("FindActiveMaintenanceTask: %v", err)
	}
	if task != nil {
		t.Fatalf("got %s, want no active task", task.ID)
	}
}

func TestFindActiveMaintenanceTask_ReturnsTaskOfAnyKind(t *testing.T) {
	f := newMaintenanceFixture(t)
	f.workflow(t, "wf-1", 0, false)
	f.task(t, "task-clean", "repo-1", "repository_cleanup", models.TaskSessionStateRunning, true)

	task, _, err := f.svc.FindActiveMaintenanceTask(context.Background(), "ws-1", "repo-1")
	if err != nil {
		t.Fatalf("FindActiveMaintenanceTask: %v", err)
	}
	if task == nil || task.ID != "task-clean" {
		t.Fatalf("got %v, want task-clean", task)
	}
}

func TestFindActiveMaintenanceTask_TerminalSessionIsNotActive(t *testing.T) {
	for _, state := range []models.TaskSessionState{
		models.TaskSessionStateCompleted, models.TaskSessionStateFailed, models.TaskSessionStateCancelled,
	} {
		f := newMaintenanceFixture(t)
		f.workflow(t, "wf-1", 0, false)
		f.task(t, "task-done", "repo-1", "plan_adaptation", state, true)

		task, _, err := f.svc.FindActiveMaintenanceTask(context.Background(), "ws-1", "repo-1")
		if err != nil {
			t.Fatalf("%s: FindActiveMaintenanceTask: %v", state, err)
		}
		if task != nil {
			t.Errorf("%s: got %s, want no active task", state, task.ID)
		}
	}
}

func TestFindActiveMaintenanceTask_TaskWithoutSessionIsActive(t *testing.T) {
	f := newMaintenanceFixture(t)
	f.workflow(t, "wf-1", 0, false)
	f.task(t, "task-nosession", "repo-1", "plan_adaptation", "", false)

	task, sessionID, err := f.svc.FindActiveMaintenanceTask(context.Background(), "ws-1", "repo-1")
	if err != nil {
		t.Fatalf("FindActiveMaintenanceTask: %v", err)
	}
	if task == nil || task.ID != "task-nosession" || sessionID != "" {
		t.Fatalf("got task=%v session=%q, want task-nosession with no session", task, sessionID)
	}
}

func TestFirstMaintenanceWorkflow_SkipsHiddenAndExcluded(t *testing.T) {
	f := newMaintenanceFixture(t)
	f.workflow(t, "wf-hidden", 0, true)
	f.workflow(t, "wf-board", 1, false)
	f.workflow(t, "wf-main", 2, false)
	f.workflow(t, "wf-later", 3, false)

	wf, err := f.svc.FirstMaintenanceWorkflow(context.Background(), "ws-1", "wf-board")
	if err != nil {
		t.Fatalf("FirstMaintenanceWorkflow: %v", err)
	}
	if wf == nil || wf.ID != "wf-main" {
		t.Fatalf("got %v, want wf-main", wf)
	}
}

func TestFirstMaintenanceWorkflow_NoneAvailable(t *testing.T) {
	f := newMaintenanceFixture(t)
	f.workflow(t, "wf-board", 0, false)

	wf, err := f.svc.FirstMaintenanceWorkflow(context.Background(), "ws-1", "wf-board")
	if err != nil {
		t.Fatalf("FirstMaintenanceWorkflow: %v", err)
	}
	if wf != nil {
		t.Fatalf("got %s, want nil", wf.ID)
	}
}
