package workflows

import (
	"strings"
	"testing"

	"github.com/kandev/kandev/internal/workflow/models"
)

func loadPlansTemplate(t *testing.T) *models.WorkflowTemplate {
	t.Helper()
	templates, err := LoadTemplates()
	if err != nil {
		t.Fatalf("LoadTemplates() returned error: %v", err)
	}
	for _, template := range templates {
		if template.ID == "plans" {
			return template
		}
	}
	t.Fatal("plans template not found")
	return nil
}

// @covers AC-TASKS-PLAN-FILES-005.2
func TestPlansTemplate_StepsAndHandoff(t *testing.T) {
	tmpl := loadPlansTemplate(t)
	wantIDs := []string{"queue", "hand-to-agent", "in-progress", "waiting-owner", "waiting-external", "deferred", "done"}
	if len(tmpl.Steps) != len(wantIDs) {
		t.Fatalf("plans template has %d steps, want %d", len(tmpl.Steps), len(wantIDs))
	}
	for i, step := range tmpl.Steps {
		if step.ID != wantIDs[i] || step.Position != i {
			t.Errorf("step %d = %q at position %d, want %q at %d", i, step.ID, step.Position, wantIDs[i], i)
		}
		if !step.AllowManualMove {
			t.Errorf("step %q must allow manual moves", step.ID)
		}
		if len(step.Events.OnTurnStart) != 0 || len(step.Events.OnTurnComplete) != 0 {
			t.Errorf("step %q must have no turn-driven transitions", step.ID)
		}
		wantStart := step.ID == "queue"
		if step.IsStartStep != wantStart {
			t.Errorf("step %q IsStartStep = %t, want %t", step.ID, step.IsStartStep, wantStart)
		}
		wantComplete := step.ID == "done"
		if step.CompleteTaskOnEnter != wantComplete {
			t.Errorf("step %q CompleteTaskOnEnter = %t, want %t", step.ID, step.CompleteTaskOnEnter, wantComplete)
		}
		hasAutoStart := len(step.Events.OnEnter) == 1 && step.Events.OnEnter[0].Type == models.OnEnterAutoStartAgent
		if hasAutoStart != (step.ID == "hand-to-agent") {
			t.Errorf("step %q auto_start_agent = %t, want only hand-to-agent", step.ID, hasAutoStart)
		}
	}
	handoff := tmpl.Steps[1]
	if !strings.HasSuffix(handoff.Prompt, "{{task_prompt}}") {
		t.Errorf("hand-to-agent prompt must end with {{task_prompt}}, got %q", handoff.Prompt)
	}
	if !strings.Contains(handoff.Prompt, "board") {
		t.Errorf("hand-to-agent prompt must mention the board key")
	}
}
