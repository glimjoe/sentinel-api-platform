// Package repository — ProjectMemberRepo (M2-F.E).
//
// ProjectMemberRepo persists rows in the project_members join table.
// The methods here satisfy the projectMemberStore interface declared in
// service/project_service.go so ProjectService can be unit-tested with a
// fake. The real impl is gorm-backed; the GORM error → errs mapping
// mirrors project_repo.go (gorm.ErrRecordNotFound → errs.ErrNotFound).
package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

// ProjectMemberRepo persists project_members rows.
type ProjectMemberRepo struct {
	db *gorm.DB
}

// NewProjectMemberRepo constructs a ProjectMemberRepo bound to db.
func NewProjectMemberRepo(db *gorm.DB) *ProjectMemberRepo {
	return &ProjectMemberRepo{db: db}
}

// Add inserts a new project_members row. Duplicate (project_id, user_id)
// returns errs.ErrConflict (mapped from MySQL 1062 by the driver).
func (r *ProjectMemberRepo) Add(ctx context.Context, m *model.ProjectMember) error {
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("add project member: %w", err)
	}
	return nil
}

// Find returns the member row for (projectID, userID), or
// errs.ErrNotFound wrapped. Used by ProjectService.RoleFor to look up
// the caller's role when they are not the project owner.
func (r *ProjectMemberRepo) Find(ctx context.Context, projectID, userID string) (*model.ProjectMember, error) {
	var m model.ProjectMember
	if err := r.db.WithContext(ctx).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("project_member_repo: %w", errs.ErrNotFound)
		}
		return nil, fmt.Errorf("find project member: %w", err)
	}
	return &m, nil
}

// ListByProject returns all members of a project. Empty slice (not
// nil) is the service's responsibility — this method returns whatever
// GORM gives it.
func (r *ProjectMemberRepo) ListByProject(ctx context.Context, projectID string) ([]*model.ProjectMember, error) {
	var list []*model.ProjectMember
	if err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list project members: %w", err)
	}
	return list, nil
}

// Remove deletes the (projectID, userID) row. Missing row is not an
// error (GORM returns RowsAffected=0 silently).
func (r *ProjectMemberRepo) Remove(ctx context.Context, projectID, userID string) error {
	if err := r.db.WithContext(ctx).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Delete(&model.ProjectMember{}).Error; err != nil {
		return fmt.Errorf("remove project member: %w", err)
	}
	return nil
}
