package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/pkg/httpx"
	"github.com/glimjoe/sentinel-api-platform/internal/service"
)

type TestRunHandler struct{ svc *service.TestRunService }

func NewTestRunHandler(svc *service.TestRunService) *TestRunHandler {
	return &TestRunHandler{svc: svc}
}

type createRunReq struct {
	Name          string `json:"name" binding:"required"`
	TargetBaseURL string `json:"target_base_url" binding:"required"`
	Mode          string `json:"mode"`
}

func (h *TestRunHandler) Create(c *gin.Context) {
	var req createRunReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	if req.Mode == "" {
		req.Mode = "sequential"
	}
	callerID := c.GetString("user_id")
	pid := c.Param("pid")
	run, err := h.svc.Create(c.Request.Context(), callerID, pid, req.Name, req.TargetBaseURL, req.Mode)
	if err != nil {
		httpx.Fail(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}
	httpx.OK(c, run)
}

func (h *TestRunHandler) Start(c *gin.Context) {
	callerID := c.GetString("user_id")
	run, err := h.svc.Start(c.Request.Context(), callerID, c.Param("runId"))
	if err != nil {
		httpx.Fail(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}
	httpx.OK(c, run)
}

func (h *TestRunHandler) List(c *gin.Context) {
	list, err := h.svc.ListByProject(c.Request.Context(), c.Param("pid"))
	if err != nil {
		httpx.Fail(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}
	httpx.OK(c, list)
}

func (h *TestRunHandler) Get(c *gin.Context) {
	run, err := h.svc.FindByID(c.Request.Context(), c.Param("runId"))
	if err != nil {
		httpx.Fail(c, http.StatusNotFound, 40400, err.Error())
		return
	}
	httpx.OK(c, run)
}
