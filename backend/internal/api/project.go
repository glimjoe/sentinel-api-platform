// Package api — project handlers (M2-F.A).
//
// Endpoints:
//   POST   /api/v1/projects                          CreateProject
//   GET    /api/v1/projects                          ListProjects
//   GET    /api/v1/projects/:pid                     GetProject       (viewer+)
//   PATCH  /api/v1/projects/:pid                     UpdateProject    (admin)
//   DELETE /api/v1/projects/:pid                     DeleteProject    (owner)
//   GET    /api/v1/projects/:pid/members             ListMembers      (viewer+)
//   POST   /api/v1/projects/:pid/members             AddMember        (admin)
//
// All endpoints write through httpx.OK / middleware.WriteError so the
// wire format is uniform. Project-level RBAC is enforced via
// middleware.RequireProjectRole, mounted in main.go per-route — handlers
// stay thin and the role check is the same primitive every endpoint uses.
package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/middleware"
	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/httpx"
)

// projectService is the contract ProjectHandler needs from the service
// layer. *service.ProjectService satisfies it (see service/project_service.go).
// Declared here, not in service/, so handler tests can supply a fake
// without dragging the *gorm.DB-backed store fakes in.
type projectService interface {
	Create(ctx context.Context, ownerID, name, slug, description string) (*model.Project, error)
	List(ctx context.Context, ownerID string) ([]*model.Project, error)
}

// ProjectHandler wires HTTP routes to the project service.
type ProjectHandler struct {
	svc projectService
}

// NewProjectHandler constructs a ProjectHandler. Caller owns the lifetime.
func NewProjectHandler(svc projectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

// createProjectRequest mirrors the JSON body of POST /projects.
type createProjectRequest struct {
	Name        string `json:"name"        binding:"required"`
	Slug        string `json:"slug"        binding:"required"`
	Description string `json:"description"`
}

// CreateProject handles POST /api/v1/projects. The caller becomes the
// owner; the service layer auto-adds them as an admin project_member.
//
// user_id must be present in the gin context — it is set by the auth
// middleware that runs before this handler. If it's missing we fail loud
// (500/50001) rather than silently create a project with empty owner_id,
// which would surface later as a confusing FK violation.
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	callerID := c.GetString("user_id")
	if callerID == "" {
		httpx.Fail(c, http.StatusInternalServerError, 50001,
			"user_id missing from request context — route not protected")
		return
	}
	var req createProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	p, err := h.svc.Create(c.Request.Context(), callerID, req.Name, req.Slug, req.Description)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, p)
}

// ListProjects handles GET /api/v1/projects. Returns the caller's
// projects (currently owner-only; member-of will be added when the
// service layer exposes a ListByMember method). Empty slice (not nil)
// when the caller has none, so the frontend can iterate without a
// nil-check.
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	callerID := c.GetString("user_id")
	if callerID == "" {
		httpx.Fail(c, http.StatusInternalServerError, 50001,
			"user_id missing from request context — route not protected")
		return
	}
	list, err := h.svc.List(c.Request.Context(), callerID)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, list)
}
