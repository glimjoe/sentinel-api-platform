// Package repository — MockHitRepo (Phase 2 M1, stub).
//
// MockHitRepo persists mock_hits rows. The full implementation lands in M2
// alongside the async batch recorder (see plan §6.6 `recorder.go`). For M1
// the struct exists so it can be wired into cmd/server/main.go; no method
// body is required yet because the engine's M1 TODO comment skips the call.
package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

// MockHitRepo persists mock_hits rows. M2 will use this from the recorder.
type MockHitRepo struct {
	db *gorm.DB
}

// NewMockHitRepo constructs a MockHitRepo bound to db.
func NewMockHitRepo(db *gorm.DB) *MockHitRepo {
	return &MockHitRepo{db: db}
}

// Create inserts a single mock_hits row. TODO(Phase 2 M2): move the body to
// the async batch recorder so high-volume mocks don't block the engine on a
// per-request INSERT.
func (r *MockHitRepo) Create(ctx context.Context, hit *model.MockHit) error {
	if err := r.db.WithContext(ctx).Create(hit).Error; err != nil {
		return fmt.Errorf("create mock_hit: %w", err)
	}
	return nil
}
