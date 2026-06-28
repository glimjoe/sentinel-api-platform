package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestHealthHandler_Healthz — GET /healthz returns 200 envelope with
// {ok:true, service, version}. Locks the new wire shape post-migration.
// Readyz is intentionally not tested here — it needs real DB+Redis or a
// substantial mock setup; it's covered by Phase 5a integration tests.
func TestHealthHandler_Healthz(t *testing.T) {
	r := gin.New()
	// HealthHandler is constructed with nil DB/Redis because Healthz
	// does not touch dependencies. (The Readyz handler would NPE on
	// nil; we mount only Healthz here.)
	h := &HealthHandler{DB: nil, Redis: nil}
	r.GET("/api/v1/healthz", h.Healthz)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 0 {
		t.Errorf("app code = %d, want 0 (OK)", env.Code)
	}
	raw, _ := json.Marshal(env.Data)
	for _, key := range []string{`"ok":true`, `"service":"sentinel"`, `"version"`} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("data missing %s: %s", key, raw)
		}
	}
}
