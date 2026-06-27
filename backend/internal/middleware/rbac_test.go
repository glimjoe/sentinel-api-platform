package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// fakeResolver is a hand-rolled fake for ProjectRoleResolver. The repo
// intentionally avoids a mocking library so the test file has no extra
// dependency; this struct is small enough to inline.
type fakeResolver struct {
	// roleByUser maps userID → role for a single fixed project (the
	// middleware only resolves one project per request, set via setRole).
	roleByUser map[string]string
	// err is returned by RoleFor if non-nil (e.g. simulating DB down).
	err error
	// calls records every (projectID, userID) pair the middleware asks about.
	calls []struct{ projectID, userID string }
}

func (f *fakeResolver) RoleFor(ctx context.Context, projectID, userID string) (string, error) {
	f.calls = append(f.calls, struct{ projectID, userID string }{projectID, userID})
	if f.err != nil {
		return "", f.err
	}
	return f.roleByUser[userID], nil
}

// runRequest wires a single handler behind RequireProjectRole and drives it
// with the given user_id + path (path carries :pid). Returns the recorder.
func runRequest(t *testing.T, resolver ProjectRoleResolver, levels []string, userID, path string) *httptest.ResponseRecorder {
	t.Helper()
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if userID != "" {
			c.Set("user_id", userID)
		}
		c.Next()
	})
	r.GET("/p/:pid/*rest", RequireProjectRole(resolver, levels...), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	return w
}

// TestRequireProjectRole_AdminPassesForAdminMember — admin member calling
// an admin-only endpoint must reach the handler.
func TestRequireProjectRole_AdminPassesForAdminMember(t *testing.T) {
	resolver := &fakeResolver{roleByUser: map[string]string{"u1": "admin"}}
	w := runRequest(t, resolver, []string{"admin", "engineer", "viewer"}, "u1", "/p/proj-1/x")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if len(resolver.calls) != 1 || resolver.calls[0].projectID != "proj-1" || resolver.calls[0].userID != "u1" {
		t.Errorf("resolver calls = %+v", resolver.calls)
	}
}

// TestRequireProjectRole_ViewerBlockedFromAdmin — viewer member must NOT
// reach an admin-only endpoint. 403 with the canonical httpx envelope shape
// (the actual response is written through httpx.Fail; we only assert the
// status code here — wire format is owned by httpx_test.go).
func TestRequireProjectRole_ViewerBlockedFromAdmin(t *testing.T) {
	resolver := &fakeResolver{roleByUser: map[string]string{"u1": "viewer"}}
	w := runRequest(t, resolver, []string{"admin"}, "u1", "/p/proj-1/x")

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

// TestRequireProjectRole_EngineerPassesForWrite — engineer role should be
// sufficient for endpoints that allow admin/engineer (write endpoints).
func TestRequireProjectRole_EngineerPassesForWrite(t *testing.T) {
	resolver := &fakeResolver{roleByUser: map[string]string{"u1": "engineer"}}
	w := runRequest(t, resolver, []string{"admin", "engineer"}, "u1", "/p/proj-1/x")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

// TestRequireProjectRole_NonMemberBlocked — a user with no entry in
// project_members (RoleFor returns "" + nil) must be 403ed.
func TestRequireProjectRole_NonMemberBlocked(t *testing.T) {
	resolver := &fakeResolver{roleByUser: map[string]string{}}
	w := runRequest(t, resolver, []string{"admin", "engineer", "viewer"}, "u1", "/p/proj-1/x")

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

// TestRequireProjectRole_ResolverErrorIs500 — if the resolver fails (e.g.
// DB down), the middleware must surface a 500 rather than silently 403
// (otherwise operators can't tell "not allowed" from "broken").
func TestRequireProjectRole_ResolverErrorIs500(t *testing.T) {
	resolver := &fakeResolver{err: errors.New("db down")}
	w := runRequest(t, resolver, []string{"admin"}, "u1", "/p/proj-1/x")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
}

// TestRequireProjectRole_MissingUserIDIs500 — fail loud if a route forgot
// to run the auth middleware before RBAC. Silently coercing to "" would
// mask a route-grouping bug as a confusing 403.
func TestRequireProjectRole_MissingUserIDIs500(t *testing.T) {
	resolver := &fakeResolver{roleByUser: map[string]string{"u1": "admin"}}
	w := runRequest(t, resolver, []string{"admin"}, "", "/p/proj-1/x")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
}