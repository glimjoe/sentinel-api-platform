package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/httpx"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// runWriteError wires a single GET endpoint that calls WriteError with the
// supplied error, then drives it. Returns the recorded response.
func runWriteError(t *testing.T, err error) *httptest.ResponseRecorder {
	t.Helper()
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		WriteError(c, err)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)
	return w
}

// decodeEnvelope parses a recorded response body as the httpx envelope.
// Helper is inlined so failures point at the specific test, not the helper.
func decodeEnvelope(t *testing.T, body []byte) httpx.Envelope {
	t.Helper()
	var env httpx.Envelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode envelope: %v (body=%s)", err, body)
	}
	return env
}

// TestWriteError_ErrBadRequest — generic 400 with code 40000.
func TestWriteError_ErrBadRequest(t *testing.T) {
	w := runWriteError(t, fmt.Errorf("name is required: %w", errs.ErrBadRequest))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 40000 {
		t.Errorf("app code = %d, want 40000", env.Code)
	}
	if env.Message == "" {
		t.Errorf("message is empty")
	}
}

// TestWriteError_ErrInvalidToken — 401 / 40100.
func TestWriteError_ErrInvalidToken(t *testing.T) {
	w := runWriteError(t, errs.ErrInvalidToken)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 40100 {
		t.Errorf("app code = %d, want 40100", env.Code)
	}
}

// TestWriteError_ErrTokenExpired — 401 / 40101.
func TestWriteError_ErrTokenExpired(t *testing.T) {
	w := runWriteError(t, errs.ErrTokenExpired)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 40101 {
		t.Errorf("app code = %d, want 40101", env.Code)
	}
}

// TestWriteError_ErrInvalidCredentials — 401 / 40102.
func TestWriteError_ErrInvalidCredentials(t *testing.T) {
	w := runWriteError(t, errs.ErrInvalidCredentials)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 40102 {
		t.Errorf("app code = %d, want 40102", env.Code)
	}
}

// TestWriteError_ErrForbidden — 403 / 40300 (same code as rbac's "not a member"
// deliberately — they're the same semantic class to clients).
func TestWriteError_ErrForbidden(t *testing.T) {
	w := runWriteError(t, fmt.Errorf("caller is not admin: %w", errs.ErrForbidden))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 40300 {
		t.Errorf("app code = %d, want 40300", env.Code)
	}
}

// TestWriteError_ErrNotFound — 404 / 40400 (same code as rbac's "project
// not found" — both surface "row missing").
func TestWriteError_ErrNotFound(t *testing.T) {
	w := runWriteError(t, errs.ErrNotFound)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 40400 {
		t.Errorf("app code = %d, want 40400", env.Code)
	}
}

// TestWriteError_ErrUserNotFound — 404 / 40400 (same code class as ErrNotFound;
// we don't distinguish "user missing" from "row missing" at the wire).
func TestWriteError_ErrUserNotFound(t *testing.T) {
	w := runWriteError(t, errs.ErrUserNotFound)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 40400 {
		t.Errorf("app code = %d, want 40400", env.Code)
	}
}

// TestWriteError_ErrConflict — 409 / 40900.
func TestWriteError_ErrConflict(t *testing.T) {
	w := runWriteError(t, fmt.Errorf("slug taken: %w", errs.ErrConflict))
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 40900 {
		t.Errorf("app code = %d, want 40900", env.Code)
	}
}

// TestWriteError_ErrEmailTaken — 409 / 40900 (same code as ErrConflict).
func TestWriteError_ErrEmailTaken(t *testing.T) {
	w := runWriteError(t, errs.ErrEmailTaken)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 40900 {
		t.Errorf("app code = %d, want 40900", env.Code)
	}
}

// TestWriteError_UnknownError — any other error must surface as 500 / 50000
// with a generic message. The full wrap chain is captured by AccessLog, so
// we don't try to unmangle it client-side.
func TestWriteError_UnknownError(t *testing.T) {
	w := runWriteError(t, errors.New("something exploded"))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 50000 {
		t.Errorf("app code = %d, want 50000", env.Code)
	}
	if env.Message != "internal server error" {
		t.Errorf("message = %q, want generic (must not leak details)", env.Message)
	}
}

// TestWriteError_EnvelopeShape — body must be {code, message, request_id}
// and must NOT contain a "data" key (Fail omits Data). Guards the wire
// contract that the frontend's response interceptor depends on.
func TestWriteError_EnvelopeShape(t *testing.T) {
	w := runWriteError(t, errs.ErrNotFound)

	// Re-parse as a generic map to check key presence/absence.
	var raw map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal: %v (body=%s)", err, w.Body.String())
	}
	if _, ok := raw["code"]; !ok {
		t.Errorf("envelope missing 'code' key: %s", w.Body.String())
	}
	if _, ok := raw["message"]; !ok {
		t.Errorf("envelope missing 'message' key: %s", w.Body.String())
	}
	if _, ok := raw["request_id"]; !ok {
		t.Errorf("envelope missing 'request_id' key: %s", w.Body.String())
	}
	if _, ok := raw["data"]; ok {
		t.Errorf("envelope should not contain 'data' key on error path: %s", w.Body.String())
	}
}
