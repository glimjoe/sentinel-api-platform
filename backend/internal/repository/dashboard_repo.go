package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/glimjoe/sentinel-api-platform/internal/service"
)

type DashboardRepo struct {
	db *gorm.DB
}

func NewDashboardRepo(db *gorm.DB) *DashboardRepo {
	return &DashboardRepo{db: db}
}

var _ service.DashboardStore = (*DashboardRepo)(nil)

func (r *DashboardRepo) CountProjects(ctx context.Context) (int, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("projects").Count(&n).Error
	return int(n), err
}

func (r *DashboardRepo) CountAPIs(ctx context.Context) (int, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("apis").Count(&n).Error
	return int(n), err
}

func (r *DashboardRepo) CountMockRules(ctx context.Context) (int, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("mock_rules").Count(&n).Error
	return int(n), err
}

func (r *DashboardRepo) CountTestCases(ctx context.Context) (int, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("test_cases").Count(&n).Error
	return int(n), err
}

func (r *DashboardRepo) ListRecentRuns(ctx context.Context, limit int) ([]service.RunSummary, error) {
	type row struct {
		ID        string `gorm:"column:id"`
		Name      string `gorm:"column:name"`
		Status    string `gorm:"column:status"`
		Passed    int    `gorm:"column:passed"`
		Failed    int    `gorm:"column:failed"`
		Errored   int    `gorm:"column:errored"`
		Skipped   int    `gorm:"column:skipped"`
		Total     int    `gorm:"column:total"`
		CreatedAt string `gorm:"column:created_at"`
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Table("test_runs").
		Order("created_at DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]service.RunSummary, len(rows))
	for i, r := range rows {
		out[i] = service.RunSummary{
			ID:     r.ID,
			Name:   r.Name,
			Status: r.Status,
			Passed: r.Passed,
			Failed: r.Failed,
			Errored: r.Errored,
			Skipped: r.Skipped,
			Total:  r.Total,
		}
	}
	return out, nil
}

func (r *DashboardRepo) CountResultsByStatus(ctx context.Context) (map[string]int, error) {
	type row struct {
		Status string `gorm:"column:status"`
		Count  int    `gorm:"column:count"`
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Table("test_results").
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := map[string]int{"pass": 0, "fail": 0, "error": 0, "skip": 0}
	for _, r := range rows {
		out[r.Status] = r.Count
	}
	return out, nil
}
