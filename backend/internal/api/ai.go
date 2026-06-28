package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/middleware"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/httpx"
	"github.com/glimjoe/sentinel-api-platform/internal/service"
)

// AIHandler handles AI feature endpoints.
type AIHandler struct{ svc *service.AIService }

// NewAIHandler constructs an AIHandler.
func NewAIHandler(svc *service.AIService) *AIHandler { return &AIHandler{svc: svc} }

type attributeReq struct {
	ResultJSON string `json:"result_json" binding:"required"`
}

func (h *AIHandler) Attribute(c *gin.Context) {
	var req attributeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	callerID := c.GetString("user_id")
	pid := c.Param("pid")
	result, err := h.svc.Attribute(c.Request.Context(), callerID, pid, req.ResultJSON)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, result)
}

type completeReq struct {
	APIID *string `json:"api_id"`
}

func (h *AIHandler) Complete(c *gin.Context) {
	var req completeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	callerID := c.GetString("user_id")
	pid := c.Param("pid")
	cases, err := h.svc.Complete(c.Request.Context(), callerID, pid, req.APIID)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, gin.H{"test_cases": cases})
}

type prioritizeReq struct {
	CaseIDs []string `json:"case_ids"`
}

func (h *AIHandler) Prioritize(c *gin.Context) {
	var req prioritizeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	callerID := c.GetString("user_id")
	pid := c.Param("pid")
	priorities, err := h.svc.Prioritize(c.Request.Context(), callerID, pid, req.CaseIDs)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, gin.H{"priorities": priorities})
}

func (h *AIHandler) Budget(c *gin.Context) {
	httpx.OK(c, h.svc.Budget())
}
