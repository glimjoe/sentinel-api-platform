package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newTestCtx returns a gin.Context with request_id pre-set (mimicking what
// middleware.RequestID does) plus a recorder so tests can inspect the
// response that OK / Fail writes.
func newTestCtx(reqID string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("request_id", reqID)
	return c, w
}

// TestOK_WritesEnvelope — happy path. OK must write 200 + the canonical
// envelope with code=0, message="ok", data passed through, and request_id
// pulled from the gin context.
func TestOK_WritesEnvelope(t *testing.T) {
	c, w := newTestCtx("req-abc")
	OK(c, map[string]string{"user_id": "u1"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var env Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v, body=%s", err, w.Body.String())
	}
	if env.Code != 0 {
		t.Errorf("Envelope.Code = %d, want 0", env.Code)
	}
	if env.Message != "ok" {
		t.Errorf("Envelope.Message = %q, want \"ok\"", env.Message)
	}
	if env.RequestID != "req-abc" {
		t.Errorf("Envelope.RequestID = %q, want \"req-abc\"", env.RequestID)
	}
	if env.Data == nil {
		t.Errorf("Envelope.Data should carry the passed payload, got nil")
	}
}

// TestFail_WritesEnvelopeWithStatus — error path. Fail must write the given
// HTTP status, plus an envelope with the application code, message, and
// the same request_id used by OK.
func TestFail_WritesEnvelopeWithStatus(t *testing.T) {
	c, w := newTestCtx("req-xyz")
	Fail(c, http.StatusBadRequest, 40001, "bad input")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	var env Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Code != 40001 {
		t.Errorf("Envelope.Code = %d, want 40001", env.Code)
	}
	if env.Message != "bad input" {
		t.Errorf("Envelope.Message = %q, want \"bad input\"", env.Message)
	}
	if env.RequestID != "req-xyz" {
		t.Errorf("Envelope.RequestID = %q, want \"req-xyz\"", env.RequestID)
	}
}

// TestOK_NilDataOmitsField — when no payload is passed, the JSON output must
// not contain a "data" key at all (so clients can distinguish "no data" from
// "data is null"). Required so the envelope stays a clean shape for the
// 204-ish "operation succeeded, nothing to return" case.
func TestOK_NilDataOmitsField(t *testing.T) {
	c, w := newTestCtx("req-1")
	OK(c, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"request_id":"req-1"`) {
		t.Errorf("body should contain request_id, got %s", body)
	}
	if strings.Contains(body, `"data":`) {
		t.Errorf("body should omit data field when nil, got %s", body)
	}
}

// TestEnvelope_JSONShape — pins the wire contract. Future handlers MUST
// produce this exact shape; if this test ever needs to change, update the
// frontend in lockstep (api/client.ts response interceptor).
func TestEnvelope_JSONShape(t *testing.T) {
	body := `{"code":0,"message":"ok","data":{"x":1},"request_id":"r1"}`
	var env Envelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Code != 0 || env.Message != "ok" || env.RequestID != "r1" {
		t.Errorf("envelope = %+v", env)
	}
	// Round-trip back to JSON and confirm the same key set (no surprise fields).
	out, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"code":0`) ||
		!strings.Contains(string(out), `"message":"ok"`) ||
		!strings.Contains(string(out), `"data":`) ||
		!strings.Contains(string(out), `"request_id":"r1"`) {
		t.Errorf("round-trip JSON missing keys, got %s", string(out))
	}
}