package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/jwt"
)

// Context keys used by AuthRequired.
const (
	CtxUserID = "user_id"
	CtxEmail  = "email"
	CtxRole   = "role"
)

// AuthRequired validates the Bearer token on the Authorization header and
// injects user_id / email / role into the gin context for downstream handlers.
// Returns 401 on missing, malformed, or expired tokens.
func AuthRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			abort401(c, "missing Authorization header")
			return
		}
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			abort401(c, "Authorization header must use Bearer scheme")
			return
		}
		token := strings.TrimSpace(header[len(prefix):])
		claims, err := jwt.Parse(secret, token)
		if err != nil {
			switch {
			case errors.Is(err, errs.ErrTokenExpired):
				abort401(c, "token expired")
			default:
				abort401(c, "invalid token")
			}
			return
		}
		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxEmail, claims.Email)
		c.Set(CtxRole, claims.Role)
		c.Next()
	}
}

func abort401(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": msg})
}