package planfiles

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	workflowcfg "github.com/kandev/kandev/config/workflows"
	"github.com/kandev/kandev/internal/planfiles/format"
	wfmodels "github.com/kandev/kandev/internal/workflow/models"
)

func loadPlansTemplate(t *testing.T) *wfmodels.WorkflowTemplate {
	t.Helper()
	templates, err := workflowcfg.LoadTemplates()
	require.NoError(t, err)
	for _, tmpl := range templates {
		if tmpl.ID == PlansTemplateID {
			return tmpl
		}
	}
	t.Fatalf("template %q not found", PlansTemplateID)
	return nil
}

// createdSteps simulates the steps the workflow service creates from a
// template: same names and positions, fresh IDs.
func createdSteps(tmpl *wfmodels.WorkflowTemplate) []*wfmodels.WorkflowStep {
	steps := make([]*wfmodels.WorkflowStep, 0, len(tmpl.Steps))
	for _, def := range tmpl.Steps {
		steps = append(steps, &wfmodels.WorkflowStep{ID: "uuid-" + def.ID, Name: def.Name, Position: def.Position})
	}
	return steps
}

// @covers AC-TASKS-PLAN-FILES-005.2
func TestTemplate_MapsEveryVisibleStatusExactlyOnce(t *testing.T) {
	tmpl := loadPlansTemplate(t)
	seen := map[format.BoardStatus]int{}
	for _, def := range tmpl.Steps {
		if status, ok := StatusForTemplateStep(def.ID); ok {
			seen[status]++
		}
	}
	for _, status := range VisibleStatuses() {
		assert.Equal(t, 1, seen[status], "status %q must map to exactly one template step", status)
	}
	assert.Len(t, seen, len(VisibleStatuses()))
	assert.NotContains(t, seen, format.BoardHidden)
}

func TestTemplate_HandoffStepHasNoStatus(t *testing.T) {
	_, ok := StatusForTemplateStep("hand-to-agent")
	assert.False(t, ok, "the handoff step is deliberately unmapped")
}

// @covers AC-TASKS-PLAN-FILES-005.2
func TestStatusStepsFromTemplate_MapsCreatedStepsByPosition(t *testing.T) {
	tmpl := loadPlansTemplate(t)

	mapping, err := StatusStepsFromTemplate(tmpl, createdSteps(tmpl))
	require.NoError(t, err)

	assert.Len(t, mapping, len(VisibleStatuses()))
	assert.Equal(t, "uuid-queue", mapping[format.BoardQueued])
	assert.Equal(t, "uuid-in-progress", mapping[format.BoardInProgress])
	assert.Equal(t, "uuid-waiting-owner", mapping[format.BoardWaitingOwner])
	assert.Equal(t, "uuid-waiting-external", mapping[format.BoardWaitingExternal])
	assert.Equal(t, "uuid-deferred", mapping[format.BoardDeferred])
	assert.Equal(t, "uuid-done", mapping[format.BoardDone])
}

func TestStatusStepsFromTemplate_FailsWhenStepsAreMissing(t *testing.T) {
	tmpl := loadPlansTemplate(t)
	steps := createdSteps(tmpl)

	_, err := StatusStepsFromTemplate(tmpl, steps[:3])
	assert.Error(t, err, "a board whose steps were not all created has no complete mapping")
}
