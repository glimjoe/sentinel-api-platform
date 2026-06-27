// Package repository — MockRuleRepo (Phase 2 M1).
//
// MockRuleRepo persists mock_rules. The engine uses ListByProject (across all
// APIs in a project) to score incoming requests; the rest is M2 CRUD.
package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

// MockRuleRepo persists mock_rules.
type MockRuleRepo struct {
	db *gorm.DB
}

// NewMockRuleRepo constructs a MockRuleRepo bound to db.
func NewMockRuleRepo(db *gorm.DB) *MockRuleRepo {
	return &MockRuleRepo{db: db}
}

// Create inserts a new mock_rule row.
func (r *MockRuleRepo) Create(ctx context.Context, rule *model.MockRule) error {
	if err := r.db.WithContext(ctx).Create(rule).Error; err != nil {
		return fmt.Errorf("create mock_rule: %w", err)
	}
	return nil
}

// FindByID returns the rule with the given id, or errs.ErrNotFound wrapped.
func (r *MockRuleRepo) FindByID(ctx context.Context, id string) (*model.MockRule, error) {
	var rule model.MockRule
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("mock_rule_repo: %w", errs.ErrNotFound)
		}
		return nil, fmt.Errorf("find mock_rule by id: %w", err)
	}
	return &rule, nil
}

// ListByAPI returns enabled rules for a single API. Unused by the engine
// (which lists across the whole project) but useful for the rule-list UI in M2.
func (r *MockRuleRepo) ListByAPI(ctx context.Context, apiID string) ([]*model.MockRule, error) {
	var rules []*model.MockRule
	if err := r.db.WithContext(ctx).
		Where("api_id = ? AND enabled = 1", apiID).
		Order("priority ASC, id ASC").
		Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("list mock_rules by api: %w", err)
	}
	return rules, nil
}

// ListByProject returns all enabled mock rules for a project, across all
// APIs. This is the engine's hot-path query — the matcher scores each rule
// against the incoming request and picks the highest. Ordering is priority
// ASC, then id ASC (the lexicographic tie-break per plan §6.6 + ADR edit).
func (r *MockRuleRepo) ListByProject(ctx context.Context, projectID string) ([]*model.MockRule, error) {
	var rules []*model.MockRule
	// Two-step join: rules → apis where apis.project_id = ? AND rules.enabled = 1.
	// The idx_mock_rules_api_enabled_priority index makes the rule-side lookup
	// cheap; the apis.id is PK so the join is also indexed.
	err := r.db.WithContext(ctx).
		Table("mock_rules AS r").
		Select("r.*").
		Joins("JOIN apis AS a ON a.id = r.api_id").
		Where("a.project_id = ? AND r.enabled = 1", projectID).
		Order("r.priority ASC, r.id ASC").
		Find(&rules).Error
	if err != nil {
		return nil, fmt.Errorf("list mock_rules by project: %w", err)
	}
	return rules, nil
}

// IncrementHitCount atomically bumps hit_count by 1. Called by the engine
// after a successful match (M2 wires the engine → recorder call; for M1 the
// engine is the caller directly).
func (r *MockRuleRepo) IncrementHitCount(ctx context.Context, id string) error {
	res := r.db.WithContext(ctx).
		Model(&model.MockRule{}).
		Where("id = ?", id).
		Update("hit_count", gorm.Expr("hit_count + 1"))
	if res.Error != nil {
		return fmt.Errorf("increment hit_count: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("mock_rule_repo: %w", errs.ErrNotFound)
	}
	return nil
}

// Update applies non-zero fields from rule to the row with the given id. Uses
// gorm's Updates with a map to avoid zero-value issues with the int/bool fields.
func (r *MockRuleRepo) Update(ctx context.Context, id string, fields map[string]any) error {
	res := r.db.WithContext(ctx).
		Model(&model.MockRule{}).
		Where("id = ?", id).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Updates(fields)
	if res.Error != nil {
		return fmt.Errorf("update mock_rule: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("mock_rule_repo: %w", errs.ErrNotFound)
	}
	return nil
}
