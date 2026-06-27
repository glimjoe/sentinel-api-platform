package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AccessLog logs one structured entry per HTTP request (method, path, status, duration, request_id).
func AccessLog(base *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int("size", c.Writer.Size()),
		}
		if raw != "" {
			fields = append(fields, zap.String("query", raw))
		}
		if rid := GetRequestID(c); rid != "" {
			fields = append(fields, zap.String("request_id", rid))
		}

		switch {
		case status >= 500:
			base.Error("http_request", fields...)
		case status >= 400:
			base.Warn("http_request", fields...)
		default:
			base.Info("http_request", fields...)
		}
	}
}
