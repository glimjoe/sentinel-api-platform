package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

type fakeCaseStore struct {
	rows map[string]*model.TestCase
}

func newFakeCaseStore() *fakeCaseStore {
	return &fakeCaseStore{rows: map[string]*model.TestCase{}}
}

func (f *fakeCaseStore) Create(_ context.Context, tc *model.TestCase) error {
	f.rows[tc.ID] = tc
	return nil
}

func (f *fakeCaseStore) FindByID(_ context.Context, id string) (*model.TestCase, error) {
	tc, ok := f.rows[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return tc, nil
}

func (f *fakeCaseStore) ListByProject(_ context.Context, projectID string) ([]*model.TestCase, error) {
	var list []*model.TestCase
	for _, tc := range f.rows {
		if tc.ProjectID == projectID {
			list = append(list, tc)
		}
	}
	return list, nil
}

func (f *fakeCaseStore) Update(_ context.Context, id string, fields map[string]any) error {
	if _, ok := f.rows[id]; !ok {
		return errs.ErrNotFound
	}
	if v, ok := fields["name"].(string); ok {
		f.rows[id].Name = v
	}
	return nil
}

func (f *fakeCaseStore) Delete(_ context.Context, id string) error {
	if _, ok := f.rows[id]; !ok {
		return errs.ErrNotFound
	}
	delete(f.rows, id)
	return nil
}

func TestTestCaseService_Create(t *testing.T) {
	store := newFakeCaseStore()
	roles := newFakeRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleEngineer
	svc := NewTestCaseService(store, roles)

	tc := &model.TestCase{Name: "test case", Method: "GET", Path: "/api/test"}
	result, err := svc.Create(context.Background(), "user-1", "proj-1", tc)
	require.NoError(t, err)

	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "proj-1", result.ProjectID)
	assert.Equal(t, "test case", result.Name)
}

func TestTestCaseService_Create_Forbidden(t *testing.T) {
	store := newFakeCaseStore()
	roles := newFakeRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleViewer
	svc := NewTestCaseService(store, roles)

	_, err := svc.Create(context.Background(), "user-1", "proj-1", &model.TestCase{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrForbidden))
}

func TestTestCaseService_ListByProject(t *testing.T) {
	store := newFakeCaseStore()
	store.rows["c1"] = &model.TestCase{ID: "c1", ProjectID: "proj-1", Name: "tc1"}
	store.rows["c2"] = &model.TestCase{ID: "c2", ProjectID: "proj-1", Name: "tc2"}
	store.rows["c3"] = &model.TestCase{ID: "c3", ProjectID: "proj-2", Name: "tc3"}
	svc := NewTestCaseService(store, newFakeRoleChecker())

	list, err := svc.ListByProject(context.Background(), "proj-1")
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestTestCaseService_FindByID(t *testing.T) {
	store := newFakeCaseStore()
	store.rows["c1"] = &model.TestCase{ID: "c1", Name: "found"}
	svc := NewTestCaseService(store, newFakeRoleChecker())

	tc, err := svc.FindByID(context.Background(), "c1")
	require.NoError(t, err)
	assert.Equal(t, "found", tc.Name)
}

func TestTestCaseService_FindByID_NotFound(t *testing.T) {
	svc := NewTestCaseService(newFakeCaseStore(), newFakeRoleChecker())
	_, err := svc.FindByID(context.Background(), "missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrNotFound))
}

func TestTestCaseService_Update(t *testing.T) {
	store := newFakeCaseStore()
	store.rows["c1"] = &model.TestCase{ID: "c1", ProjectID: "proj-1", Name: "old"}
	roles := newFakeRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleAdmin
	svc := NewTestCaseService(store, roles)

	result, err := svc.Update(context.Background(), "user-1", "proj-1", "c1", map[string]any{"name": "updated"})
	require.NoError(t, err)
	assert.Equal(t, "updated", result.Name)
}

func TestTestCaseService_Update_Forbidden(t *testing.T) {
	store := newFakeCaseStore()
	store.rows["c1"] = &model.TestCase{ID: "c1", ProjectID: "proj-1", Name: "old"}
	roles := newFakeRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleViewer
	svc := NewTestCaseService(store, roles)

	_, err := svc.Update(context.Background(), "user-1", "proj-1", "c1", map[string]any{"name": "x"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrForbidden))
}

func TestTestCaseService_Delete(t *testing.T) {
	store := newFakeCaseStore()
	store.rows["c1"] = &model.TestCase{ID: "c1", ProjectID: "proj-1"}
	roles := newFakeRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleEngineer
	svc := NewTestCaseService(store, roles)

	err := svc.Delete(context.Background(), "user-1", "proj-1", "c1")
	require.NoError(t, err)
	_, err = store.FindByID(context.Background(), "c1")
	assert.True(t, errors.Is(err, errs.ErrNotFound))
}

func TestTestCaseService_Delete_Forbidden(t *testing.T) {
	store := newFakeCaseStore()
	store.rows["c1"] = &model.TestCase{ID: "c1", ProjectID: "proj-1"}
	roles := newFakeRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleViewer
	svc := NewTestCaseService(store, roles)

	err := svc.Delete(context.Background(), "user-1", "proj-1", "c1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrForbidden))
}
