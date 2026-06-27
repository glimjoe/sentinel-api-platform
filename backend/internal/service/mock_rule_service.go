// Package service — mock_rule_service stub (M2-E TDD RED, revised).
//
// MockRuleService handles CRUD on the mock_rules table. RBAC checks are
// delegated to the shared projectRoleChecker; the caller's caller
// (handler) supplies projectID explicitly so the service doesn't need to
// walk the api → project chain itself.
//
// Persistence contract (mockRuleStore) mirrors MockRuleRepo (Phase 2 M1):
// Create, FindByID, ListByAPI, Update(id, fields map), IncrementHitCount.
// No Delete yet — MockRuleRepo lacks it, will land when M2-F wires the
// DELETE handler.
//
// No ListByProject here — model.MockRule doesn't carry project_id, and
// MockRuleRepo's ListByProject does the JOIN internally. The engine calls
// repo.ListByProject directly; this service is for CRUD via the handler.
//
// M2 TDD: this file is the RED-phase stub. Methods land once
// mock_rule_service_test.go fails for the expected reason (undefined:
// MockRuleService methods).
package service

import (
	"context"
	"encoding/json"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

// mockRuleStore is the persistence contract MockRuleService needs from
// the mock_rules table. Mirrors the five methods on repository.MockRuleRepo
// (Create, FindByID, ListByAPI, Update, IncrementHitCount).
type mockRuleStore interface {
	Create(ctx context.Context, r *model.MockRule) error
	FindByID(ctx context.Context, id string) (*model.MockRule, error)
	ListByAPI(ctx context.Context, apiID string) ([]*model.MockRule, error)
	// Update applies the given field map to the rule with id. The repo
	// implementation uses gorm Updates(map) so zero-value semantics differ
	// from Save. See mock_rule_repo.go Update for details.
	Update(ctx context.Context, id string, fields map[string]any) error
	IncrementHitCount(ctx context.Context, id string) error
}

// CreateRuleSpec bundles the optional rule fields so the Create signature
// stays readable. Priority defaults to 100 if zero; ResponseStatus defaults
// to 200 if zero; DelayMs defaults to 0.
type CreateRuleSpec struct {
	Name            string
	MatchJSON       json.RawMessage
	ResponseStatus  int
	ResponseHeaders json.RawMessage
	ResponseBody    json.RawMessage
	ExtractorJSON   json.RawMessage
	Priority        int
	DelayMs         int
}

// MockRuleService is the entry point for mock rule CRUD business logic.
type MockRuleService struct{}

// NewMockRuleService wires a MockRuleService. Caller owns the lifetime.
func NewMockRuleService(rules mockRuleStore, roles projectRoleChecker) *MockRuleService {
	_ = rules
	_ = roles
	return &MockRuleService{}
}