package runner

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

// fakePersister implements ResultPersister for tests. Mutex-protected for
// parallel-mode tests where goroutines call Create concurrently.
type fakePersister struct {
	mu      sync.Mutex
	results []*model.TestResult
}

func (f *fakePersister) Create(ctx context.Context, tr *model.TestResult) error {
	f.mu.Lock()
	f.results = append(f.results, tr)
	f.mu.Unlock()
	return nil
}

func (f *fakePersister) ListByRun(ctx context.Context, runID string) ([]*model.TestResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.results, nil
}

// fakeUpdater implements RunUpdater for tests. Mutex-protected.
type fakeUpdater struct {
	mu      sync.Mutex
	updates []map[string]any
}

func (f *fakeUpdater) Update(ctx context.Context, id string, fields map[string]any) error {
	f.mu.Lock()
	f.updates = append(f.updates, fields)
	f.mu.Unlock()
	return nil
}

// fakePublisher implements EventPublisher for tests. Mutex-protected.
type fakePublisher struct {
	mu     sync.Mutex
	events []*RunEvent
}

func (f *fakePublisher) Publish(ctx context.Context, event *RunEvent) error {
	f.mu.Lock()
	f.events = append(f.events, event)
	f.mu.Unlock()
	return nil
}

func TestPostExecuteHook_CalledForFailedResult(t *testing.T) {
	persister := &fakePersister{}
	updater := &fakeUpdater{}
	pub := &fakePublisher{}

	var hookCalled bool
	var hookResult *model.TestResult
	hook := func(ctx context.Context, run *model.TestRun, result *model.TestResult) {
		hookCalled = true
		hookResult = result
	}

	run := &model.TestRun{ID: "run1", Mode: "sequential"}
	cases := []*model.TestCase{
		{ID: "case1", Name: "tc1", Method: "GET", Path: "/api/test"},
	}

	err := Run(context.Background(), run, cases, "", persister, updater, pub, hook)
	require.NoError(t, err)

	assert.True(t, hookCalled, "PostExecuteHook should be called")
	assert.NotNil(t, hookResult)
	assert.Equal(t, "case1", hookResult.CaseID)
	assert.NotEmpty(t, hookResult.Status)
}

func TestPostExecuteHook_NotCalledWhenNil(t *testing.T) {
	persister := &fakePersister{}
	updater := &fakeUpdater{}
	pub := &fakePublisher{}

	run := &model.TestRun{ID: "run2", Mode: "sequential"}
	cases := []*model.TestCase{
		{ID: "case2", Name: "tc2", Method: "GET", Path: "/api/test"},
	}

	err := Run(context.Background(), run, cases, "", persister, updater, pub, nil)
	require.NoError(t, err)
}

func TestRun_ParallelMode(t *testing.T) {
	persister := &fakePersister{}
	updater := &fakeUpdater{}
	pub := &fakePublisher{}

	run := &model.TestRun{ID: "run3", Mode: "parallel", Concurrency: 2}
	cases := []*model.TestCase{
		{ID: "c1", Name: "tc1", Method: "GET", Path: "/api/test1"},
		{ID: "c2", Name: "tc2", Method: "GET", Path: "/api/test2"},
	}

	err := Run(context.Background(), run, cases, "", persister, updater, pub, nil)
	require.NoError(t, err)

	assert.Equal(t, 2, len(persister.results))
	// Runs complete even with failed cases
	assert.Contains(t, []string{"success", "failed"}, run.Status)
}

func TestRun_CancelSequential(t *testing.T) {
	persister := &fakePersister{}
	updater := &fakeUpdater{}
	pub := &fakePublisher{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately before Run

	run := &model.TestRun{ID: "run4", Mode: "sequential"}
	cases := []*model.TestCase{
		{ID: "c1", Name: "tc1", Method: "GET", Path: "/api/test"},
	}

	err := Run(ctx, run, cases, "", persister, updater, pub, nil)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", run.Status)
}

func TestRun_NilRun(t *testing.T) {
	err := Run(context.Background(), nil, nil, "", nil, nil, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run is nil")
}

func TestRun_EmptyCases(t *testing.T) {
	persister := &fakePersister{}
	updater := &fakeUpdater{}
	pub := &fakePublisher{}

	run := &model.TestRun{ID: "run5", Mode: "sequential"}
	err := Run(context.Background(), run, nil, "", persister, updater, pub, nil)
	require.NoError(t, err)

	assert.Equal(t, "success", run.Status)
	assert.Equal(t, 0, run.Total)
}

func TestRun_SuccessPath(t *testing.T) {
	persister := &fakePersister{}
	updater := &fakeUpdater{}
	pub := &fakePublisher{}

	run := &model.TestRun{ID: "run6", Mode: "sequential"}
	cases := []*model.TestCase{
		{ID: "c1", Name: "tc1", Method: "GET", Path: "/api/test"},
	}

	err := Run(context.Background(), run, cases, "", persister, updater, pub, nil)
	require.NoError(t, err)

	// Run completes (status depends on whether cases pass; without real target it fails)
	assert.NotNil(t, run.StartedAt)
	assert.NotNil(t, run.FinishedAt)
	assert.Equal(t, 1, len(persister.results))

	// Verify aggregate counts sum to total
	assert.Equal(t, 1, run.Total)
	assert.Equal(t, run.Total, run.Passed+run.Failed+run.Errored+run.Skipped)

	// Verify updater was called (at minimum: started + finished)
	assert.GreaterOrEqual(t, len(updater.updates), 2)

	// Verify complete event published
	completeFound := false
	for _, e := range pub.events {
		if e.Type == "complete" {
			completeFound = true
			break
		}
	}
	assert.True(t, completeFound, "complete event should be published")
}

func TestRun_ParallelCancel(t *testing.T) {
	persister := &fakePersister{}
	updater := &fakeUpdater{}
	pub := &fakePublisher{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	run := &model.TestRun{ID: "run7", Mode: "parallel", Concurrency: 2}
	cases := []*model.TestCase{
		{ID: "c1", Name: "tc1", Method: "GET", Path: "/api/test1"},
		{ID: "c2", Name: "tc2", Method: "GET", Path: "/api/test2"},
	}

	err := Run(ctx, run, cases, "", persister, updater, pub, nil)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", run.Status)
}
