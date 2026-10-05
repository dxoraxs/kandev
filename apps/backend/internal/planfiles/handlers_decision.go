package planfiles

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// decisionStatus maps a decision error code to its HTTP status.
var decisionStatus = map[string]int{
	CodeNotPlanTask:     http.StatusNotFound,
	CodeNotWaitingOwner: http.StatusConflict,
	CodeFileChanged:     http.StatusConflict,
	CodeInvalidDecision: http.StatusBadRequest,
}

// registerDecisionRoutes registers POST /tasks/:taskId/decision. The workspace
// comes from the task's sync row, so the route takes no workspace parameter.
func (c *Controller) registerDecisionRoutes(api *gin.RouterGroup) {
	api.POST("/tasks/:taskId/decision", c.httpDecision)
}

func (c *Controller) httpDecision(ctx *gin.Context) {
	var req DecisionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{errKey: msgInvalidPayload, "code": CodeInvalidDecision})
		return
	}
	result, err := c.service.Decide(ctx.Request.Context(), strings.TrimSpace(ctx.Param("taskId")), req)
	var decisionErr *DecisionError
	switch {
	case errors.As(err, &decisionErr):
		ctx.JSON(decisionStatus[decisionErr.Code], gin.H{errKey: decisionErr.Message, "code": decisionErr.Code})
	case errors.Is(err, ErrNotConfigured):
		ctx.JSON(http.StatusNotFound, gin.H{errKey: msgConfigNone})
	case err != nil:
		c.failure(ctx, "failed to apply the plan decision", err)
	default:
		ctx.JSON(http.StatusOK, result)
	}
}
