// Package api — auth handlers (Phase 1, migrated to httpx in M2-F.B).
//
// Endpoints:
//   POST /api/v1/auth/register   { email, password, display_name? }   → 200 { user, access_token }
//   POST /api/v1/auth/login      { email, password }                 → 200 { user, access_token }
//   GET  /api/v1/auth/me         (auth required)                     → 200 { user }
//
// All responses go through httpx.OK / middleware.WriteError so the
// envelope is uniform. The previous writeAuthError local switcher was
// removed — middleware.WriteError is the single source of truth for
// the errs → HTTP mapping.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/middleware"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/httpx"
	"github.com/glimjoe/sentinel-api-platform/internal/service"
)

// AuthHandler holds the auth service dependency.
type AuthHandler struct {
	svc *service.AuthService
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// registerRequest mirrors the JSON body of POST /auth/register.
type registerRequest struct {
	Email       string `json:"email"        binding:"required,email"`
	Password    string `json:"password"     binding:"required,min=8"`
	DisplayName string `json:"display_name"`
}

// loginRequest mirrors the JSON body of POST /auth/login.
type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// authResponse is the success payload for Register / Login / Refresh.
type authResponse struct {
	User         any    `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

// Register handles POST /auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	u, accessToken, refreshToken, err := h.svc.Register(c.Request.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, authResponse{User: u, AccessToken: accessToken, RefreshToken: refreshToken, TokenType: "Bearer"})
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	u, accessToken, refreshToken, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, authResponse{User: u, AccessToken: accessToken, RefreshToken: refreshToken, TokenType: "Bearer"})
}

// Me handles GET /auth/me. Caller is expected to be authenticated; the
// middleware has already injected user_id into the gin context. We re-resolve
// the row so a deleted/disabled account can't keep a stale token alive.
//
// If user_id is missing or not a string, fail loud with 500 — this means a
// route-grouping mistake (e.g. registering Me on the unprotected engine
// instead of inside the protected group). Silently coercing to "" would
// surface as a confusing 404 via FindByID("", ...).
func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		httpx.Fail(c, http.StatusInternalServerError, 50001,
			"user_id missing from request context — route not protected")
		return
	}
	u, err := h.svc.Me(c.Request.Context(), userID)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, gin.H{"user": u})
}

// refreshRequest is the JSON body for POST /auth/refresh.
type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh handles POST /auth/refresh. Accepts a refresh token, returns a new
// access_token + refresh_token pair. This endpoint is public (no auth required).
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	u, accessToken, refreshToken, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, authResponse{User: u, AccessToken: accessToken, RefreshToken: refreshToken, TokenType: "Bearer"})
}

// Logout handles POST /auth/logout. Revoked refresh tokens are no longer usable.
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		httpx.Fail(c, http.StatusInternalServerError, 50001,
			"user_id missing from request context — route not protected")
		return
	}
	if err := h.svc.Logout(c.Request.Context(), userID); err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, nil)
}