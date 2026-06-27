// Package service — MockRuleService (M2-E).
//
// MockRuleService handles CRUD on the mock_rules table — the rows that
// configure how the engine responds to a matching request. RBAC checks
// are delegated to the shared projectRoleChecker; the caller's caller
// (handler) supplies projectID explicitly so the service doesn't need to
// walk the api → project chain itself.
//
// Authorization:
//   - Create / Update: admin OR engineer (engineers tune rules in real time)
//   - FindByID / ListByAPI: no RBAC at this layer (handler middleware enforces)
//   - IncrementHitCount: internal-only — the engine calls it after each match
//
// No Delete yet — MockRuleRepo lacks the method; the DELETE handler in M2-F
// will land with both the repo extension and the service method together.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/id"
)

// mockRuleStore is the persistence contract MockRuleService needs from
// the mock_rules table. Mirrors the five methods on repository.MockRuleRepo.
type mockRuleStore interface {
	Create(ctx context.Context, r *model.MockRule) error
	FindByID(ctx context.Context, id string) (*model.MockRule, error)
	ListByAPI(ctx context.Context, apiID string) ([]*model.MockRule, error)
	Update(ctx context.Context, id string, fields map[string]any) error
	IncrementHitCount(ctx context.Context, id string) error
}

// CreateRuleSpec bundles the optional rule fields so the Create signature
// stays readable. Fields are zero-valued if absent; priority defaults to
// 100, responseStatus to 200.
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

// MockRuleService wires the mock rule CRUD business logic.
type MockRuleService struct {
	rules mockRuleStore
	roles projectRoleChecker
}

// NewMockRuleService constructs a MockRuleService. Caller owns the lifetime.
func NewMockRuleService(rules mockRuleStore, roles projectRoleChecker) *MockRuleService {
	return &MockRuleService{rules: rules, roles: roles}
}

// Create inserts a new rule for the given API. Caller must be admin or
// engineer on projectID.
//
// Errors:
//   - errs.ErrForbidden  — caller is not admin/engineer
//   - errs.ErrBadRequest — empty name or invalid match_json
//   - errs.ErrConflict   — (api_id, name) already exists
func (s *MockRuleService) Create(ctx context.Context, callerID, projectID, apiID string, spec CreateRuleSpec) (*model.MockRule, error) {
	callerRole, err := s.roles.RoleFor(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if callerRole != model.ProjectRoleAdmin && callerRole != model.ProjectRoleEngineer {
		return nil, errs.ErrForbidden
	}

	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", errs.ErrBadRequest)
	}
	if len(spec.MatchJSON) == 0 {
		return nil, fmt.Errorf("%w: match_json is required", errs.ErrBadRequest)
	}
	if !json.Valid(spec.MatchJSON) {
		return nil, fmt.Errorf("%w: match_json is not valid JSON", errs.ErrBadRequest)
	}

	status := spec.ResponseStatus
	if status == 0 {
		status = 200
	}
	priority := spec.Priority
	if priority == 0 {
		priority = 100
	}

	r := &model.MockRule{
		ID:                  id.New(),
		APIID:               apiID,
		Name:                name,
		MatchJSON:           spec.MatchJSON,
		ResponseStatus:      status,
		ResponseHeadersJSON: spec.ResponseHeaders,
		ResponseBodyJSON:    spec.ResponseBody,
		ExtractorJSON:       spec.ExtractorJSON,
		Priority:            priority,
		DelayMs:             spec.DelayMs,
		Enabled:             true,
	}
	if err := s.rules.Create(ctx, r); err != nil {
		if errors.Is(err, errs.ErrConflict) {
			return nil, errs.ErrConflict
		}
		return nil, fmt.Errorf("create mock_rule: %w", err)
	}
	return r, nil
}

// FindByID returns the rule with the given id, or errs.ErrNotFound.
func (s *MockRuleService) FindByID(ctx context.Context, id string) (*model.MockRule, error) {
	r, err := s.rules.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("find mock_rule: %w", err)
	}
	return r, nil
}

// ListByAPI returns the rules for the given API. Empty slice, not nil,
// when none — so handlers can iterate without a nil check.
func (s *MockRuleService) ListByAPI(ctx context.Context, apiID string) ([]*model.MockRule, error) {
	list, err := s.rules.ListByAPI(ctx, apiID)
	if err != nil {
		return nil, fmt.Errorf("list mock_rules: %w", err)
	}
	if list == nil {
		list = []*model.MockRule{}
	}
	return list, nil
}

// Update applies the field map to the rule and returns the refreshed row.
// Caller must be admin or engineer on projectID.
//
// Errors:
//   - errs.ErrForbidden — caller is not admin/engineer
//   - errs.ErrNotFound  — rule doesn't exist
func (s *MockRuleService) Update(ctx context.Context, callerID, projectID, ruleID string, fields map[string]any) (*model.MockRule, error) {
	callerRole, err := s.roles.RoleFor(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if callerRole != model.ProjectRoleAdmin && callerRole != model.ProjectRoleEngineer {
		return nil, errs.ErrForbidden
	}
	if err := s.rules.Update(ctx, ruleID, fields); err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("update mock_rule: %w", err)
	}
	return s.FindByID(ctx, ruleID)
}

// IncrementHitCount bumps the rule's hit counter by 1. Called by the
// engine after a successful match.
func (s *MockRuleService) IncrementHitCount(ctx context.Context, ruleID string) error {
	if err := s.rules.IncrementHitCount(ctx, ruleID); err != nil {
		return fmt.Errorf("increment hit count: %w", err)
	}
	return nil
}