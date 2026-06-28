package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/id"
)

// AuditRecorder persists audit entries.
type AuditRecorder interface {
	Insert(ctx context.Context, entry *model.AuditLog) error
}

// Audit logs POST/PATCH/DELETE requests after the handler completes.
// Only metadata is captured (no request body) to avoid consuming the
// io.Reader before the handler can bind it.
func Audit(recorder AuditRecorder) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		method := c.Request.Method
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			return
		}

		entry := &model.AuditLog{
			ID:           id.New(),
			UserID:       c.GetString("user_id"),
			Action:       method + " " + c.FullPath(),
			ResourceType: extractResType(c.FullPath()),
			ResourceID:   c.Param("pid"),
			IP:           c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
			CreatedAt:    time.Now(),
		}
		_ = recorder.Insert(c.Request.Context(), entry)
	}
}

func extractResType(path string) string {
	if len(path) == 0 {
		return "unknown"
	}
	slash := 0
	start := 0
	for i, b := range []byte(path) {
		if b == '/' {
			slash++
			if slash == 3 {
				start = i + 1
				break
			}
		}
	}
	if start == 0 || start >= len(path) {
		return path
	}
	end := len(path)
	for i := start; i < len(path); i++ {
		if path[i] == '/' || path[i] == ':' {
			end = i
			break
		}
	}
	return path[start:end]
}
