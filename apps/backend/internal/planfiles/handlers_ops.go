package planfiles

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// gitStatusCodes maps a git operation error code to its HTTP status.
var gitStatusCodes = map[string]int{
	CodeRepositoryBusy:     http.StatusConflict,
	CodeNothingToCommit:    http.StatusConflict,
	CodeCommitFailed:       http.StatusUnprocessableEntity,
	CodeRepositoryNotFound: http.StatusNotFound,
	CodeInvalidCommit:      http.StatusBadRequest,
}

// registerGitRoutes registers GET /git-status and POST /commit.
func (c *Controller) registerGitRoutes(api *gin.RouterGroup) {
	api.GET("/git-status", c.httpGitStatus)
	api.POST("/commit", c.httpCommit)
}

func (c *Controller) httpGitStatus(ctx *gin.Context) {
	workspaceID, ok := c.requireWorkspaceID(ctx)
	if !ok {
		return
	}
	repos, err := c.service.GitStatus(ctx.Request.Context(), workspaceID)
	if err != nil {
		c.failure(ctx, "failed to read the plan file git state", err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"repositories": repos})
}

func (c *Controller) httpCommit(ctx *gin.Context) {
	workspaceID, ok := c.requireWorkspaceID(ctx)
	if !ok {
		return
	}
	var req CommitRequest
	if err := ctx.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.RepositoryID) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{errKey: msgInvalidPayload, "code": CodeInvalidCommit})
		return
	}
	req.RepositoryID = strings.TrimSpace(req.RepositoryID)
	result, err := c.service.CommitPlans(ctx.Request.Context(), workspaceID, req)
	var gitErr *GitError
	if errors.As(err, &gitErr) {
		body := gin.H{errKey: gitErr.Message, "code": gitErr.Code}
		if gitErr.Output != "" {
			body["output"] = gitErr.Output
		}
		ctx.JSON(gitStatusCodes[gitErr.Code], body)
		return
	}
	if err != nil {
		c.failure(ctx, "failed to commit the plan files", err)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

// createStatusCodes maps a create error code to its HTTP status.
var createStatusCodes = map[string]int{
	CodeFileExists:         http.StatusConflict,
	CodeInvalidPlan:        http.StatusBadRequest,
	CodeRepositoryNotFound: http.StatusNotFound,
}

// registerCreateRoutes registers POST /plans.
func (c *Controller) registerCreateRoutes(api *gin.RouterGroup) {
	api.POST("/plans", c.httpCreatePlan)
}

func (c *Controller) httpCreatePlan(ctx *gin.Context) {
	workspaceID, ok := c.requireWorkspaceID(ctx)
	if !ok {
		return
	}
	var req CreatePlanRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{errKey: msgInvalidPayload, "code": CodeInvalidPlan})
		return
	}
	req.RepositoryID = strings.TrimSpace(req.RepositoryID)
	result, err := c.service.CreatePlan(ctx.Request.Context(), workspaceID, req)
	var createErr *CreatePlanError
	switch {
	case errors.As(err, &createErr):
		ctx.JSON(createStatusCodes[createErr.Code], gin.H{errKey: createErr.Message, "code": createErr.Code})
	case errors.Is(err, ErrNotConfigured):
		ctx.JSON(http.StatusNotFound, gin.H{errKey: msgConfigNone})
	case err != nil:
		c.failure(ctx, "failed to create the plan file", err)
	default:
		ctx.JSON(http.StatusCreated, result)
	}
}
