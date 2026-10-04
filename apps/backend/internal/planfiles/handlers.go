package planfiles

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/kandev/kandev/internal/common/logger"
	"github.com/kandev/kandev/internal/task/repository/repoerrors"
)

const (
	errKey            = "error"
	msgWorkspaceNone  = "workspace not found"
	msgConfigNone     = "plan files config not found"
	msgInvalidPayload = "invalid payload"
)

// workspaceDenied reports whether err is the per-user workspace access denial
// surfaced by the workspace authorizer. Handlers map it to 404 so a workspace
// the caller may not access is indistinguishable from a missing one.
func workspaceDenied(err error) bool {
	return errors.Is(err, repoerrors.ErrWorkspaceNotFound)
}

// Controller holds the HTTP handlers for plan files.
type Controller struct {
	service *Service
	logger  *logger.Logger
}

// RegisterRoutes wires the plan-file HTTP endpoints. Every route takes the
// workspace as the workspace_id query parameter, except the task decision,
// which takes it from the task's row.
func RegisterRoutes(router *gin.Engine, svc *Service, log *logger.Logger) {
	ctrl := &Controller{service: svc, logger: log}
	api := router.Group("/api/v1/plan-files")
	api.GET("/config", ctrl.httpGetConfig)
	api.PUT("/config", ctrl.httpPutConfig)
	api.POST("/board", ctrl.httpCreateBoard)
	api.GET("/unadapted", ctrl.httpUnadapted)
	ctrl.registerSyncRoutes(api)
	ctrl.registerDecisionRoutes(api)
	ctrl.registerGitRoutes(api)
}

// registerSyncRoutes registers POST /sync, the "Sync now" action.
func (c *Controller) registerSyncRoutes(api *gin.RouterGroup) {
	api.POST("/sync", c.httpSync)
}

// httpSync runs one pass now. A pass or write already holding the workspace
// lock answers 409; the caller retries or waits for the next poll.
func (c *Controller) httpSync(ctx *gin.Context) {
	workspaceID, ok := c.requireWorkspaceID(ctx)
	if !ok {
		return
	}
	summary, err := c.service.SyncNow(ctx.Request.Context(), workspaceID)
	switch {
	case errors.Is(err, ErrNotConfigured):
		ctx.JSON(http.StatusNotFound, gin.H{errKey: msgConfigNone})
	case errors.Is(err, ErrPassRunning):
		ctx.JSON(http.StatusConflict, gin.H{errKey: err.Error()})
	case err != nil:
		c.failure(ctx, "failed to run plan files sync", err)
	default:
		ctx.JSON(http.StatusOK, summary)
	}
}

func (c *Controller) requireWorkspaceID(ctx *gin.Context) (string, bool) {
	id := strings.TrimSpace(ctx.Query("workspace_id"))
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{errKey: "workspace_id is required"})
		return "", false
	}
	return id, true
}

// failure answers a service error: a denied workspace is a 404, any other
// error is logged and returned as a generic 500 so driver and query details
// never reach clients.
func (c *Controller) failure(ctx *gin.Context, msg string, err error) {
	if workspaceDenied(err) {
		ctx.JSON(http.StatusNotFound, gin.H{errKey: msgWorkspaceNone})
		return
	}
	c.logger.Error(msg, zap.Error(err))
	ctx.JSON(http.StatusInternalServerError, gin.H{errKey: msg})
}

func (c *Controller) httpGetConfig(ctx *gin.Context) {
	workspaceID, ok := c.requireWorkspaceID(ctx)
	if !ok {
		return
	}
	cfg, err := c.service.GetConfig(ctx.Request.Context(), workspaceID)
	if err != nil {
		c.failure(ctx, "failed to load plan files config", err)
		return
	}
	if cfg == nil {
		ctx.JSON(http.StatusNotFound, gin.H{errKey: msgConfigNone})
		return
	}
	ctx.JSON(http.StatusOK, cfg)
}

func (c *Controller) httpPutConfig(ctx *gin.Context) {
	workspaceID, ok := c.requireWorkspaceID(ctx)
	if !ok {
		return
	}
	var req PutConfigRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{errKey: msgInvalidPayload})
		return
	}
	cfg, err := c.service.PutConfig(ctx.Request.Context(), workspaceID, &req)
	if errors.Is(err, ErrInvalidConfig) {
		body := gin.H{errKey: err.Error()}
		var cfgErr *ConfigError
		if errors.As(err, &cfgErr) {
			body["code"] = cfgErr.Code
		}
		ctx.JSON(http.StatusBadRequest, body)
		return
	}
	if err != nil {
		c.failure(ctx, "failed to save plan files config", err)
		return
	}
	ctx.JSON(http.StatusOK, cfg)
}

func (c *Controller) httpCreateBoard(ctx *gin.Context) {
	workspaceID, ok := c.requireWorkspaceID(ctx)
	if !ok {
		return
	}
	result, err := c.service.CreateBoard(ctx.Request.Context(), workspaceID)
	if err != nil {
		c.failure(ctx, "failed to create plans board", err)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

// httpUnadapted lists local repositories with Markdown plan files that have no
// board key. It needs no sync config and stores nothing.
func (c *Controller) httpUnadapted(ctx *gin.Context) {
	workspaceID, ok := c.requireWorkspaceID(ctx)
	if !ok {
		return
	}
	repos, err := c.service.UnadaptedCounts(
		ctx.Request.Context(), workspaceID, strings.TrimSpace(ctx.Query("repository_id")))
	if err != nil {
		c.failure(ctx, "failed to count unadapted plan files", err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"repositories": repos})
}
