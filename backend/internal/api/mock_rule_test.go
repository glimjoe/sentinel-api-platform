// Package api — mock_rule handler tests (RED, M2-F.C).
//
// 7 endpoints under test:
//   POST   /api/v1/rules                     CreateRule
//   GET    /api/v1/rules/:rid                GetRule
//   PATCH  /api/v1/rules/:rid                UpdateRule
//   DELETE /api/v1/rules/:rid                DeleteRule
//   GET    /api/v1/apis/:apiId/rules         ListRules
//   POST   /api/v1/rules/:rid/hits           RecordHit
//   GET    /api/v1/rules/:rid/hits           ListHits
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/httpx"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type fakeMockRuleService struct {
	CreateRuleFunc func(ctx context.Context, callerID, projectID, apiID string, spec CreateRuleSpec) (*model.MockRule, error)
	GetRuleFunc    func(ctx context.Context, id string) (*model.MockRule, error)
	UpdateRuleFunc func(ctx context.Context, callerID, projectID, ruleID string, fields map[string]any) (*model.MockRule, error)
	DeleteRuleFunc func(ctx context.Context, callerID, projectID, ruleID string) error
	ListRulesFunc  func(ctx context.Context, apiID string) ([]*model.MockRule, error)
	RecordHitFunc  func(ctx context.Context, ruleID string) error
	ListHitsFunc   func(ctx context.Context, ruleID string) (int64, error)
}

func (f *fakeMockRuleService) CreateRule(ctx context.Context, callerID, projectID, apiID string, spec CreateRuleSpec) (*model.MockRule, error) {
	return f.CreateRuleFunc(ctx, callerID, projectID, apiID, spec)
}

func (f *fakeMockRuleService) GetRule(ctx context.Context, id string) (*model.MockRule, error) {
	return f.GetRuleFunc(ctx, id)
}

func (f *fakeMockRuleService) UpdateRule(ctx context.Context, callerID, projectID, ruleID string, fields map[string]any) (*model.MockRule, error) {
	return f.UpdateRuleFunc(ctx, callerID, projectID, ruleID, fields)
}

func (f *fakeMockRuleService) DeleteRule(ctx context.Context, callerID, projectID, ruleID string) error {
	return f.DeleteRuleFunc(ctx, callerID, projectID, ruleID)
}

func (f *fakeMockRuleService) ListRules(ctx context.Context, apiID string) ([]*model.MockRule, error) {
	return f.ListRulesFunc(ctx, apiID)
}

func (f *fakeMockRuleService) RecordHit(ctx context.Context, ruleID string) error {
	return f.RecordHitFunc(ctx, ruleID)
}

func (f *fakeMockRuleService) ListHits(ctx context.Context, ruleID string) (int64, error) {
	return f.ListHitsFunc(ctx, ruleID)
}

func newMockRuleTestEngine(t *testing.T, svc *fakeMockRuleService, callerID string) *gin.Engine {
	t.Helper()
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if callerID != "" {
			c.Set("user_id", callerID)
		}
		c.Next()
	})
	h := &MockRuleHandler{svc: svc}
	r.POST("/api/v1/rules", h.CreateRule)
	r.GET("/api/v1/rules/:rid", h.GetRule)
	r.PATCH("/api/v1/rules/:rid", h.UpdateRule)
	r.DELETE("/api/v1/rules/:rid", h.DeleteRule)
	r.GET("/api/v1/apis/:apiId/rules", h.ListRules)
	r.POST("/api/v1/rules/:rid/hits", h.RecordHit)
	r.GET("/api/v1/rules/:rid/hits", h.ListHits)
	return r
}

func decodeMockRuleEnvelope(t *testing.T, body []byte) httpx.Envelope {
	t.Helper()
	var env httpx.Envelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode envelope: %v (body=%s)", err, body)
	}
	return env
}

func doCreateRule(t *testing.T, r *gin.Engine, name, matchJSON string, responseStatus int) *httptest.ResponseRecorder {
	t.Helper()
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if matchJSON != "" {
		body["match_json"] = json.RawMessage(matchJSON)
	}
	if responseStatus != 0 {
		body["response_status"] = responseStatus
	}
	bs, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules?api_id=01HA", bytes.NewReader(bs))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestCreateRule_HappyPath(t *testing.T) {
	svc := &fakeMockRuleService{
		CreateRuleFunc: func(_ context.Context, callerID, projectID, apiID string, spec CreateRuleSpec) (*model.MockRule, error) {
			if callerID != "u-engineer" {
				t.Errorf("callerID = %q, want u-engineer", callerID)
			}
			if apiID != "01HA" {
				t.Errorf("apiID = %q, want 01HA", apiID)
			}
			return &model.MockRule{ID: "01HMR", APIID: apiID, Name: spec.Name, MatchJSON: spec.MatchJSON}, nil
		},
	}
	r := newMockRuleTestEngine(t, svc, "u-engineer")
	w := doCreateRule(t, r, "rule-1", `{"query":{"a":"1"}}`, 200)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestCreateRule_Forbidden(t *testing.T) {
	svc := &fakeMockRuleService{
		CreateRuleFunc: func(_ context.Context, _, _, _ string, _ CreateRuleSpec) (*model.MockRule, error) {
			return nil, errs.ErrForbidden
		},
	}
	r := newMockRuleTestEngine(t, svc, "u-viewer")
	w := doCreateRule(t, r, "rule-1", `{"query":{}}`, 200)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

func TestGetRule_HappyPath(t *testing.T) {
	svc := &fakeMockRuleService{
		GetRuleFunc: func(_ context.Context, id string) (*model.MockRule, error) {
			return &model.MockRule{ID: id, Name: "rule-1"}, nil
		},
	}
	r := newMockRuleTestEngine(t, svc, "u-viewer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/rules/01HMR", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestGetRule_NotFound(t *testing.T) {
	svc := &fakeMockRuleService{
		GetRuleFunc: func(_ context.Context, _ string) (*model.MockRule, error) {
			return nil, errs.ErrNotFound
		},
	}
	r := newMockRuleTestEngine(t, svc, "u-viewer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/rules/missing", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateRule_HappyPath(t *testing.T) {
	svc := &fakeMockRuleService{
		UpdateRuleFunc: func(_ context.Context, callerID, projectID, ruleID string, fields map[string]any) (*model.MockRule, error) {
			return &model.MockRule{ID: ruleID, Name: "rule-1", Priority: 50}, nil
		},
	}
	r := newMockRuleTestEngine(t, svc, "u-engineer")
	body := map[string]any{"priority": 50}
	bs, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/rules/01HMR", bytes.NewReader(bs))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteRule_HappyPath(t *testing.T) {
	svc := &fakeMockRuleService{
		DeleteRuleFunc: func(_ context.Context, callerID, projectID, ruleID string) error {
			return nil
		},
	}
	r := newMockRuleTestEngine(t, svc, "u-admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/rules/01HMR", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestListRules_HappyPath(t *testing.T) {
	svc := &fakeMockRuleService{
		ListRulesFunc: func(_ context.Context, apiID string) ([]*model.MockRule, error) {
			return []*model.MockRule{
				{ID: "01HMR-1", APIID: apiID, Name: "lo"},
				{ID: "01HMR-2", APIID: apiID, Name: "hi"},
			}, nil
		},
	}
	r := newMockRuleTestEngine(t, svc, "u-viewer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/apis/01HA/rules", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestRecordHit_HappyPath(t *testing.T) {
	called := false
	svc := &fakeMockRuleService{
		RecordHitFunc: func(_ context.Context, ruleID string) error {
			called = true
			return nil
		},
	}
	r := newMockRuleTestEngine(t, svc, "u-engineer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules/01HMR/hits", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if !called {
		t.Error("RecordHit not called")
	}
}

func TestListHits_HappyPath(t *testing.T) {
	svc := &fakeMockRuleService{
		ListHitsFunc: func(_ context.Context, ruleID string) (int64, error) {
			return 42, nil
		},
	}
	r := newMockRuleTestEngine(t, svc, "u-viewer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/rules/01HMR/hits", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	env := decodeMockRuleEnvelope(t, w.Body.Bytes())
	if !strings.Contains(string(mustMarshal(env.Data)), "42") {
		t.Errorf("data missing 42: %s", env.Data)
	}
}

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
