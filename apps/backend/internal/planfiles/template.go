package planfiles

import (
	"fmt"

	"github.com/kandev/kandev/internal/planfiles/format"
	wfmodels "github.com/kandev/kandev/internal/workflow/models"
)

// PlansTemplateID is the built-in workflow template for plan boards
// (config/workflows/plans.yml).
const PlansTemplateID = "plans"

// templateStepStatus maps the Plans template's step IDs to board statuses.
// The template's hand-to-agent step is intentionally absent: it has no status
// of its own, because an agent that works a plan keeps the file's status.
var templateStepStatus = map[string]format.BoardStatus{
	"queue":            format.BoardQueued,
	"in-progress":      format.BoardInProgress,
	"waiting-owner":    format.BoardWaitingOwner,
	"waiting-external": format.BoardWaitingExternal,
	"deferred":         format.BoardDeferred,
	"done":             format.BoardDone,
}

// VisibleStatuses lists every board status that needs a step, in board order.
// The hidden status has no step: such files are archived.
func VisibleStatuses() []format.BoardStatus {
	return []format.BoardStatus{
		format.BoardQueued, format.BoardInProgress, format.BoardWaitingOwner,
		format.BoardWaitingExternal, format.BoardDeferred, format.BoardDone,
	}
}

// StatusForTemplateStep returns the board status of a Plans template step ID.
func StatusForTemplateStep(stepID string) (format.BoardStatus, bool) {
	status, ok := templateStepStatus[stepID]
	return status, ok
}

// StatusStepsFromTemplate builds the status-to-step mapping for a workflow
// created from the Plans template. The workflow service replaces template step
// IDs with fresh IDs, so created steps are matched to template steps by
// position, which the service copies unchanged.
func StatusStepsFromTemplate(
	tmpl *wfmodels.WorkflowTemplate, created []*wfmodels.WorkflowStep,
) (map[format.BoardStatus]string, error) {
	byPosition := make(map[int]string, len(created))
	for _, step := range created {
		byPosition[step.Position] = step.ID
	}
	mapping := make(map[format.BoardStatus]string, len(templateStepStatus))
	for _, def := range tmpl.Steps {
		status, ok := templateStepStatus[def.ID]
		if !ok {
			continue
		}
		stepID, found := byPosition[def.Position]
		if !found {
			return nil, fmt.Errorf("plans board has no step at position %d for %q", def.Position, def.ID)
		}
		mapping[status] = stepID
	}
	for _, status := range VisibleStatuses() {
		if mapping[status] == "" {
			return nil, fmt.Errorf("plans template does not map status %q", status)
		}
	}
	return mapping, nil
}
