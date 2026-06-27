// Package repository — APIRepo (Phase 2 M1).
//
// APIRepo persists API rows. The engine uses ListByProject to find candidate
// rules for a path; CRUD endpoints are wired in M2.
package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

// APIRepo persists API rows.
type APIRepo struct {
	db *gorm.DB
}

// NewAPIRepo constructs an APIRepo bound to db.
func NewAPIRepo(db *gorm.DB) *APIRepo {
	return &APIRepo{db: db}
}

// Create inserts a new API row. The (project_id, method, path) unique index
// causes a duplicate-key error if the same endpoint is registered twice;
// service layer maps that to errs.ErrConflict.
func (r *APIRepo) Create(ctx context.Context, a *model.API) error {
	if err := r.db.WithContext(ctx).Create(a).Error; err != nil {
		return fmt.Errorf("create api: %w", err)
	}
	return nil
}

// FindByID returns the API with the given id, or errs.ErrNotFound wrapped.
func (r *APIRepo) FindByID(ctx context.Context, id string) (*model.API, error) {
	var a model.API
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&a).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("api_repo: %w", errs.ErrNotFound)
		}
		return nil, fmt.Errorf("find api by id: %w", err)
	}
	return &a, nil
}

// ListByProject returns all APIs in the given project. The engine uses this
// to resolve (method, path) → api_id when scoring rules.
func (r *APIRepo) ListByProject(ctx context.Context, projectID string) ([]*model.API, error) {
	var as []*model.API
	if err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Find(&as).Error; err != nil {
		return nil, fmt.Errorf("list apis by project: %w", err)
	}
	return as, nil
}

// Delete removes an API by id. Cascades to mock_rules via FK.
func (r *APIRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.API{}).Error; err != nil {
		return fmt.Errorf("delete api: %w", err)
	}
	return nil
}
