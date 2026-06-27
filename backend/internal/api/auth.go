// Package api — auth handlers (Phase 1).
//
// Endpoints:
//   POST /api/v1/auth/register   { email, password, display_name? }   → 201 { user, access_token }
//   POST /api/v1/auth/login      { email, password }                 → 200 { user, access_token }
//   GET  /api/v1/auth/me         (auth required)                     → 200 { user }
package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
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

// Register handles POST /auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, token, err := h.svc.Register(c.Request.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		writeAuthError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"user":         u,
		"access_token": token,
		"token_type":   "Bearer",
	})
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, token, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		writeAuthError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user":         u,
		"access_token": token,
		"token_type":   "Bearer",
	})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id missing from request context — route not protected"})
		return
	}
	u, err := h.svc.Me(c.Request.Context(), userID)
	if err != nil {
		writeAuthError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": u})
}

// writeAuthError maps service errors to HTTP responses. The default branch
// returns a generic 500 body — the full wrap chain is already captured by
// AccessLog/Recovery middleware, so we don't try to unmangle it client-side.
func writeAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errs.ErrEmailTaken):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, errs.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	case errors.Is(err, errs.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, errs.ErrUserInactive):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, errs.ErrBadRequest):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}