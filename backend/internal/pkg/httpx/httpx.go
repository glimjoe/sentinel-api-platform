// Package httpx provides a unified response envelope for the Sentinel API.
//
// All HTTP handlers respond through OK() or Fail() so clients see a stable
// shape: { code, message, data, request_id }. The request_id is propagated
// from the gin context (placed there by middleware.RequestID) so logs and
// clients can correlate a failure to its access-log line.
//
// Convention:
//   - OK = success; code is 0, message is "ok", data carries the payload.
//   - Fail = error; httpStatus is the HTTP-level status (400/401/403/404/409/422/500),
//     code is the application-level error code (see internal/pkg/errs/codes.go —
//     added in Phase 5a), message is human-readable.
//
// Why envelope: the previous (Phase 1) handlers returned ad-hoc gin.H{"error": ...}
// bodies, which forced the frontend to branch on shape. One unified shape means
// one response interceptor on the client.
//
// Why requestIDFromContext is inlined rather than imported from middleware:
// middleware/ will import this package (RBAC middleware writes through
// httpx.Fail). If httpx imported middleware in return, the two packages would
// form an import cycle. The helper is one line so duplication is cheaper than
// a third package for it.
package httpx

import (
	"github.com/gin-gonic/gin"
)

// Envelope is the wire contract every Sentinel API response conforms to.
// Data carries `omitempty` so OK(c, nil) emits no "data" key at all,
// distinguishing "no payload" from "payload is null".
type Envelope struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"request_id"`
}

// OK writes a 200 envelope. data may be nil (in which case the data key is
// omitted from the JSON body — see Envelope.Data tag).
func OK(c *gin.Context, data any) {
	c.JSON(200, Envelope{
		Code:      0,
		Message:   "ok",
		Data:      data,
		RequestID: requestIDFromContext(c),
	})
}

// Fail writes an error envelope. httpStatus is the HTTP status code; code is
// the application-level error code; message is shown to the caller.
func Fail(c *gin.Context, httpStatus, code int, message string) {
	c.JSON(httpStatus, Envelope{
		Code:      code,
		Message:   message,
		RequestID: requestIDFromContext(c),
	})
}

// requestIDFromContext reads the request id placed by middleware.RequestID.
// Returns "" if absent — the test suite relies on this graceful fall-through
// rather than panicking, since the helper may be invoked before the
// request_id middleware runs in edge-case wiring.
func requestIDFromContext(c *gin.Context) string {
	if v, ok := c.Get("request_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}