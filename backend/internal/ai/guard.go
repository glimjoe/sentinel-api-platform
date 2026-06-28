package ai

import (
	"context"
	"fmt"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

// UsageStore is the persistence contract the Guard needs.
type UsageStore interface {
	RecordUsage(ctx context.Context, m *model.AiUsage) error
	GetDailyUsage(ctx context.Context) (float64, error)
	GetMonthlyUsage(ctx context.Context) (float64, error)
}

// Guard enforces AI budget limits.
type Guard struct {
	store        UsageStore
	dailyLimit   float64
	monthlyLimit float64
}

// NewGuard creates a Guard with the given limits.
func NewGuard(store UsageStore, dailyLimit, monthlyLimit float64) *Guard {
	return &Guard{store: store, dailyLimit: dailyLimit, monthlyLimit: monthlyLimit}
}

// Allow checks whether the current daily and monthly usage are within limits.
func (g *Guard) Allow(ctx context.Context) error {
	daily, err := g.store.GetDailyUsage(ctx)
	if err != nil {
		return fmt.Errorf("guard: check daily usage: %w", err)
	}
	if daily >= g.dailyLimit {
		return ErrDailyBudgetExceeded
	}
	monthly, err := g.store.GetMonthlyUsage(ctx)
	if err != nil {
		return fmt.Errorf("guard: check monthly usage: %w", err)
	}
	if monthly >= g.monthlyLimit {
		return ErrDailyBudgetExceeded
	}
	return nil
}

// Record persists the usage entry after a successful LLM call.
func (g *Guard) Record(ctx context.Context, m *model.AiUsage) error {
	return g.store.RecordUsage(ctx, m)
}
