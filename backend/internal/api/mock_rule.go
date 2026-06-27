// Package api — mock_rule handlers (M2-F.C).
//
// Endpoints:
//   POST   /api/v1/rules                     CreateRule     (engineer+)
//   GET    /api/v1/rules/:rid                GetRule        (viewer+)
//   PATCH  /api/v1/rules/:rid                UpdateRule     (engineer+)
//   DELETE /api/v1/rules/:rid                DeleteRule     (engineer+)
//   GET    /api/v1/apis/:apiId/rules         ListRules      (viewer+)
//   POST   /api/v1/rules/:rid/hits           RecordHit      (engine+)
//   GET    /api/v1/rules/:rid/hits           ListHits       (viewer+)
package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/middleware"
	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/httpx"
)

// CreateRuleSpec is the request body for CreateRule.
type CreateRuleSpec struct {
	Name             string          `json:"name"`
	MatchJSON        json.RawMessage `json:"match_json"`
	ResponseStatus   int             `json:"response_status"`
	ResponseHeaders  json.RawMessage `json:"response_headers"`
	ResponseBody     json.RawMessage `json:"response_body"`
	ExtractorJSON    json.RawMessage `json:"extractor_json"`
	Priority         int             `json:"priority"`
	DelayMs          int             `json:"delay_ms"`
}

// mockRuleService is the contract MockRuleHandler needs from the service
// layer. *service.MockRuleService satisfies it.
type mockRuleService interface {
	CreateRule(ctx context.Context, callerID, projectID, apiID string, spec CreateRuleSpec) (*model.MockRule, error)
	GetRule(ctx context.Context, id string) (*model.MockRule, error)
	UpdateRule(ctx context.Context, callerID, projectID, ruleID string, fields map[string]any) (*model.MockRule, error)
	DeleteRule(ctx context.Context, callerID, projectID, ruleID string) error
	ListRules(ctx context.Context, apiID string) ([]*model.MockRule, error)
	RecordHit(ctx context.Context, ruleID string) error
	ListHits(ctx context.Context, ruleID string) (int64, error)
}

// MockRuleHandler wires HTTP routes to the mock rule service.
type MockRuleHandler struct {
	svc mockRuleService
}

// NewMockRuleHandler constructs a MockRuleHandler. Caller owns the lifetime.
func NewMockRuleHandler(svc mockRuleService) *MockRuleHandler {
	return &MockRuleHandler{svc: svc}
}

// CreateRule handles POST /api/v1/rules. The engineer's RBAC is enforced
// at the service layer.
func (h *MockRuleHandler) CreateRule(c *gin.Context) {
	apiID := c.Query("api_id")
	if apiID == "" {
		httpx.Fail(c, http.StatusBadRequest, 40000, "api_id is required")
		return
	}
	var req CreateRuleSpec
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	callerID := c.GetString("user_id")
	rule, err := h.svc.CreateRule(c.Request.Context(), callerID, "", apiID, req)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, rule)
}

// GetRule handles GET /api/v1/rules/:rid.
func (h *MockRuleHandler) GetRule(c *gin.Context) {
	rule, err := h.svc.GetRule(c.Request.Context(), c.Param("rid"))
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, rule)
}

// UpdateRule handles PATCH /api/v1/rules/:rid.
func (h *MockRuleHandler) UpdateRule(c *gin.Context) {
	var fields map[string]any
	if err := c.ShouldBindJSON(&fields); err != nil {
		httpx.Fail(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	callerID := c.GetString("user_id")
	rule, err := h.svc.UpdateRule(c.Request.Context(), callerID, "", c.Param("rid"), fields)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, rule)
}

// DeleteRule handles DELETE /api/v1/rules/:rid.
func (h *MockRuleHandler) DeleteRule(c *gin.Context) {
	callerID := c.GetString("user_id")
	if err := h.svc.DeleteRule(c.Request.Context(), callerID, "", c.Param("rid")); err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, nil)
}

// ListRules handles GET /api/v1/apis/:apiId/rules.
func (h *MockRuleHandler) ListRules(c *gin.Context) {
	rules, err := h.svc.ListRules(c.Request.Context(), c.Param("apiId"))
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, rules)
}

// RecordHit handles POST /api/v1/rules/:rid/hits.
func (h *MockRuleHandler) RecordHit(c *gin.Context) {
	if err := h.svc.RecordHit(c.Request.Context(), c.Param("rid")); err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, nil)
}

// ListHits handles GET /api/v1/rules/:rid/hits.
func (h *MockRuleHandler) ListHits(c *gin.Context) {
	count, err := h.svc.ListHits(c.Request.Context(), c.Param("rid"))
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	httpx.OK(c, count)
}
