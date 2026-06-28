package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

// fakePersister implements ResultPersister for tests.
type fakePersister struct {
	results []*model.TestResult
}

func (f *fakePersister) Create(ctx context.Context, tr *model.TestResult) error {
	f.results = append(f.results, tr)
	return nil
}

func (f *fakePersister) ListByRun(ctx context.Context, runID string) ([]*model.TestResult, error) {
	return f.results, nil
}

// fakeUpdater implements RunUpdater for tests.
type fakeUpdater struct {
	updates []map[string]any
}

func (f *fakeUpdater) Update(ctx context.Context, id string, fields map[string]any) error {
	f.updates = append(f.updates, fields)
	return nil
}

// fakePublisher implements EventPublisher for tests.
type fakePublisher struct {
	events []*RunEvent
}

func (f *fakePublisher) Publish(ctx context.Context, event *RunEvent) error {
	f.events = append(f.events, event)
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
