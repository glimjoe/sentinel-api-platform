// Package api — API handlers (M2-F.C).
//
// Endpoints:
//   POST   /api/v1/projects/:pid/apis              CreateAPI       (admin/engineer)
//   GET    /api/v1/projects/:pid/apis              ListAPIs        (viewer+)
//   GET    /api/v1/projects/:pid/apis/:apiId       GetAPI          (viewer+)
//   PATCH  /api/v1/projects/:pid/apis/:apiId       UpdateAPI       (admin/engineer)
//   DELETE /api/v1/projects/:pid/apis/:apiId       DeleteAPI       (admin)
//   POST   /api/v1/projects/:pid/apis/:apiId/tags   AddTag          (admin/engineer)
//   DELETE /api/v1/projects/:pid/apis/:apiId/tags/:tag RemoveTag      (admin/engineer)
package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/middleware"
	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/httpx"
)

type apiService interface {
	Create(ctx context.Context, callerID, projectID, name, method, path, operationID string) (*model.API, error)
	ListByProject(ctx context.Context, projectID string) ([]*model.API, error)
	FindByID(ctx context.Context, id string) (*model.API, error)
	Update(ctx context.Context, callerID, apiID string, fields map[string]any) (*model.API, error)
	Delete(ctx context.Context, callerID, apiID string) error
}

type APIHandler struct {
	svc apiService
}

func NewAPIHandler(svc apiService) *APIHandler {
	return &APIHandler{svc: svc}
}

type createAPIRequest struct {
	Name        string `json:"name"        binding:"required"`
	Method      string `json:"method"      binding:"required"`
	Path        string `json:"path"        binding:"required"`
	OperationID string `json:"operation_id"`
}

func (h *APIHandler) CreateAPI(c *gin.Context) {
	callerID := c.GetString("user_id")
	if callerID == "" {
		httpx.Fail(c, http.StatusInternalServerError, 50001, "user_id missing from request context — route not protected")
		return
	}
	var req createAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	pid := c.Param("pid")
	a, err := h.svc.Create(c.Request.Context(), callerID, pid, req.Name, req.Method, req.Path, req.OperationID)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, a)
}

func (h *APIHandler) ListAPIs(c *gin.Context) {
	if c.GetString("user_id") == "" {
		httpx.Fail(c, http.StatusInternalServerError, 50001, "user_id missing from request context — route not protected")
		return
	}
	pid := c.Param("pid")
	list, err := h.svc.ListByProject(c.Request.Context(), pid)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, list)
}

func (h *APIHandler) GetAPI(c *gin.Context) {
	if c.GetString("user_id") == "" {
		httpx.Fail(c, http.StatusInternalServerError, 50001, "user_id missing from request context — route not protected")
		return
	}
	apiID := c.Param("apiId")
	a, err := h.svc.FindByID(c.Request.Context(), apiID)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, a)
}

type updateAPIRequest struct {
	Name        string `json:"name"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	OperationID string `json:"operation_id"`
	Deprecated  *bool  `json:"deprecated"`
}

func (h *APIHandler) UpdateAPI(c *gin.Context) {
	callerID := c.GetString("user_id")
	if callerID == "" {
		httpx.Fail(c, http.StatusInternalServerError, 50001, "user_id missing from request context — route not protected")
		return
	}
	var req updateAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	fields := map[string]any{}
	if req.Name != "" {
		fields["name"] = req.Name
	}
	if req.Method != "" {
		fields["method"] = req.Method
	}
	if req.Path != "" {
		fields["path"] = req.Path
	}
	if req.OperationID != "" {
		fields["operation_id"] = req.OperationID
	}
	if req.Deprecated != nil {
		fields["deprecated"] = *req.Deprecated
	}
	apiID := c.Param("apiId")
	a, err := h.svc.Update(c.Request.Context(), callerID, apiID, fields)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, a)
}

func (h *APIHandler) DeleteAPI(c *gin.Context) {
	callerID := c.GetString("user_id")
	if callerID == "" {
		httpx.Fail(c, http.StatusInternalServerError, 50001, "user_id missing from request context — route not protected")
		return
	}
	apiID := c.Param("apiId")
	if err := h.svc.Delete(c.Request.Context(), callerID, apiID); err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, nil)
}

type addTagRequest struct {
	Tag string `json:"tag" binding:"required"`
}

func (h *APIHandler) AddTag(c *gin.Context) {
	callerID := c.GetString("user_id")
	if callerID == "" {
		httpx.Fail(c, http.StatusInternalServerError, 50001, "user_id missing from request context — route not protected")
		return
	}
	var req addTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	apiID := c.Param("apiId")
	a, err := h.svc.FindByID(c.Request.Context(), apiID)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	tags := decodeTags(a.TagsJSON)
	tags = appendUnique(tags, req.Tag)
	updated, err := h.svc.Update(c.Request.Context(), callerID, apiID, map[string]any{
		"tags_json": encodeTags(tags),
	})
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, updated)
}

func (h *APIHandler) RemoveTag(c *gin.Context) {
	callerID := c.GetString("user_id")
	if callerID == "" {
		httpx.Fail(c, http.StatusInternalServerError, 50001, "user_id missing from request context — route not protected")
		return
	}
	apiID := c.Param("apiId")
	tag := c.Param("tag")
	a, err := h.svc.FindByID(c.Request.Context(), apiID)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	tags := removeTag(decodeTags(a.TagsJSON), tag)
	updated, err := h.svc.Update(c.Request.Context(), callerID, apiID, map[string]any{
		"tags_json": encodeTags(tags),
	})
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, updated)
}

func decodeTags(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var tags []string
	_ = json.Unmarshal(raw, &tags)
	if tags == nil {
		return []string{}
	}
	return tags
}

func encodeTags(tags []string) []byte {
	b, _ := json.Marshal(tags)
	return b
}

func appendUnique(tags []string, tag string) []string {
	for _, t := range tags {
		if t == tag {
			return tags
		}
	}
	return append(tags, tag)
}

func removeTag(tags []string, tag string) []string {
	out := tags[:0]
	for _, t := range tags {
		if t != tag {
			out = append(out, t)
		}
	}
	return out
}
