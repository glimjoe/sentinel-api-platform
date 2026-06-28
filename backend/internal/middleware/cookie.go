// Package middleware — cookie helpers for ADR-0008 (JWT → httpOnly cookie).
//
// Cookie naming:
//
//	sent_access  — access token  (HttpOnly; Secure in prod; SameSite=Lax;  Path=/;             Max-Age=900)
//	sent_refresh — refresh token (HttpOnly; Secure in prod; SameSite=Strict; Path=/api/v1/auth; Max-Age=604800)
//	sent_csrf    — CSRF token   (readable by JS;  Secure in prod; SameSite=Lax;  Path=/;             Max-Age=86400)
package middleware

import (
	"net/http"
	"time"
)

const (
	cookieAccess  = "sent_access"
	cookieRefresh = "sent_refresh"
	cookieCSRF    = "sent_csrf"

	defaultAccessTTL  = 15 * time.Minute
	defaultRefreshTTL = 7 * 24 * time.Hour
	defaultCSRFTTL    = 24 * time.Hour
)

// cookieConfig carries the deployment-specific cookie knobs.
type cookieConfig struct {
	secure bool
	domain string
}

// cookieCfg is set by main and defaults to secure=false (dev-friendly).
var cookieCfg = cookieConfig{secure: false}

// SetCookieConfig allows main to toggle Secure flag and Domain for production.
func SetCookieConfig(secure bool, domain string) {
	cookieCfg = cookieConfig{secure: secure, domain: domain}
}

// SetAuthCookies writes sent_access, sent_refresh, and sent_csrf to the response.
func SetAuthCookies(w http.ResponseWriter, accessToken, refreshToken, csrfToken string) {
	setCookie(w, cookieAccess, accessToken, "/", defaultAccessTTL, true)
	setCookie(w, cookieRefresh, refreshToken, "/api/v1/auth", defaultRefreshTTL, true)
	setCookie(w, cookieCSRF, csrfToken, "/", defaultCSRFTTL, false)
}

// ClearAuthCookies removes all three auth cookies by setting Max-Age=0.
func ClearAuthCookies(w http.ResponseWriter) {
	setCookie(w, cookieAccess, "", "/", 0, true)
	setCookie(w, cookieRefresh, "", "/api/v1/auth", 0, true)
	setCookie(w, cookieCSRF, "", "/", 0, false)
}

func setCookie(w http.ResponseWriter, name, value, path string, ttl time.Duration, httpOnly bool) {
	maxAge := int(ttl.Seconds())
	if ttl == 0 {
		maxAge = -1
	}
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		HttpOnly: httpOnly,
		Secure:   cookieCfg.secure,
		SameSite: http.SameSiteLaxMode,
	}
	if name == cookieRefresh {
		c.SameSite = http.SameSiteStrictMode
	}
	if cookieCfg.domain != "" {
		c.Domain = cookieCfg.domain
	}
	http.SetCookie(w, c)
}
