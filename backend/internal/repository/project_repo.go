// Package repository — ProjectRepo (Phase 2 M1).
//
// ProjectRepo persists Project + ProjectMember rows. The engine only needs
// FindBySlug; the rest is wired in M2 when the project CRUD API is built.
package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

// ProjectRepo persists projects and project members.
type ProjectRepo struct {
	db *gorm.DB
}

// NewProjectRepo constructs a ProjectRepo bound to db.
func NewProjectRepo(db *gorm.DB) *ProjectRepo {
	return &ProjectRepo{db: db}
}

// Create inserts a new project row.
func (r *ProjectRepo) Create(ctx context.Context, p *model.Project) error {
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	return nil
}

// FindByID returns the project with the given id, or errs.ErrNotFound wrapped.
func (r *ProjectRepo) FindByID(ctx context.Context, id string) (*model.Project, error) {
	var p model.Project
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("project_repo: %w", errs.ErrNotFound)
		}
		return nil, fmt.Errorf("find project by id: %w", err)
	}
	return &p, nil
}

// FindBySlug returns the project with the given slug, or errs.ErrNotFound
// wrapped. This is the hot-path lookup for the public /mock/:slug/* route.
func (r *ProjectRepo) FindBySlug(ctx context.Context, slug string) (*model.Project, error) {
	var p model.Project
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("project_repo: %w", errs.ErrNotFound)
		}
		return nil, fmt.Errorf("find project by slug: %w", err)
	}
	return &p, nil
}

// ListByOwner returns all projects owned by the given user id. Used in M2 by
// the project list handler. For M1 this is unused; included for completeness.
func (r *ProjectRepo) ListByOwner(ctx context.Context, ownerID string) ([]*model.Project, error) {
	var ps []*model.Project
	if err := r.db.WithContext(ctx).Where("owner_id = ?", ownerID).Find(&ps).Error; err != nil {
		return nil, fmt.Errorf("list projects by owner: %w", err)
	}
	return ps, nil
}

// AddMember inserts a row into project_members. The composite PK enforces
// uniqueness; GORM will return a duplicate-key error on collision which the
// service layer maps to errs.ErrConflict.
func (r *ProjectRepo) AddMember(ctx context.Context, m *model.ProjectMember) error {
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("add project member: %w", err)
	}
	return nil
}
