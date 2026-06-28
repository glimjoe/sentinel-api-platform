package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
	"github.com/glimjoe/sentinel-api-platform/internal/runner"
)

type fakeRunStore struct {
	rows map[string]*model.TestRun
}

func newFakeRunStore() *fakeRunStore {
	return &fakeRunStore{rows: map[string]*model.TestRun{}}
}

func (f *fakeRunStore) Create(_ context.Context, run *model.TestRun) error {
	f.rows[run.ID] = run
	return nil
}

func (f *fakeRunStore) FindByID(_ context.Context, id string) (*model.TestRun, error) {
	r, ok := f.rows[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return r, nil
}

func (f *fakeRunStore) ListByProject(_ context.Context, projectID string) ([]*model.TestRun, error) {
	var list []*model.TestRun
	for _, r := range f.rows {
		if r.ProjectID == projectID {
			list = append(list, r)
		}
	}
	return list, nil
}

func (f *fakeRunStore) Update(_ context.Context, id string, fields map[string]any) error {
	if r, ok := f.rows[id]; ok {
		if v, ok := fields["status"].(string); ok {
			r.Status = v
		}
		return nil
	}
	return errs.ErrNotFound
}

type fakeResultStore struct {
	results []*model.TestResult
}

func (f *fakeResultStore) Create(_ context.Context, tr *model.TestResult) error {
	f.results = append(f.results, tr)
	return nil
}

func (f *fakeResultStore) ListByRun(_ context.Context, runID string) ([]*model.TestResult, error) {
	var list []*model.TestResult
	for _, r := range f.results {
		if r.RunID == runID {
			list = append(list, r)
		}
	}
	return list, nil
}

type fakeEventPublisher struct{}

func (f *fakeEventPublisher) Publish(_ context.Context, _ *runner.RunEvent) error {
	return nil
}

func TestTestRunService_Create(t *testing.T) {
	store := newFakeRunStore()
	roles := newFakeRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleEngineer
	svc := NewTestRunService(store, newFakeCaseStore(), &fakeResultStore{}, roles, &fakeEventPublisher{})

	run, err := svc.Create(context.Background(), "user-1", "proj-1", "smoke test", "http://example.com", "sequential")
	require.NoError(t, err)

	assert.NotEmpty(t, run.ID)
	assert.Equal(t, "proj-1", run.ProjectID)
	assert.Equal(t, "smoke test", run.Name)
	assert.Equal(t, "queued", run.Status)
	assert.Equal(t, "manual", run.TriggerType)
	assert.Equal(t, "user-1", *run.TriggeredBy)
}

func TestTestRunService_Create_Forbidden(t *testing.T) {
	roles := newFakeRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleViewer
	svc := NewTestRunService(newFakeRunStore(), newFakeCaseStore(), &fakeResultStore{}, roles, &fakeEventPublisher{})

	_, err := svc.Create(context.Background(), "user-1", "proj-1", "test", "http://example.com", "sequential")
	require.Error(t, err)
	assert.ErrorIs(t, err, errs.ErrForbidden)
}

func TestTestRunService_Cancel(t *testing.T) {
	store := newFakeRunStore()
	store.rows["run-1"] = &model.TestRun{ID: "run-1", ProjectID: "proj-1", Status: "running"}
	svc := NewTestRunService(store, newFakeCaseStore(), &fakeResultStore{}, newFakeRoleChecker(), &fakeEventPublisher{})

	err := svc.Cancel(context.Background(), "user-1", "run-1")
	require.NoError(t, err)

	run, _ := store.FindByID(context.Background(), "run-1")
	assert.Equal(t, "cancelled", run.Status)
}

func TestTestRunService_Cancel_WrongStatus(t *testing.T) {
	store := newFakeRunStore()
	store.rows["run-1"] = &model.TestRun{ID: "run-1", ProjectID: "proj-1", Status: "success"}
	svc := NewTestRunService(store, newFakeCaseStore(), &fakeResultStore{}, newFakeRoleChecker(), &fakeEventPublisher{})

	err := svc.Cancel(context.Background(), "user-1", "run-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot cancel")
}

func TestTestRunService_FindByID(t *testing.T) {
	store := newFakeRunStore()
	store.rows["run-1"] = &model.TestRun{ID: "run-1", Name: "smoke"}
	svc := NewTestRunService(store, newFakeCaseStore(), &fakeResultStore{}, newFakeRoleChecker(), &fakeEventPublisher{})

	run, err := svc.FindByID(context.Background(), "run-1")
	require.NoError(t, err)
	assert.Equal(t, "smoke", run.Name)
}

func TestTestRunService_ListByProject(t *testing.T) {
	store := newFakeRunStore()
	store.rows["r1"] = &model.TestRun{ID: "r1", ProjectID: "proj-1"}
	store.rows["r2"] = &model.TestRun{ID: "r2", ProjectID: "proj-1"}
	store.rows["r3"] = &model.TestRun{ID: "r3", ProjectID: "proj-2"}
	svc := NewTestRunService(store, newFakeCaseStore(), &fakeResultStore{}, newFakeRoleChecker(), &fakeEventPublisher{})

	list, err := svc.ListByProject(context.Background(), "proj-1")
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestTestRunService_ExportResults(t *testing.T) {
	resultStore := &fakeResultStore{}
	resultStore.results = []*model.TestResult{
		{ID: "res1", RunID: "run-1", Status: "pass"},
		{ID: "res2", RunID: "run-1", Status: "fail"},
		{ID: "res3", RunID: "run-2", Status: "pass"},
	}
	svc := NewTestRunService(newFakeRunStore(), newFakeCaseStore(), resultStore, newFakeRoleChecker(), &fakeEventPublisher{})

	results, err := svc.ExportResults(context.Background(), "run-1")
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestTestRunService_SetPostExecuteHook(t *testing.T) {
	svc := NewTestRunService(newFakeRunStore(), newFakeCaseStore(), &fakeResultStore{}, newFakeRoleChecker(), &fakeEventPublisher{})

	called := false
	hook := func(ctx context.Context, run *model.TestRun, result *model.TestResult) {
		called = true
	}
	svc.SetPostExecuteHook(hook)
	assert.NotNil(t, svc.hook)

	svc.hook(context.Background(), &model.TestRun{}, &model.TestResult{})
	assert.True(t, called)
}

func TestTestRunService_Start_NotFound(t *testing.T) {
	svc := NewTestRunService(newFakeRunStore(), newFakeCaseStore(), &fakeResultStore{}, newFakeRoleChecker(), &fakeEventPublisher{})

	_, err := svc.Start(context.Background(), "user-1", "missing-run")
	require.Error(t, err)
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

func TestTestRunService_Start_AlreadyStarted(t *testing.T) {
	store := newFakeRunStore()
	store.rows["run-1"] = &model.TestRun{ID: "run-1", ProjectID: "proj-1", Status: "running"}
	svc := NewTestRunService(store, newFakeCaseStore(), &fakeResultStore{}, newFakeRoleChecker(), &fakeEventPublisher{})

	_, err := svc.Start(context.Background(), "user-1", "run-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

func TestTestRunService_Start_NoCases(t *testing.T) {
	store := newFakeRunStore()
	store.rows["run-1"] = &model.TestRun{ID: "run-1", ProjectID: "proj-1", Status: "queued"}
	// caseStore is empty — ListByProject returns empty slice
	svc := NewTestRunService(store, newFakeCaseStore(), &fakeResultStore{}, newFakeRoleChecker(), &fakeEventPublisher{})

	_, err := svc.Start(context.Background(), "user-1", "run-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, errs.ErrBadRequest)
}

func TestTestRunService_Create_RoleForError(t *testing.T) {
	roles := newFakeRoleChecker()
	roles.err = errs.ErrNotFound
	svc := NewTestRunService(newFakeRunStore(), newFakeCaseStore(), &fakeResultStore{}, roles, &fakeEventPublisher{})

	_, err := svc.Create(context.Background(), "user-1", "proj-1", "test", "http://example.com", "sequential")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, errs.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
