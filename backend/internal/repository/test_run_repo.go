package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/id"
)

type TestRunRepo struct{ db *gorm.DB }

func NewTestRunRepo(db *gorm.DB) *TestRunRepo { return &TestRunRepo{db: db} }

func (r *TestRunRepo) Create(ctx context.Context, run *model.TestRun) error {
	if run.ID == "" { run.ID = id.New() }
	run.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(run).Error
}

func (r *TestRunRepo) FindByID(ctx context.Context, id string) (*model.TestRun, error) {
	var run model.TestRun
	if err := r.db.WithContext(ctx).First(&run, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *TestRunRepo) ListByProject(ctx context.Context, projectID string) ([]*model.TestRun, error) {
	var list []*model.TestRun
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Order("created_at DESC").Find(&list).Error
	return list, err
}

func (r *TestRunRepo) Update(ctx context.Context, id string, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.TestRun{}).Where("id = ?", id).Updates(fields).Error
}
