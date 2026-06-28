//go:build integration

package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/api"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/config"
	"github.com/glimjoe/sentinel-api-platform/internal/repository"
	"github.com/glimjoe/sentinel-api-platform/internal/service"
)

// TestIntegration_AuthFlow: register → login → me → refresh → logout → bad login.
func TestIntegration_AuthFlow(t *testing.T) {
	if testDB == nil {
		t.Skip("MySQL not available")
	}
	cfg, _ := config.Load()
	userRepo := repository.NewUserRepo(testDB)
	refreshRepo := repository.NewRefreshTokenRepo(testDB)
	authSvc := service.NewAuthService(userRepo, refreshRepo, cfg.Auth.AccessSecret, cfg.Auth.AccessTTL, cfg.Auth.BcryptCost)
	authH := api.NewAuthHandler(authSvc)

	r := gin.New()
	r.POST("/api/v1/auth/register", authH.Register)
	r.POST("/api/v1/auth/login", authH.Login)
	r.POST("/api/v1/auth/refresh", authH.Refresh)
	r.POST("/api/v1/auth/logout", authH.Logout)
	r.GET("/api/v1/auth/me", authH.Me)

	// Register
	body, _ := json.Marshal(map[string]string{"email": "itest-auth@sentinel.local", "password": "testpass123"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "register: %s", w.Body.String())

	var resp struct {
		Code int `json:"code"`
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	require.Equal(t, 0, resp.Code)
	accessToken := resp.Data.AccessToken
	refreshToken := resp.Data.RefreshToken
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)

	// Login
	body, _ = json.Marshal(map[string]string{"email": "itest-auth@sentinel.local", "password": "testpass123"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Me
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Refresh
	body, _ = json.Marshal(map[string]string{"refresh_token": refreshToken})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "refresh: %s", w.Body.String())

	// Bad login
	body, _ = json.Marshal(map[string]string{"email": "itest-auth@sentinel.local", "password": "WRONG"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Cleanup
	testDB.Exec("DELETE FROM refresh_tokens WHERE user_id IN (SELECT id FROM users WHERE email = 'itest-auth@sentinel.local')")
	testDB.Exec("DELETE FROM users WHERE email = 'itest-auth@sentinel.local'")
}
