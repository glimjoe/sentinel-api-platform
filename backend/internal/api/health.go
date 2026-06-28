// Package api — health handlers (M2-F.B migrated to httpx).
//
// Healthz / Readyz responses flow through httpx.OK / httpx.Fail so the
// envelope is uniform with the rest of the API. Readyz uses a custom
// 5xxxx application code to distinguish 'service not ready' from a
// generic 500; clients can render a different message in the UI.
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/glimjoe/sentinel-api-platform/internal/pkg/httpx"
)

// HealthHandler holds dependencies for liveness/readiness probes.
type HealthHandler struct {
	DB    *gorm.DB
	Redis *redis.Client
}

// Healthz returns 200 if the process is alive. Does NOT touch dependencies.
func (h *HealthHandler) Healthz(c *gin.Context) {
	httpx.OK(c, gin.H{
		"ok":      true,
		"service": "sentinel",
		"version": "0.1.0",
	})
}

// Readyz returns 200 only if MySQL + Redis are both reachable. If either
// dependency is down, returns 503 with the per-dependency status in the
// envelope data. The application code 50300 tells the client "I'm alive
// but not ready" — distinct from a generic 500.
func (h *HealthHandler) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	status := gin.H{"db": "ok", "redis": "ok"}
	ready := true

	sqlDB, err := h.DB.DB()
	if err != nil || sqlDB.PingContext(ctx) != nil {
		status["db"] = "down"
		ready = false
	}
	if err := h.Redis.Ping(ctx).Err(); err != nil {
		status["redis"] = "down"
		ready = false
	}
	if !ready {
		httpx.Fail(c, http.StatusServiceUnavailable, 50300, "service not ready")
		return
	}
	httpx.OK(c, status)
}
