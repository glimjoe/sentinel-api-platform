package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/middleware"
	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/httpx"
	"github.com/glimjoe/sentinel-api-platform/internal/service"
)

type TestCaseHandler struct{ svc *service.TestCaseService }

func NewTestCaseHandler(svc *service.TestCaseService) *TestCaseHandler {
	return &TestCaseHandler{svc: svc}
}

func (h *TestCaseHandler) Create(c *gin.Context) {
	var tc model.TestCase
	if err := c.ShouldBindJSON(&tc); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	callerID := c.GetString("user_id")
	pid := c.Param("pid")
	result, err := h.svc.Create(c.Request.Context(), callerID, pid, &tc)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, result)
}

func (h *TestCaseHandler) List(c *gin.Context) {
	list, err := h.svc.ListByProject(c.Request.Context(), c.Param("pid"))
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, list)
}

func (h *TestCaseHandler) Get(c *gin.Context) {
	tc, err := h.svc.FindByID(c.Request.Context(), c.Param("caseId"))
	if err != nil {
		httpx.Fail(c, http.StatusNotFound, 40400, err.Error())
		return
	}
	httpx.OK(c, tc)
}

func (h *TestCaseHandler) Update(c *gin.Context) {
	var fields map[string]any
	if err := c.ShouldBindJSON(&fields); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	callerID := c.GetString("user_id")
	pid := c.Param("pid")
	tc, err := h.svc.Update(c.Request.Context(), callerID, pid, c.Param("caseId"), fields)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, tc)
}

func (h *TestCaseHandler) Delete(c *gin.Context) {
	callerID := c.GetString("user_id")
	pid := c.Param("pid")
	if err := h.svc.Delete(c.Request.Context(), callerID, pid, c.Param("caseId")); err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, nil)
}
