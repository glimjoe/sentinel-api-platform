package runner

import (
	"context"
	"fmt"
	"log"
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

// PostExecuteHook is called after each test case result is persisted.
// Use it for side effects like AI attribution, notifications, etc.
type PostExecuteHook func(ctx context.Context, run *model.TestRun, result *model.TestResult)

// Run orchestrates a test run. Publishes SSE events via the optional publisher.
// If hook is non-nil, it is called after each result is persisted.
func Run(ctx context.Context, run *model.TestRun, cases []*model.TestCase, baseURL string, persister ResultPersister, updater RunUpdater, pub EventPublisher, hook PostExecuteHook) error {
	if run == nil {
		return fmt.Errorf("run is nil")
	}
	run.Total = len(cases)
	run.Status = "running"
	now := time.Now()
	run.StartedAt = &now

	// Mark running
	logUpdateErr(updater.Update(ctx, run.ID, map[string]any{
		"status": run.Status, "total": run.Total, "started_at": run.StartedAt,
	}))

	var mu sync.Mutex
	// aggregate increments counters under lock and returns a snapshot
	// so callers can read without racing on run.Passed/Failed/Errored.
	aggregate := func(status string) (int, int, int, int) {
		mu.Lock()
		defer mu.Unlock()
		switch status {
		case "pass":
			run.Passed++
		case "fail":
			run.Failed++
		case "error":
			run.Errored++
		case "skip":
			run.Skipped++
		}
		return run.Passed, run.Failed, run.Errored, run.Skipped
	}

	if run.Mode == "parallel" && run.Concurrency > 1 {
		runParallel(ctx, run, cases, baseURL, persister, pub, updater, aggregate, hook)
	} else {
		runSequential(ctx, run, cases, baseURL, persister, pub, updater, aggregate, hook)
	}

	// Guard: don't overwrite a cancelled run's status.
	if run.Status == "cancelled" {
		finished := time.Now()
		run.FinishedAt = &finished
		logUpdateErr(updater.Update(ctx, run.ID, map[string]any{"finished_at": run.FinishedAt}))
		return nil
	}

	finished := time.Now()
	run.FinishedAt = &finished
	if run.Failed > 0 || run.Errored > 0 {
		run.Status = "failed"
	} else {
		run.Status = "success"
	}

	if pub != nil {
		logPubErr(pub.Publish(ctx, &RunEvent{
			Type: "complete", RunID: run.ID,
			Total: run.Total, Passed: run.Passed, Failed: run.Failed,
			Errored: run.Errored, Skipped: run.Skipped,
			Status: run.Status, Timestamp: time.Now().Unix(),
		}))
	}

	logUpdateErr(updater.Update(ctx, run.ID, map[string]any{
		"status":      run.Status,
		"passed":      run.Passed,
		"failed":      run.Failed,
		"errored":     run.Errored,
		"finished_at": run.FinishedAt,
	}))

	return nil
}

func runSequential(ctx context.Context, run *model.TestRun, cases []*model.TestCase, baseURL string, persister ResultPersister, pub EventPublisher, updater RunUpdater, aggregate func(string) (int, int, int, int), hook PostExecuteHook) {
	for _, tc := range cases {
		select {
		case <-ctx.Done():
			run.Status = "cancelled"
			logUpdateErr(updater.Update(ctx, run.ID, map[string]any{"status": "cancelled", "finished_at": time.Now()}))
			if pub != nil {
				logPubErr(pub.Publish(ctx, &RunEvent{
					Type: "complete", RunID: run.ID, Status: "cancelled",
					Timestamp: time.Now().Unix(),
				}))
			}
			return
		default:
		}
		execOne(ctx, run, tc, baseURL, persister, pub, aggregate, hook)
	}
}

func runParallel(ctx context.Context, run *model.TestRun, cases []*model.TestCase, baseURL string, persister ResultPersister, pub EventPublisher, updater RunUpdater, aggregate func(string) (int, int, int, int), hook PostExecuteHook) {
	sem := make(chan struct{}, run.Concurrency)
	var wg sync.WaitGroup

	for _, tc := range cases {
		select {
		case <-ctx.Done():
			run.Status = "cancelled"
			logUpdateErr(updater.Update(ctx, run.ID, map[string]any{"status": "cancelled", "finished_at": time.Now()}))
			if pub != nil {
				logPubErr(pub.Publish(ctx, &RunEvent{Type: "complete", RunID: run.ID, Status: "cancelled", Timestamp: time.Now().Unix()}))
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
			execOne(ctx, run, tc, baseURL, persister, pub, aggregate, hook)
		}(tc)
	}
	wg.Wait()
}

func logUpdateErr(err error) {
	if err != nil {
		log.Printf("runner: updater.Update: %v", err)
	}
}

func logPubErr(err error) {
	if err != nil {
		log.Printf("runner: pub.Publish: %v", err)
	}
}

func execOne(ctx context.Context, run *model.TestRun, tc *model.TestCase, baseURL string, persister ResultPersister, pub EventPublisher, aggregate func(string) (int, int, int, int), hook PostExecuteHook) {
	result := Execute(ctx, tc, baseURL)
	result.RunID = run.ID
	result.ID = id.New()

	// Hook runs before persistence so it can enrich the result
	// (e.g., AI attribution stored in result.AIAttributionJSON).
	if hook != nil {
		hook(ctx, run, result)
	}

	if err := persister.Create(ctx, result); err != nil {
		result.Status = "error"
		result.ErrorMsg = fmt.Sprintf("persist: %v", err)
	}
	p, f, e, s := aggregate(result.Status)

	if pub != nil {
		logPubErr(pub.Publish(ctx, &RunEvent{
			Type: "progress", RunID: run.ID,
			Total: run.Total, Passed: p, Failed: f,
			Errored: e, Skipped: s,
			Status: "running", Timestamp: time.Now().Unix(),
		}))
	}
}
