// Package api — auth handlers (Phase 1, cookie-based auth per ADR-0008).
//
// Endpoints:
//
//	POST /api/v1/auth/register   { email, password, display_name? }   → 200 { user }
//	POST /api/v1/auth/login      { email, password }                 → 200 { user }
//	GET  /api/v1/auth/me         (auth required)                     → 200 { user }
//	POST /api/v1/auth/refresh    (reads sent_refresh cookie)         → 200 { user }
//	POST /api/v1/auth/logout     (auth required, clears cookies)
//
// All auth responses set httpOnly cookies (sent_access, sent_refresh) plus a
// readable CSRF token cookie (sent_csrf) per ADR-0008.
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
// ADR-0008: tokens are set as httpOnly cookies, so the body only returns user.
type authResponse struct {
	User interface{} `json:"user"`
}

// Register handles POST /auth/register.
// Sets sent_access, sent_refresh (httpOnly), and sent_csrf (readable) cookies.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	u, accessToken, refreshToken, err := h.svc.Register(
		c.Request.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	csrfToken := middleware.GenerateCSRFToken()
	middleware.SetAuthCookies(c.Writer, accessToken, refreshToken, csrfToken)
	httpx.OK(c, authResponse{User: u})
}

// Login handles POST /auth/login.
// Sets sent_access, sent_refresh (httpOnly), and sent_csrf (readable) cookies.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	u, accessToken, refreshToken, err := h.svc.Login(
		c.Request.Context(), req.Email, req.Password)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	csrfToken := middleware.GenerateCSRFToken()
	middleware.SetAuthCookies(c.Writer, accessToken, refreshToken, csrfToken)
	httpx.OK(c, authResponse{User: u})
}

// Me handles GET /auth/me.
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

// Refresh handles POST /auth/refresh. Reads the refresh token from the
// sent_refresh cookie (ADR-0008) instead of a JSON body.
func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("sent_refresh")
	if err != nil || refreshToken == "" {
		httpx.Fail(c, http.StatusUnauthorized, 40100, "missing refresh cookie")
		return
	}
	u, accessToken, newRefreshToken, svcErr := h.svc.Refresh(
		c.Request.Context(), refreshToken)
	if svcErr != nil {
		middleware.WriteError(c, svcErr)
		return
	}
	csrfToken := middleware.GenerateCSRFToken()
	middleware.SetAuthCookies(c.Writer, accessToken, newRefreshToken, csrfToken)
	httpx.OK(c, authResponse{User: u})
}

// Logout handles POST /auth/logout. Revokes all refresh tokens for the user
// and clears all auth cookies.
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
	middleware.ClearAuthCookies(c.Writer)
	httpx.OK(c, nil)
}
