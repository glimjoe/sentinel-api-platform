package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/id"
)

// AiRepo persists AI usage records.
type AiRepo struct {
	db *gorm.DB
}

// NewAiRepo constructs an AiRepo bound to db.
func NewAiRepo(db *gorm.DB) *AiRepo { return &AiRepo{db: db} }

// RecordUsage inserts an AI usage entry.
func (r *AiRepo) RecordUsage(ctx context.Context, m *model.AiUsage) error {
	m.ID = id.New()
	m.CreatedAt = time.Now()
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("record ai usage: %w", err)
	}
	return nil
}

// GetDailyUsage returns total cost since start of today (UTC).
func (r *AiRepo) GetDailyUsage(ctx context.Context) (float64, error) {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	var total float64
	if err := r.db.WithContext(ctx).
		Model(&model.AiUsage{}).
		Where("created_at >= ?", start).
		Select("COALESCE(SUM(cost_usd), 0)").
		Scan(&total).Error; err != nil {
		return 0, fmt.Errorf("get daily usage: %w", err)
	}
	return total, nil
}

// GetMonthlyUsage returns total cost since start of current month (UTC).
func (r *AiRepo) GetMonthlyUsage(ctx context.Context) (float64, error) {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	var total float64
	if err := r.db.WithContext(ctx).
		Model(&model.AiUsage{}).
		Where("created_at >= ?", start).
		Select("COALESCE(SUM(cost_usd), 0)").
		Scan(&total).Error; err != nil {
		return 0, fmt.Errorf("get monthly usage: %w", err)
	}
	return total, nil
}
