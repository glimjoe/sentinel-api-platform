package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/id"
)

type TestResultRepo struct{ db *gorm.DB }

func NewTestResultRepo(db *gorm.DB) *TestResultRepo { return &TestResultRepo{db: db} }

func (r *TestResultRepo) Create(ctx context.Context, tr *model.TestResult) error {
	if tr.ID == "" { tr.ID = id.New() }
	tr.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(tr).Error
}

func (r *TestResultRepo) ListByRun(ctx context.Context, runID string) ([]*model.TestResult, error) {
	var list []*model.TestResult
	err := r.db.WithContext(ctx).Where("run_id = ?", runID).Order("created_at ASC").Find(&list).Error
	return list, err
}
