package runner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/id"
)

// ResultPersister manages test results.
type ResultPersister interface {
	Create(ctx context.Context, tr *model.TestResult) error
	ListByRun(ctx context.Context, runID string) ([]*model.TestResult, error)
}

// RunUpdater updates test run aggregate counts after each case completes.
type RunUpdater interface {
	Update(ctx context.Context, id string, fields map[string]any) error
}

// Run orchestrates a test run. Publishes SSE events via the optional publisher.
func Run(ctx context.Context, run *model.TestRun, cases []*model.TestCase, baseURL string, persister ResultPersister, updater RunUpdater, pub EventPublisher) error {
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

	if run.Mode == "parallel" && run.Concurrency > 1 {
		runParallel(ctx, run, cases, baseURL, persister, pub, aggregate)
	} else {
		runSequential(ctx, run, cases, baseURL, persister, pub, updater, aggregate)
	}

	finished := time.Now()
	run.FinishedAt = &finished
	if run.Failed > 0 || run.Errored > 0 {
		run.Status = "failed"
	} else {
		run.Status = "success"
	}

	if pub != nil {
		pub.Publish(ctx, &RunEvent{
			Type: "complete", RunID: run.ID,
			Total: run.Total, Passed: run.Passed, Failed: run.Failed,
			Errored: run.Errored, Skipped: run.Skipped,
			Status: run.Status, Timestamp: time.Now().Unix(),
		})
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

func runSequential(ctx context.Context, run *model.TestRun, cases []*model.TestCase, baseURL string, persister ResultPersister, pub EventPublisher, updater RunUpdater, aggregate func(string)) {
	for _, tc := range cases {
		select {
		case <-ctx.Done():
			run.Status = "cancelled"
			updater.Update(ctx, run.ID, map[string]any{"status": "cancelled", "finished_at": time.Now()})
			if pub != nil {
				pub.Publish(ctx, &RunEvent{
					Type: "complete", RunID: run.ID, Status: "cancelled",
					Timestamp: time.Now().Unix(),
				})
			}
			return
		default:
		}
		execOne(ctx, run, tc, baseURL, persister, pub, aggregate)
	}
}

func runParallel(ctx context.Context, run *model.TestRun, cases []*model.TestCase, baseURL string, persister ResultPersister, pub EventPublisher, aggregate func(string)) {
	sem := make(chan struct{}, run.Concurrency)
	var wg sync.WaitGroup

	for _, tc := range cases {
		select {
		case <-ctx.Done():
			run.Status = "cancelled"
			if pub != nil {
				pub.Publish(ctx, &RunEvent{Type: "complete", RunID: run.ID, Status: "cancelled", Timestamp: time.Now().Unix()})
			}
			wg.Wait()
			return
		default:
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(tc *model.TestCase) {
			defer wg.Done()
			defer func() { <-sem }()
			execOne(ctx, run, tc, baseURL, persister, pub, aggregate)
		}(tc)
	}
	wg.Wait()
}

func execOne(ctx context.Context, run *model.TestRun, tc *model.TestCase, baseURL string, persister ResultPersister, pub EventPublisher, aggregate func(string)) {
	result := Execute(ctx, tc, baseURL)
	result.RunID = run.ID
	result.ID = id.New()

	if err := persister.Create(ctx, result); err != nil {
		result.Status = "error"
		result.ErrorMsg = fmt.Sprintf("persist: %v", err)
	}
	aggregate(result.Status)

	if pub != nil {
		pub.Publish(ctx, &RunEvent{
			Type: "progress", RunID: run.ID,
			Total: run.Total, Passed: run.Passed, Failed: run.Failed,
			Errored: run.Errored, Skipped: run.Skipped,
			Status: "running", Timestamp: time.Now().Unix(),
		})
	}
}
