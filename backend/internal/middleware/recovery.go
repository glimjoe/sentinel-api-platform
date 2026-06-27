package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery converts panics into a 500 JSON response and logs the stack trace.
func Recovery(base *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				fields := []zap.Field{
					zap.Any("panic", err),
					zap.ByteString("stack", stack),
				}
				if rid := GetRequestID(c); rid != "" {
					fields = append(fields, zap.String("request_id", rid))
				}
				base.Error("panic_recovered", fields...)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    "internal_error",
					"message": "an unexpected error occurred",
				})
			}
		}()
		c.Next()
	}
}
