// Package middleware contains HTTP middleware for the Sentinel API.
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-Id"

// RequestID generates or propagates an X-Request-Id header and stores it in
// both the response header and the gin Context (key: "request_id").
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Writer.Header().Set(RequestIDHeader, id)
		c.Set("request_id", id)
		c.Next()
	}
}

// GetRequestID retrieves the request id from a gin context. Returns "" if absent.
func GetRequestID(c *gin.Context) string {
	if v, ok := c.Get("request_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
