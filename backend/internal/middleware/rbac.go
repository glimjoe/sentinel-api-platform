// Package middleware — RBAC primitives (M2-D).
//
// ProjectRoleResolver decouples this package from the repository layer:
// the middleware asks "what role does user X have on project Y?" and the
// service layer supplies an implementation backed by (project.owner_id ==
// user_id) or the project_members table. This keeps tests trivial —
// supply a fake resolver, no DB needed.
//
// Response shape: every non-200 path writes through httpx.Fail so the
// caller sees the canonical {code, message, request_id} envelope. M2-F
// handlers will mount RequireProjectRole in front of write endpoints.
package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/httpx"
)

// ProjectRoleResolver returns the caller's project-level role.
//   - ("<role>", nil) — caller is a member with that role.
//   - ("", nil)        — caller is not a member of this project.
//   - (_, err)         — genuine failure (DB down, timeout). Caller MUST
//     distinguish this from "not a member" — see the 500 branch below.
type ProjectRoleResolver interface {
	RoleFor(ctx context.Context, projectID, userID string) (string, error)
}

// RequireProjectRole blocks requests whose caller's project role is not
// in `levels`. The project id is read from the :pid URL param.
//
// Status codes:
//   200 → reach handler (role ∈ levels)
//   403 → not a member, OR member but role ∉ levels
//   404 → project does not exist (resolver returns ErrNotFound)
//   500 → missing user_id (route-grouping bug), missing :pid, or resolver error
//
// Owner check is delegated to the resolver: a typical impl returns "admin"
// if project.owner_id == userID before consulting project_members, so the
// owner doesn't need a redundant project_members row.
func RequireProjectRole(resolver ProjectRoleResolver, levels ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		if userID == "" {
			// Fail loud — a protected route should have run the auth
			// middleware before this. Silently coercing to "" would mask
			// a route-grouping bug as a confusing 403.
			httpx.Fail(c, http.StatusInternalServerError, 50001,
				"user_id missing from request context — route not protected")
			return
		}
		projectID := c.Param("pid")
		if projectID == "" {
			// Same logic for project id — :pid is required.
			httpx.Fail(c, http.StatusInternalServerError, 50002,
				"project id missing from URL — route not parameterized with :pid")
			return
		}

		role, err := resolver.RoleFor(c.Request.Context(), projectID, userID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				// Project doesn't exist → 404, not 403. Lets clients
				// distinguish "wrong URL" from "no access".
				httpx.Fail(c, http.StatusNotFound, 40400, "project not found")
				return
			}
			// Genuine failure (DB down, timeout). 500, not silent 403 —
			// operators must be able to tell "not allowed" from "broken".
			httpx.Fail(c, http.StatusInternalServerError, 50003, "rbac resolver failed")
			return
		}
		if role == "" {
			httpx.Fail(c, http.StatusForbidden, 40300, "not a member of this project")
			return
		}
		if !containsRole(levels, role) {
			httpx.Fail(c, http.StatusForbidden, 40301, "insufficient project role")
			return
		}
		c.Next()
	}
}

// containsRole reports whether role is one of levels. Kept as a tiny
// helper so the loop above reads as a single sentence.
func containsRole(levels []string, role string) bool {
	for _, l := range levels {
		if l == role {
			return true
		}
	}
	return false
}