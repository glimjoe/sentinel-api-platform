// Package httpx provides a unified response envelope for the Sentinel API.
//
// All HTTP handlers respond through OK() or Fail() so clients see a stable
// shape: { code, message, data, request_id }. The request_id is propagated
// from middleware.GetRequestID so logs and clients can correlate a failure
// to its access-log line.
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
package httpx

import (
	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/middleware"
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
		RequestID: middleware.GetRequestID(c),
	})
}

// Fail writes an error envelope. httpStatus is the HTTP status code; code is
// the application-level error code; message is shown to the caller.
func Fail(c *gin.Context, httpStatus, code int, message string) {
	c.JSON(httpStatus, Envelope{
		Code:      code,
		Message:   message,
		RequestID: middleware.GetRequestID(c),
	})
}