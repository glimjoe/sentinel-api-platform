// Package middleware — CSRF verification (double-submit cookie pattern).
//
// ADR-0008: sent_csrf is a readable cookie set at login/register. The frontend
// reads it via document.cookie and sends it back as X-CSRF-Token header. This
// middleware verifies they match on all state-changing methods.
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/pkg/httpx"
)

// GenerateCSRFToken returns a random 32-byte hex token.
func GenerateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// CSRFRequired returns a middleware that verifies X-CSRF-Token matches the
// sent_csrf cookie on POST, PUT, PATCH, and DELETE requests.
func CSRFRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		csrfCookie, err := c.Cookie(cookieCSRF)
		if err != nil || csrfCookie == "" {
			httpx.Fail(c, http.StatusForbidden, 40301, "missing CSRF cookie — re-authenticate")
			c.Abort()
			return
		}

		csrfHeader := c.GetHeader("X-CSRF-Token")
		if csrfHeader == "" {
			httpx.Fail(c, http.StatusForbidden, 40301, "missing X-CSRF-Token header")
			c.Abort()
			return
		}

		if csrfCookie != csrfHeader {
			httpx.Fail(c, http.StatusForbidden, 40301, "CSRF token mismatch")
			c.Abort()
			return
		}

		c.Next()
	}
}
