package orchestrator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kandev/kandev/internal/orchestrator/executor"
	"github.com/kandev/kandev/internal/task/models"
	v1 "github.com/kandev/kandev/pkg/api/v1"
)

// @covers AC-TASKS-TASK-OPEN-WITHOUT-AGENT-001.1, 001.2, 001.3, 001.4
func TestEnsureSession_NoAgentProfile(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepo(t)
	now := time.Now().UTC()
	agentMgr := &mockAgentManager{
		launchAgentFunc: func(_ context.Context, _ *executor.LaunchAgentRequest) (*executor.LaunchAgentResponse, error) {
			return &executor.LaunchAgentResponse{AgentExecutionID: "exec-1"}, nil
		},
	}
	taskRepo := newMockTaskRepo()
	svc := createTestServiceWithScheduler(repo, newMockStepGetter(), taskRepo, agentMgr)

	if err := repo.CreateWorkspace(ctx, &models.Workspace{ID: "ws1", Name: "Test", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
	if err := repo.CreateWorkflow(ctx, &models.Workflow{ID: "wf1", WorkspaceID: "ws1", Name: "Test Workflow", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("seed workflow: %v", err)
	}
	seedTask := func(taskID string, metadata map[string]any) {
		if err := repo.CreateTask(ctx, &models.Task{ID: taskID, WorkflowID: "wf1", WorkspaceID: "ws1", Title: "T", Metadata: metadata, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatalf("seed task %s: %v", taskID, err)
		}
		taskRepo.tasks[taskID] = &v1.Task{ID: taskID, WorkspaceID: "ws1", Metadata: metadata}
	}
	sessionCount := func(taskID string) int {
		sessions, err := repo.ListTaskSessions(ctx, taskID)
		if err != nil {
			t.Fatalf("list sessions: %v", err)
		}
		return len(sessions)
	}

	t.Run("passive open reports no_agent_profile and creates nothing", func(t *testing.T) {
		seedTask("task-unassigned", map[string]any{})
		resp, err := svc.EnsureSession(ctx, "task-unassigned", EnsureSessionOptions{
			ActivationSource: LaunchActivationSourceSessionOpen,
		})
		if err != nil {
			t.Fatalf("ensure session: %v", err)
		}
		if !resp.Success || resp.Source != "no_agent_profile" {
			t.Fatalf("got success=%v source=%q, want success no_agent_profile", resp.Success, resp.Source)
		}
		if resp.SessionID != "" || resp.NewlyCreated {
			t.Fatalf("got session_id=%q newly_created=%v, want none", resp.SessionID, resp.NewlyCreated)
		}
		if n := sessionCount("task-unassigned"); n != 0 {
			t.Fatalf("sessions = %d, want 0", n)
		}
	})

	t.Run("explicit user action still fails without a profile", func(t *testing.T) {
		seedTask("task-unassigned-explicit", map[string]any{})
		_, err := svc.EnsureSession(ctx, "task-unassigned-explicit", EnsureSessionOptions{
			ActivationSource: LaunchActivationSourceUserAction,
		})
		if !errors.Is(err, executor.ErrNoAgentProfileID) {
			t.Fatalf("err = %v, want ErrNoAgentProfileID", err)
		}
	})

	t.Run("control: passive open with a resolved profile creates a session", func(t *testing.T) {
		seedTask("task-assigned", map[string]any{"agent_profile_id": "profile1"})
		resp, err := svc.EnsureSession(ctx, "task-assigned", EnsureSessionOptions{
			ActivationSource: LaunchActivationSourceSessionOpen,
		})
		if err != nil {
			t.Fatalf("ensure session: %v", err)
		}
		if resp.SessionID == "" || resp.Source == "no_agent_profile" {
			t.Fatalf("got session_id=%q source=%q, want a created session", resp.SessionID, resp.Source)
		}
		if n := sessionCount("task-assigned"); n != 1 {
			t.Fatalf("sessions = %d, want 1", n)
		}
	})
}
