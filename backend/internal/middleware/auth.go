package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/jwt"
)

// Context keys used by AuthRequired and CookieAuthRequired.
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

// CookieAuthRequired (ADR-0008) reads the access token from the sent_access
// cookie. Returns 401 with X-Token-Expired header when the access token is
// expired so the frontend can trigger a silent refresh via /auth/refresh.
func CookieAuthRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken, err := c.Cookie(cookieAccess)
		if err != nil || accessToken == "" {
			abort401(c, "missing access cookie")
			return
		}

		claims, parseErr := jwt.Parse(secret, accessToken)
		if parseErr == nil {
			c.Set(CtxUserID, claims.UserID)
			c.Set(CtxEmail, claims.Email)
			c.Set(CtxRole, claims.Role)
			c.Next()
			return
		}

		if errors.Is(parseErr, errs.ErrTokenExpired) {
			c.Header("X-Token-Expired", "1")
			abort401(c, "token expired")
			return
		}

		abort401(c, "invalid access token")
	}
}

// TokenQueryAuth validates a JWT passed as the ?token= query parameter.
// Used for SSE endpoints where EventSource cannot send custom headers.
func TokenQueryAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			abort401(c, "missing token query parameter")
			return
		}
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
