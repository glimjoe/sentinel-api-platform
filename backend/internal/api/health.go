// Package api contains HTTP handlers (Gin) grouped by resource.
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// HealthHandler holds dependencies for liveness/readiness probes.
type HealthHandler struct {
	DB    *gorm.DB
	Redis *redis.Client
}

// Healthz returns 200 if the process is alive. Does NOT touch dependencies.
func (h *HealthHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"service": "sentinel",
		"version": "0.1.0",
	})
}

// Readyz returns 200 only if MySQL + Redis are both reachable.
func (h *HealthHandler) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	status := gin.H{"db": "ok", "redis": "ok"}
	overall := http.StatusOK

	sqlDB, err := h.DB.DB()
	if err != nil || sqlDB.PingContext(ctx) != nil {
		status["db"] = "down"
		overall = http.StatusServiceUnavailable
	}
	if err := h.Redis.Ping(ctx).Err(); err != nil {
		status["redis"] = "down"
		overall = http.StatusServiceUnavailable
	}
	c.JSON(overall, status)
}
