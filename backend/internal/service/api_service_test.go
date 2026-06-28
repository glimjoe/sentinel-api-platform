package service

import (
	"context"
	"errors"
	"testing"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

// -----------------------------------------------------------------------------
// Fakes
// -----------------------------------------------------------------------------

// fakeAPIStore is a hand-rolled fake for apiStore. Tracks (project_id,
// method, path) tuples so the duplicate-key check can be exercised without
// GORM.
type fakeAPIStore struct {
	createErr error // injected for Create error tests
	rows      map[string]*model.API
	// composite uniqueness: projectID + ":" + method + ":" + path → api id
	keys map[string]string
}

func newFakeAPIStore() *fakeAPIStore {
	return &fakeAPIStore{
		rows: map[string]*model.API{},
		keys: map[string]string{},
	}
}

func (f *fakeAPIStore) key(projectID, method, path string) string {
	return projectID + ":" + method + ":" + path
}

func (f *fakeAPIStore) Create(_ context.Context, a *model.API) error {
	if f.createErr != nil {
		return f.createErr
	}
	k := f.key(a.ProjectID, a.Method, a.Path)
	if _, exists := f.keys[k]; exists {
		return errs.ErrConflict
	}
	f.rows[a.ID] = a
	f.keys[k] = a.ID
	return nil
}

func (f *fakeAPIStore) FindByID(_ context.Context, id string) (*model.API, error) {
	a, ok := f.rows[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return a, nil
}

func (f *fakeAPIStore) ListByProject(_ context.Context, projectID string) ([]*model.API, error) {
	var list []*model.API
	for _, a := range f.rows {
		if a.ProjectID == projectID {
			list = append(list, a)
		}
	}
	return list, nil
}

func (f *fakeAPIStore) Delete(_ context.Context, id string) error {
	a, ok := f.rows[id]
	if !ok {
		return errs.ErrNotFound
	}
	delete(f.rows, id)
	delete(f.keys, f.key(a.ProjectID, a.Method, a.Path))
	return nil
}

func (f *fakeAPIStore) Update(_ context.Context, id string, fields map[string]any) error {
	a, ok := f.rows[id]
	if !ok {
		return errs.ErrNotFound
	}
	if name, ok := fields["name"].(string); ok {
		a.Name = name
	}
	if method, ok := fields["method"].(string); ok {
		oldKey := f.key(a.ProjectID, a.Method, a.Path)
		delete(f.keys, oldKey)
		a.Method = method
		f.keys[f.key(a.ProjectID, a.Method, a.Path)] = a.ID
	}
	if path, ok := fields["path"].(string); ok {
		oldKey := f.key(a.ProjectID, a.Method, a.Path)
		delete(f.keys, oldKey)
		a.Path = path
		f.keys[f.key(a.ProjectID, a.Method, a.Path)] = a.ID
	}
	if op, ok := fields["operation_id"].(string); ok {
		a.OperationID = op
	}
	if d, ok := fields["deprecated"].(bool); ok {
		a.Deprecated = d
	}
	if t, ok := fields["tags_json"].([]byte); ok {
		a.TagsJSON = t
	}
	return nil
}

// fakeRoleChecker is the stand-in for ProjectService.RoleFor used in tests.
// Lets each test set up exactly the roles it needs.
type fakeRoleChecker struct {
	// roleByUser maps projectID+":"+userID → role.
	roleByUser map[string]string
	err        error
}

func newFakeRoleChecker() *fakeRoleChecker {
	return &fakeRoleChecker{roleByUser: map[string]string{}}
}

func (f *fakeRoleChecker) RoleFor(_ context.Context, projectID, userID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.roleByUser[projectID+":"+userID], nil
}

// -----------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------

// TestAPIService_Create_HappyPath — admin creates an API row.
func TestAPIService_Create_HappyPath(t *testing.T) {
	apis, roles := newFakeAPIStore(), newFakeRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleAdmin
	svc := NewAPIService(apis, roles)

	a, err := svc.Create(context.Background(), "user-1", "proj-1", "FindPets", "GET", "/pet/findByStatus", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if a.Name != "FindPets" || a.Method != "GET" || a.Path != "/pet/findByStatus" {
		t.Errorf("api = %+v", a)
	}
	if a.ID == "" {
		t.Error("api.ID should be assigned (ULID)")
	}
}

// TestAPIService_Create_DuplicateMethodPath — second Create with same
// (project, method, path) → ErrConflict (so handler returns 409).
func TestAPIService_Create_DuplicateMethodPath(t *testing.T) {
	apis, roles := newFakeAPIStore(), newFakeRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleAdmin
	svc := NewAPIService(apis, roles)

	if _, err := svc.Create(context.Background(), "user-1", "proj-1", "A", "GET", "/x", ""); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err := svc.Create(context.Background(), "user-1", "proj-1", "B", "GET", "/x", "")
	if !errors.Is(err, errs.ErrConflict) {
		t.Errorf("err = %v, want ErrConflict", err)
	}
}

// TestAPIService_Create_ViewerForbidden — viewer trying to add an API → 403.
func TestAPIService_Create_ViewerForbidden(t *testing.T) {
	apis, roles := newFakeAPIStore(), newFakeRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleViewer
	svc := NewAPIService(apis, roles)

	_, err := svc.Create(context.Background(), "user-1", "proj-1", "A", "GET", "/x", "")
	if !errors.Is(err, errs.ErrForbidden) {
		t.Errorf("err = %v, want ErrForbidden", err)
	}
}

// TestAPIService_Create_NonMemberForbidden — non-member → 403 (not 404,
// because the project is known to exist; the role lookup just returns "").
func TestAPIService_Create_NonMemberForbidden(t *testing.T) {
	apis, roles := newFakeAPIStore(), newFakeRoleChecker()
	// user-1 has no role on proj-1
	svc := NewAPIService(apis, roles)

	_, err := svc.Create(context.Background(), "user-1", "proj-1", "A", "GET", "/x", "")
	if !errors.Is(err, errs.ErrForbidden) {
		t.Errorf("err = %v, want ErrForbidden", err)
	}
}

// TestAPIService_Create_InvalidMethod — method not in HTTP enum → 400.
func TestAPIService_Create_InvalidMethod(t *testing.T) {
	apis, roles := newFakeAPIStore(), newFakeRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleAdmin
	svc := NewAPIService(apis, roles)

	_, err := svc.Create(context.Background(), "user-1", "proj-1", "A", "FOOBAR", "/x", "")
	if !errors.Is(err, errs.ErrBadRequest) {
		t.Errorf("err = %v, want ErrBadRequest", err)
	}
}

// TestAPIService_FindByID_NotFound — unknown id → ErrNotFound.
func TestAPIService_FindByID_NotFound(t *testing.T) {
	svc := NewAPIService(newFakeAPIStore(), newFakeRoleChecker())
	_, err := svc.FindByID(context.Background(), "missing")
	if !errors.Is(err, errs.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

// TestAPIService_ListByProject_EmptyReturnsEmpty — empty slice (not nil)
// so handlers can iterate without a nil check.
func TestAPIService_ListByProject_EmptyReturnsEmpty(t *testing.T) {
	svc := NewAPIService(newFakeAPIStore(), newFakeRoleChecker())
	list, err := svc.ListByProject(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if list == nil {
		t.Error("ListByProject should return empty slice, not nil")
	}
	if len(list) != 0 {
		t.Errorf("len(list) = %d, want 0", len(list))
	}
}

// TestAPIService_Delete_HappyPath — admin can delete; row gone afterwards.
func TestAPIService_Delete_HappyPath(t *testing.T) {
	apis, roles := newFakeAPIStore(), newFakeRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleAdmin
	svc := NewAPIService(apis, roles)

	a, _ := svc.Create(context.Background(), "user-1", "proj-1", "A", "GET", "/x", "")
	if err := svc.Delete(context.Background(), "user-1", a.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.FindByID(context.Background(), a.ID); !errors.Is(err, errs.ErrNotFound) {
		t.Errorf("api should be gone, got FindByID err = %v", err)
	}
}

// TestAPIService_Delete_ViewerForbidden — viewer can't delete.
func TestAPIService_Delete_ViewerForbidden(t *testing.T) {
	apis, roles := newFakeAPIStore(), newFakeRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleAdmin
	roles.roleByUser["proj-1:user-2"] = model.ProjectRoleViewer
	svc := NewAPIService(apis, roles)

	a, _ := svc.Create(context.Background(), "user-1", "proj-1", "A", "GET", "/x", "")
	err := svc.Delete(context.Background(), "user-2", a.ID)
	if !errors.Is(err, errs.ErrForbidden) {
		t.Errorf("err = %v, want ErrForbidden", err)
	}
}