package service

import (
	"context"
	"time"
)

// DashboardStats is the aggregated payload for the ECharts dashboard.
type DashboardStats struct {
	Projects        int              `json:"projects"`
	APIs            int              `json:"apis"`
	MockRules       int              `json:"mock_rules"`
	TestCases       int              `json:"test_cases"`
	RecentRuns      []RunSummary     `json:"recent_runs"`
	StatusBreakdown map[string]int   `json:"status_breakdown"`
}

// RunSummary is a lightweight run row for the recent-runs chart.
type RunSummary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Passed    int       `json:"passed"`
	Failed    int       `json:"failed"`
	Errored   int       `json:"errored"`
	Skipped   int       `json:"skipped"`
	Total     int       `json:"total"`
	CreatedAt time.Time `json:"created_at"`
}

type DashboardStore interface {
	CountProjects(ctx context.Context) (int, error)
	CountAPIs(ctx context.Context) (int, error)
	CountMockRules(ctx context.Context) (int, error)
	CountTestCases(ctx context.Context) (int, error)
	ListRecentRuns(ctx context.Context, limit int) ([]RunSummary, error)
	CountResultsByStatus(ctx context.Context) (map[string]int, error)
}

type DashboardService struct {
	store DashboardStore
}

func NewDashboardService(store DashboardStore) *DashboardService {
	return &DashboardService{store: store}
}

func (s *DashboardService) GetStats(ctx context.Context) (*DashboardStats, error) {
	projects, _ := s.store.CountProjects(ctx)
	apis, _ := s.store.CountAPIs(ctx)
	rules, _ := s.store.CountMockRules(ctx)
	cases, _ := s.store.CountTestCases(ctx)

	runs, _ := s.store.ListRecentRuns(ctx, 10)
	breakdown, _ := s.store.CountResultsByStatus(ctx)

	return &DashboardStats{
		Projects:        projects,
		APIs:            apis,
		MockRules:       rules,
		TestCases:       cases,
		RecentRuns:      runs,
		StatusBreakdown: breakdown,
	}, nil
}
