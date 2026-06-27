// Package httpx provides a unified response envelope for the Sentinel API.
//
// All HTTP handlers respond through OK() or Fail() so clients see a stable
// shape: { code, message, data, request_id }. The request_id is propagated
// from middleware.GetRequestID so logs and clients can correlate a failure
// to its access-log line.
//
// M2 TDD: this file is the RED-phase stub. Implementation lands once
// httpx_test.go fails for the expected reason (undefined: OK, Fail, Envelope).
package httpx