package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/id"
)

type AuditLogRepo struct{ db *gorm.DB }

func NewAuditLogRepo(db *gorm.DB) *AuditLogRepo { return &AuditLogRepo{db: db} }

func (r *AuditLogRepo) Insert(ctx context.Context, entry *model.AuditLog) error {
	if entry.ID == "" {
		entry.ID = id.New()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	return r.db.WithContext(ctx).Create(entry).Error
}
