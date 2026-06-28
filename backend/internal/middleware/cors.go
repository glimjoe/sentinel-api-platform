package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS allows the configured frontend origin to call the API.
// For development, also handles preflight (OPTIONS) requests.
func CORS(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip the header when no origin is configured: writing an empty value
		// is interpreted by browsers as "no origin allowed" and silently
		// breaks every cross-origin XHR. Production validation in pkg/config
		// requires FRONTEND_ORIGIN to be set; in dev, leaving it unset is OK
		// and behaves like a same-origin server.
		if allowedOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-Id, X-CSRF-Token")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
