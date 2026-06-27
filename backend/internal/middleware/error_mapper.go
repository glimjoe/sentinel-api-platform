// Package middleware — WriteError (M2-F foundation).
//
// WriteError is the canonical "service-layer error → HTTP response" translator
// for the Sentinel API. Every handler that calls into a service should defer
// to WriteError rather than switching on errors.Is in-line, so the wire
// format stays uniform across resources and the application code space
// (see codes in httpx.Fail) is owned in one place.
//
// Code assignments align with the rbac middleware's existing codes
// (40300/40400) so clients see one semantic class per HTTP status — the
// distinction between "you are not a member" (rbac) and "your role is too
// low" (service ErrForbidden) is too fine to expose on the wire.
//
// Unknown errors collapse to 500 / 50000 with a generic message. The full
// wrap chain is captured by the AccessLog middleware, so we don't try to
// unmangle it client-side (and risk leaking internals).
package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/httpx"
)

// WriteError maps a service-layer error to an httpx.Fail envelope.
// See package doc for the full code table.
func WriteError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errs.ErrBadRequest):
		httpx.Fail(c, http.StatusBadRequest, 40000, err.Error())
	case errors.Is(err, errs.ErrInvalidToken):
		httpx.Fail(c, http.StatusUnauthorized, 40100, "invalid token")
	case errors.Is(err, errs.ErrTokenExpired):
		httpx.Fail(c, http.StatusUnauthorized, 40101, "token expired")
	case errors.Is(err, errs.ErrInvalidCredentials):
		httpx.Fail(c, http.StatusUnauthorized, 40102, "invalid email or password")
	case errors.Is(err, errs.ErrForbidden), errors.Is(err, errs.ErrUserInactive):
		httpx.Fail(c, http.StatusForbidden, 40300, err.Error())
	case errors.Is(err, errs.ErrNotFound), errors.Is(err, errs.ErrUserNotFound):
		httpx.Fail(c, http.StatusNotFound, 40400, err.Error())
	case errors.Is(err, errs.ErrConflict), errors.Is(err, errs.ErrEmailTaken):
		httpx.Fail(c, http.StatusConflict, 40900, err.Error())
	default:
		// Generic message — don't leak the wrap chain. The full error
		// is in the access log with the request_id so operators can
		// correlate from the client's response.
		httpx.Fail(c, http.StatusInternalServerError, 50000, "internal server error")
	}
}
