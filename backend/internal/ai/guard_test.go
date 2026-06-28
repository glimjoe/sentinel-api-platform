package ai

import (
	"context"
	"testing"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

type fakeUsageStore struct {
	daily   float64
	monthly float64
}

func (f *fakeUsageStore) RecordUsage(_ context.Context, _ *model.AiUsage) error { return nil }
func (f *fakeUsageStore) GetDailyUsage(_ context.Context) (float64, error)      { return f.daily, nil }
func (f *fakeUsageStore) GetMonthlyUsage(_ context.Context) (float64, error)    { return f.monthly, nil }

func TestGuard_Allow_UnderBudget(t *testing.T) {
	g := &Guard{store: &fakeUsageStore{daily: 0.5, monthly: 5.0}, dailyLimit: 1.0, monthlyLimit: 20.0}
	if err := g.Allow(context.Background()); err != nil {
		t.Errorf("expected Allow() under budget, got %v", err)
	}
}

func TestGuard_Allow_DailyExceeded(t *testing.T) {
	g := &Guard{store: &fakeUsageStore{daily: 2.0, monthly: 5.0}, dailyLimit: 1.0, monthlyLimit: 20.0}
	if err := g.Allow(context.Background()); err != ErrDailyBudgetExceeded {
		t.Errorf("expected ErrDailyBudgetExceeded, got %v", err)
	}
}

func TestGuard_Allow_MonthlyExceeded(t *testing.T) {
	g := &Guard{store: &fakeUsageStore{daily: 0.1, monthly: 30.0}, dailyLimit: 1.0, monthlyLimit: 20.0}
	if err := g.Allow(context.Background()); err != ErrMonthlyBudgetExceeded {
		t.Errorf("expected ErrMonthlyBudgetExceeded, got %v", err)
	}
}

func TestGuard_Record(t *testing.T) {
	g := &Guard{store: &fakeUsageStore{}}
	if err := g.Record(context.Background(), &model.AiUsage{Model: "mock", Function: "attribution", CostUSD: 0.001}); err != nil {
		t.Errorf("Record: %v", err)
	}
}
