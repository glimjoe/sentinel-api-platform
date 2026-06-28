package runner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/id"
)

// ResultPersister saves test results during a run.
type ResultPersister interface {
	Create(ctx context.Context, tr *model.TestResult) error
}

// RunUpdater updates test run aggregate counts after each case completes.
type RunUpdater interface {
	Update(ctx context.Context, id string, fields map[string]any) error
}

// Run orchestrates a test run. It loads the given cases, executes each one,
// persists results, and updates the run's aggregate counters.
func Run(ctx context.Context, run *model.TestRun, cases []*model.TestCase, baseURL string, persister ResultPersister, updater RunUpdater) error {
	if run == nil {
		return fmt.Errorf("run is nil")
	}
	run.Total = len(cases)
	run.Status = "running"
	now := time.Now()
	run.StartedAt = &now

	// Mark running
	updater.Update(ctx, run.ID, map[string]any{
		"status": run.Status, "total": run.Total, "started_at": run.StartedAt,
	})

	var mu sync.Mutex
	aggregate := func(status string) {
		mu.Lock()
		defer mu.Unlock()
		switch status {
		case "pass": run.Passed++
		case "fail": run.Failed++
		case "error": run.Errored++
		case "skip": run.Skipped++
		}
	}

	// Sequential execution (parallel deferred to later Phase 3 iteration)
	for _, tc := range cases {
		result := Execute(ctx, tc, baseURL)
		result.RunID = run.ID
		result.ID = id.New()

		// Persist result
		if err := persister.Create(ctx, result); err != nil {
			result.Status = "error"
			result.ErrorMsg = fmt.Sprintf("persist: %v", err)
		}
		aggregate(result.Status)
	}

	finished := time.Now()
	run.FinishedAt = &finished
	if run.Failed > 0 || run.Errored > 0 {
		run.Status = "failed"
	} else {
		run.Status = "success"
	}

	updater.Update(ctx, run.ID, map[string]any{
		"status":      run.Status,
		"passed":      run.Passed,
		"failed":      run.Failed,
		"errored":     run.Errored,
		"finished_at": run.FinishedAt,
	})

	return nil
}
