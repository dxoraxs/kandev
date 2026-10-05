package handlers

import (
	"context"
	"expvar"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/kandev/kandev/internal/orchestrator"
	"github.com/kandev/kandev/internal/planfiles"
	"github.com/kandev/kandev/internal/planfiles/format"
	"github.com/kandev/kandev/internal/sysprompt"
	"github.com/kandev/kandev/internal/task/models"
	"github.com/kandev/kandev/internal/task/service"
)

// Repository maintenance kinds accepted by the maintenance endpoint.
const (
	maintenanceKindPlanAdaptation = "plan_adaptation"
)

// Typed reasons returned in the response body; the frontend maps them to
// localized copy.
const (
	maintenanceReasonNoAgentProfile     = "no_agent_profile"
	maintenanceReasonNoWorkflow         = "no_workflow"
	maintenanceReasonRepositoryNotLocal = "repository_not_local"
	maintenanceReasonKindUnavailable    = "kind_unavailable"
)

// Metric outcomes. The set is closed; no repository, workspace, or task
// identifier is ever a label.
const (
	maintenanceOutcomeCreated  = "created"
	maintenanceOutcomeExisting = "existing"
	maintenanceOutcomeRejected = "rejected"
)

const localRepositorySourceType = "local"

var maintenanceTaskTotal = expvar.NewMap("repository_maintenance_task_total")

// PlanFilesSetup is the plan-file capability the plan adaptation kind needs.
// It is nil when plan files are not wired, which makes the kind unavailable.
type PlanFilesSetup interface {
	EnsureBoard(ctx context.Context, workspaceID string) (bool, error)
	UnadaptedCounts(ctx context.Context, workspaceID, repositoryID string) ([]planfiles.UnadaptedRepo, error)
	GetConfig(ctx context.Context, workspaceID string) (*planfiles.Config, error)
}

// SetPlanFilesSetup enables the plan adaptation maintenance kind.
func (h *TaskHandlers) SetPlanFilesSetup(setup PlanFilesSetup) {
	h.planFiles = setup
}

// maintenanceInput is what a kind receives after the launcher has resolved the
// repository and its checked-out branch.
type maintenanceInput struct {
	workspaceID string
	repository  *models.Repository
	baseBranch  string
}

// maintenanceKind describes one repository maintenance action. The launcher
// owns everything else (guard, resolution, task creation, launch), so a new
// kind is one table entry.
type maintenanceKind struct {
	// available reports whether the kind's feature is enabled.
	available func(h *TaskHandlers) bool
	// prepare runs before the task is created; it may be nil.
	prepare func(ctx context.Context, h *TaskHandlers, in maintenanceInput) error
	// promptName is the embedded prompt template rendered as the task description.
	promptName string
	// variables returns the raw template values; the launcher strips system tags.
	variables func(ctx context.Context, h *TaskHandlers, in maintenanceInput) (map[string]string, error)
	// title builds the English task title.
	title func(repositoryName string) string
}

var maintenanceKinds = map[string]maintenanceKind{
	maintenanceKindPlanAdaptation: {
		available:  func(h *TaskHandlers) bool { return h.planFiles != nil },
		prepare:    preparePlanAdaptation,
		promptName: "plan-file-adaptation",
		variables:  planAdaptationVariables,
		title:      func(name string) string { return "Adapt plan files: " + name },
	},
	maintenanceKindRepositoryCleanup: {
		available:  repositoryCleanupAvailable,
		promptName: "repository-cleanup",
		variables:  repositoryCleanupVariables,
		title:      func(name string) string { return "Clean up repository: " + name },
	},
}

func preparePlanAdaptation(ctx context.Context, h *TaskHandlers, in maintenanceInput) error {
	_, err := h.planFiles.EnsureBoard(ctx, in.workspaceID)
	return err
}

func planAdaptationVariables(ctx context.Context, h *TaskHandlers, in maintenanceInput) (map[string]string, error) {
	directories, err := planAdaptationDirectories(ctx, h.planFiles, in)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"repository_name":  in.repository.Name,
		"repository_path":  in.repository.LocalPath,
		"current_branch":   in.baseBranch,
		"plan_directories": strings.Join(directories, ", "),
		"board_statuses":   boardStatusList(),
	}, nil
}

// planAdaptationDirectories lists the directories that hold unadapted files in
// the repository, falling back to the configured (or default) scan directories
// when none do.
func planAdaptationDirectories(ctx context.Context, setup PlanFilesSetup, in maintenanceInput) ([]string, error) {
	rows, err := setup.UnadaptedCounts(ctx, in.workspaceID, in.repository.ID)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.RepositoryID == in.repository.ID && len(row.Directories) > 0 {
			return row.Directories, nil
		}
	}
	cfg, err := setup.GetConfig(ctx, in.workspaceID)
	if err != nil {
		return nil, err
	}
	if cfg != nil && len(cfg.Directories) > 0 {
		return cfg.Directories, nil
	}
	return planfiles.DefaultDirectories(), nil
}

func boardStatusList() string {
	statuses := []format.BoardStatus{
		format.BoardQueued, format.BoardInProgress, format.BoardWaitingOwner,
		format.BoardWaitingExternal, format.BoardDeferred, format.BoardDone, format.BoardHidden,
	}
	names := make([]string, len(statuses))
	for i, status := range statuses {
		names[i] = string(status)
	}
	return strings.Join(names, ", ")
}

// renderMaintenancePrompt renders a kind's template with every value stripped
// of system tags, so repository-derived text cannot close the system block.
func renderMaintenancePrompt(promptName string, vars map[string]string) string {
	stripped := make(map[string]string, len(vars))
	for key, value := range vars {
		stripped[key] = sysprompt.StripTags(value)
	}
	return sysprompt.Resolve(promptName, stripped)
}

type maintenanceTaskRequest struct {
	Kind string `json:"kind"`
}

type maintenanceTaskResponse struct {
	TaskID    string `json:"task_id"`
	SessionID string `json:"session_id"`
	Existing  bool   `json:"existing"`
}

// maintenanceResult is the HTTP outcome of one launcher run.
type maintenanceResult struct {
	status  int
	body    any
	outcome string
}

func maintenanceRejection(status int, reason string) maintenanceResult {
	return maintenanceResult{
		status: status, body: gin.H{"reason": reason}, outcome: maintenanceOutcomeRejected,
	}
}

func maintenanceFailure(message string) maintenanceResult {
	return maintenanceResult{status: http.StatusInternalServerError, body: gin.H{"error": message}}
}

func (h *TaskHandlers) maintenanceLock(repositoryID string) *sync.Mutex {
	lock, _ := h.maintenanceLocks.LoadOrStore(repositoryID, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func (h *TaskHandlers) recordMaintenanceOutcome(kind, outcome string) {
	if outcome == "" {
		return
	}
	maintenanceTaskTotal.Add("kind="+kind+",outcome="+outcome, 1)
	h.logger.Info("repository_maintenance.metric.task",
		zap.String("kind", kind), zap.String("outcome", outcome))
}

// httpStartMaintenanceTask creates and starts an agent task on a repository's
// main checkout, or returns the active maintenance task of the
// repository, whatever its kind.
func (h *TaskHandlers) httpStartMaintenanceTask(c *gin.Context) {
	var body maintenanceTaskRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	kind, ok := maintenanceKinds[body.Kind]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown maintenance kind"})
		return
	}
	ctx := c.Request.Context()
	repository, err := h.service.GetRepository(ctx, c.Param("id"))
	if err != nil {
		handleNotFound(c, h.logger, err, "repository not found")
		return
	}
	if repository == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	lock := h.maintenanceLock(repository.ID)
	lock.Lock()
	defer lock.Unlock()

	result := h.runMaintenanceTask(ctx, body.Kind, kind, repository)
	h.recordMaintenanceOutcome(body.Kind, result.outcome)
	c.JSON(result.status, result.body)
}

func (h *TaskHandlers) runMaintenanceTask(
	ctx context.Context, kindName string, kind maintenanceKind, repository *models.Repository,
) maintenanceResult {
	if repository.SourceType != localRepositorySourceType || repository.LocalPath == "" {
		return maintenanceRejection(http.StatusConflict, maintenanceReasonRepositoryNotLocal)
	}
	if !kind.available(h) {
		return maintenanceRejection(http.StatusNotFound, maintenanceReasonKindUnavailable)
	}
	workspaceID := repository.WorkspaceID
	existing, sessionID, err := h.service.FindActiveMaintenanceTask(ctx, workspaceID, repository.ID)
	if err != nil {
		h.logger.Error("maintenance task guard lookup failed", zap.Error(err))
		return maintenanceFailure("failed to start maintenance task")
	}
	if existing != nil {
		return maintenanceResult{
			status:  http.StatusOK,
			body:    maintenanceTaskResponse{TaskID: existing.ID, SessionID: sessionID, Existing: true},
			outcome: maintenanceOutcomeExisting,
		}
	}
	return h.createMaintenanceTask(ctx, kindName, kind, repository)
}

func (h *TaskHandlers) createMaintenanceTask(
	ctx context.Context, kindName string, kind maintenanceKind, repository *models.Repository,
) maintenanceResult {
	workspaceID := repository.WorkspaceID
	workspace, err := h.service.GetWorkspace(ctx, workspaceID)
	if err != nil || workspace == nil {
		h.logger.Error("maintenance task workspace lookup failed", zap.Error(err))
		return maintenanceFailure("failed to start maintenance task")
	}
	if workspace.DefaultAgentProfileID == nil || *workspace.DefaultAgentProfileID == "" {
		return maintenanceRejection(http.StatusConflict, maintenanceReasonNoAgentProfile)
	}
	agentProfileID := *workspace.DefaultAgentProfileID

	workflow, err := h.maintenanceWorkflow(ctx, workspaceID)
	if err != nil {
		h.logger.Error("maintenance task workflow lookup failed", zap.Error(err))
		return maintenanceFailure("failed to start maintenance task")
	}
	if workflow == nil {
		return maintenanceRejection(http.StatusConflict, maintenanceReasonNoWorkflow)
	}

	baseBranch, err := h.service.RepositoryCurrentBranch(ctx, repository.ID)
	if err != nil {
		h.logger.Error("maintenance task branch lookup failed", zap.Error(err))
		return maintenanceFailure("failed to start maintenance task")
	}
	if baseBranch == "" {
		baseBranch = repository.DefaultBranch
	}
	in := maintenanceInput{workspaceID: workspaceID, repository: repository, baseBranch: baseBranch}
	prompt, err := h.prepareMaintenancePrompt(ctx, kind, in)
	if err != nil {
		h.logger.Error("maintenance task preparation failed", zap.String("kind", kindName), zap.Error(err))
		return maintenanceFailure("failed to start maintenance task")
	}
	return h.launchMaintenanceTask(ctx, maintenanceLaunch{
		kindName: kindName, title: kind.title(repository.Name), prompt: prompt,
		input: in, workflowID: workflow.ID, agentProfileID: agentProfileID,
	})
}

// maintenanceWorkflow resolves the first visible workflow other than the plan
// board. The board is only known when plan files are wired.
func (h *TaskHandlers) maintenanceWorkflow(ctx context.Context, workspaceID string) (*models.Workflow, error) {
	boardID := ""
	if h.planFiles != nil {
		cfg, err := h.planFiles.GetConfig(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		if cfg != nil {
			boardID = cfg.WorkflowID
		}
	}
	return h.service.FirstMaintenanceWorkflow(ctx, workspaceID, boardID)
}

func (h *TaskHandlers) prepareMaintenancePrompt(
	ctx context.Context, kind maintenanceKind, in maintenanceInput,
) (string, error) {
	if kind.prepare != nil {
		if err := kind.prepare(ctx, h, in); err != nil {
			return "", err
		}
	}
	vars, err := kind.variables(ctx, h, in)
	if err != nil {
		return "", err
	}
	return renderMaintenancePrompt(kind.promptName, vars), nil
}

type maintenanceLaunch struct {
	kindName       string
	title          string
	prompt         string
	input          maintenanceInput
	workflowID     string
	agentProfileID string
}

// launchMaintenanceTask creates the task on the repository's main checkout with
// the local executor and starts the agent. The workflow step is resolved by
// task creation for an immediate agent start, exactly as for any other task
// started from the UI, and the rendered prompt is the first message.
func (h *TaskHandlers) launchMaintenanceTask(ctx context.Context, launch maintenanceLaunch) maintenanceResult {
	result, err := h.service.CreateTask(ctx, &service.CreateTaskRequest{
		WorkspaceID: launch.input.workspaceID,
		WorkflowID:  launch.workflowID,
		Title:       launch.title,
		Description: launch.prompt,
		Repositories: []service.TaskRepositoryInput{{
			RepositoryID: launch.input.repository.ID,
			BaseBranch:   launch.input.baseBranch,
		}},
		ExecutorID: models.ExecutorIDLocal,
		StartAgent: true,
		Metadata: map[string]interface{}{
			service.MetaKeyMaintenanceKind:         launch.kindName,
			service.MetaKeyMaintenanceRepositoryID: launch.input.repository.ID,
			models.MetaKeyAgentProfileID:           launch.agentProfileID,
			models.MetaKeyExecutorID:               models.ExecutorIDLocal,
		},
	})
	if err != nil {
		h.logger.Error("failed to create maintenance task", zap.Error(err))
		return maintenanceFailure("failed to create maintenance task")
	}
	task := result.Task
	resp, err := h.orchestrator.LaunchSession(ctx, &orchestrator.LaunchSessionRequest{
		TaskID:          task.ID,
		Intent:          orchestrator.IntentStart,
		AgentProfileID:  launch.agentProfileID,
		ProfileExplicit: true,
		ExecutorID:      models.ExecutorIDLocal,
		Prompt:          launch.prompt,
		WorkflowStepID:  task.WorkflowStepID,
	})
	if err != nil {
		h.logger.Error("failed to start maintenance session", zap.Error(err), zap.String("task_id", task.ID))
		return maintenanceResult{
			status: http.StatusInternalServerError,
			body:   gin.H{"error": "failed to start session", "task_id": task.ID},
		}
	}
	return maintenanceResult{
		status:  http.StatusCreated,
		body:    maintenanceTaskResponse{TaskID: task.ID, SessionID: resp.SessionID},
		outcome: maintenanceOutcomeCreated,
	}
}
