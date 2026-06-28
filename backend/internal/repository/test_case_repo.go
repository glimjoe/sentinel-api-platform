package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/id"
)

type TestCaseRepo struct{ db *gorm.DB }

func NewTestCaseRepo(db *gorm.DB) *TestCaseRepo { return &TestCaseRepo{db: db} }

func (r *TestCaseRepo) Create(ctx context.Context, tc *model.TestCase) error {
	if tc.ID == "" { tc.ID = id.New() }
	now := time.Now()
	tc.CreatedAt = now
	tc.UpdatedAt = now
	return r.db.WithContext(ctx).Create(tc).Error
}

func (r *TestCaseRepo) FindByID(ctx context.Context, id string) (*model.TestCase, error) {
	var tc model.TestCase
	if err := r.db.WithContext(ctx).First(&tc, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &tc, nil
}

func (r *TestCaseRepo) ListByProject(ctx context.Context, projectID string) ([]*model.TestCase, error) {
	var list []*model.TestCase
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Order("priority ASC").Find(&list).Error
	return list, err
}

func (r *TestCaseRepo) Update(ctx context.Context, id string, fields map[string]any) error {
	fields["updated_at"] = time.Now()
	return r.db.WithContext(ctx).Model(&model.TestCase{}).Where("id = ?", id).Updates(fields).Error
}

func (r *TestCaseRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.TestCase{}, "id = ?", id).Error
}
