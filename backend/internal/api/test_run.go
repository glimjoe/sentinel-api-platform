package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/pkg/httpx"
	"github.com/glimjoe/sentinel-api-platform/internal/runner"
	"github.com/glimjoe/sentinel-api-platform/internal/service"
)

type TestRunHandler struct {
	svc    *service.TestRunService
	broker runner.EventSubscriber
}

func NewTestRunHandler(svc *service.TestRunService, broker runner.EventSubscriber) *TestRunHandler {
	return &TestRunHandler{svc: svc, broker: broker}
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

func (h *TestRunHandler) Cancel(c *gin.Context) {
	callerID := c.GetString("user_id")
	if err := h.svc.Cancel(c.Request.Context(), callerID, c.Param("runId")); err != nil {
		httpx.Fail(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}
	httpx.OK(c, nil)
}

// Stream handles GET /api/v1/projects/:pid/runs/:runId/stream — SSE.
func (h *TestRunHandler) Export(c *gin.Context) {
	results, err := h.svc.ExportResults(c.Request.Context(), c.Param("runId"))
	if err != nil {
		httpx.Fail(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}
	httpx.OK(c, results)
}

func (h *TestRunHandler) Stream(c *gin.Context) {
	runID := c.Param("runId")
	if h.broker == nil {
		httpx.Fail(c, http.StatusNotImplemented, 50100, "SSE not configured")
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	ch, cleanup, err := h.broker.Subscribe(ctx, runID)
	if err != nil {
		httpx.Fail(c, http.StatusInternalServerError, 50000, err.Error())
		return
	}
	defer cleanup()

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		httpx.Fail(c, http.StatusInternalServerError, 50000, "streaming not supported")
		return
	}

	for evt := range ch {
		data, _ := json.Marshal(evt)
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		flusher.Flush()
		if evt.Type == "complete" {
			return
		}
	}
}
