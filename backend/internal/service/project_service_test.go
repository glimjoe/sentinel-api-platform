package service

import (
	"context"
	"fmt"
	"errors"
	"testing"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

// fakeProjectStore is a hand-rolled fake. The package intentionally avoids
// a mocking library so the test file has no extra dependency; this struct
// is small enough to inline.
type fakeProjectStore struct {
	createErr error // injected for Create error tests
	findErr   error // injected for FindByID error tests
	projects  map[string]*model.Project
	slugs     map[string]string // slug → id (uniqueness check)
}

func newFakeProjectStore() *fakeProjectStore {
	return &fakeProjectStore{
		projects: map[string]*model.Project{},
		slugs:    map[string]string{},
	}
}

func (f *fakeProjectStore) Create(_ context.Context, p *model.Project) error {
	if f.createErr != nil {
		return f.createErr
	}
	if _, exists := f.slugs[p.Slug]; exists {
		return errs.ErrConflict
	}
	f.projects[p.ID] = p
	f.slugs[p.Slug] = p.ID
	return nil
}

func (f *fakeProjectStore) FindByID(_ context.Context, id string) (*model.Project, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	p, ok := f.projects[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return p, nil
}

func (f *fakeProjectStore) FindBySlug(_ context.Context, slug string) (*model.Project, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	id, ok := f.slugs[slug]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return f.projects[id], nil
}

func (f *fakeProjectStore) ListByOwner(_ context.Context, ownerID string) ([]*model.Project, error) {
	var list []*model.Project
	for _, p := range f.projects {
		if p.OwnerID == ownerID {
			list = append(list, p)
		}
	}
	return list, nil
}

func (f *fakeProjectStore) Update(_ context.Context, p *model.Project) error {
	f.projects[p.ID] = p
	return nil
}

func (f *fakeProjectStore) Delete(_ context.Context, id string) error {
	delete(f.projects, id)
	return nil
}

// fakeMemberStore mirrors fakeProjectStore for project_members. Key is
// projectID+":"+userID so the composite PK maps cleanly to a map key.
type fakeMemberStore struct {
	members   map[string]*model.ProjectMember
	listErr   error
}

func newFakeMemberStore() *fakeMemberStore {
	return &fakeMemberStore{members: map[string]*model.ProjectMember{}}
}

func (f *fakeMemberStore) key(projectID, userID string) string {
	return projectID + ":" + userID
}

func (f *fakeMemberStore) Add(_ context.Context, m *model.ProjectMember) error {
	f.members[f.key(m.ProjectID, m.UserID)] = m
	return nil
}

func (f *fakeMemberStore) Remove(_ context.Context, projectID, userID string) error {
	delete(f.members, f.key(projectID, userID))
	return nil
}

func (f *fakeMemberStore) ListByProject(_ context.Context, projectID string) ([]*model.ProjectMember, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	var list []*model.ProjectMember
	for _, m := range f.members {
		if m.ProjectID == projectID {
			list = append(list, m)
		}
	}
	return list, nil
}

func (f *fakeMemberStore) Find(_ context.Context, projectID, userID string) (*model.ProjectMember, error) {
	m, ok := f.members[f.key(projectID, userID)]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return m, nil
}

// -----------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------

// TestProjectService_Create_HappyPath — owner becomes admin member on Create.
func TestProjectService_Create_HappyPath(t *testing.T) {
	projects, members := newFakeProjectStore(), newFakeMemberStore()
	svc := NewProjectService(projects, members)

	p, err := svc.Create(context.Background(), "user-1", "Petstore", "petstore", "demo")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.Name != "Petstore" || p.Slug != "petstore" || p.OwnerID != "user-1" {
		t.Errorf("project = %+v", p)
	}
	// Owner should be auto-added as admin member.
	m, err := members.Find(context.Background(), p.ID, "user-1")
	if err != nil {
		t.Fatalf("owner should be admin member: %v", err)
	}
	if m.Role != model.ProjectRoleAdmin {
		t.Errorf("owner member role = %q, want admin", m.Role)
	}
}

// TestProjectService_Create_DuplicateSlug — second Create with same slug
// must return ErrConflict so the handler can map to 409.
func TestProjectService_Create_DuplicateSlug(t *testing.T) {
	projects, members := newFakeProjectStore(), newFakeMemberStore()
	svc := NewProjectService(projects, members)

	if _, err := svc.Create(context.Background(), "user-1", "First", "petstore", ""); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err := svc.Create(context.Background(), "user-2", "Second", "petstore", "")
	if !errors.Is(err, errs.ErrConflict) {
		t.Errorf("err = %v, want ErrConflict", err)
	}
}

// TestProjectService_Create_EmptyName — empty name/slug is ErrBadRequest.
func TestProjectService_Create_EmptyName(t *testing.T) {
	svc := NewProjectService(newFakeProjectStore(), newFakeMemberStore())
	_, err := svc.Create(context.Background(), "user-1", "", "x", "")
	if !errors.Is(err, errs.ErrBadRequest) {
		t.Errorf("err = %v, want ErrBadRequest", err)
	}
}

// TestProjectService_RoleFor_OwnerIsAdmin — even if no project_members row
// exists for the owner, RoleFor must return "admin" (delegation pattern;
// owner is admin by virtue of project.owner_id).
func TestProjectService_RoleFor_OwnerIsAdmin(t *testing.T) {
	projects, members := newFakeProjectStore(), newFakeMemberStore()
	svc := NewProjectService(projects, members)
	p, _ := svc.Create(context.Background(), "user-1", "P", "p", "")

	role, err := svc.RoleFor(context.Background(), p.ID, "user-1")
	if err != nil {
		t.Fatalf("RoleFor: %v", err)
	}
	if role != model.ProjectRoleAdmin {
		t.Errorf("owner role = %q, want admin", role)
	}
}

// TestProjectService_RoleFor_NonMemberReturnsEmpty — non-member gets "" + nil
// (the contract ProjectRoleResolver expects).
func TestProjectService_RoleFor_NonMemberReturnsEmpty(t *testing.T) {
	projects, members := newFakeProjectStore(), newFakeMemberStore()
	svc := NewProjectService(projects, members)
	p, _ := svc.Create(context.Background(), "user-1", "P", "p", "")

	role, err := svc.RoleFor(context.Background(), p.ID, "user-2")
	if err != nil {
		t.Fatalf("RoleFor: %v", err)
	}
	if role != "" {
		t.Errorf("role = %q, want empty", role)
	}
}

// TestProjectService_RoleFor_MemberReturnsRole — added member gets their role.
func TestProjectService_RoleFor_MemberReturnsRole(t *testing.T) {
	projects, members := newFakeProjectStore(), newFakeMemberStore()
	svc := NewProjectService(projects, members)
	p, _ := svc.Create(context.Background(), "user-1", "P", "p", "")
	if err := svc.AddMember(context.Background(), "user-1", p.ID, "user-2", model.ProjectRoleEngineer); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	role, err := svc.RoleFor(context.Background(), p.ID, "user-2")
	if err != nil {
		t.Fatalf("RoleFor: %v", err)
	}
	if role != model.ProjectRoleEngineer {
		t.Errorf("role = %q, want engineer", role)
	}
}

// TestProjectService_Update_NonAdminForbidden — viewer trying to update
// gets ErrForbidden, not the new value silently applied.
func TestProjectService_Update_NonAdminForbidden(t *testing.T) {
	projects, members := newFakeProjectStore(), newFakeMemberStore()
	svc := NewProjectService(projects, members)
	p, _ := svc.Create(context.Background(), "user-1", "P", "p", "")
	_ = svc.AddMember(context.Background(), "user-1", p.ID, "user-2", model.ProjectRoleViewer)

	_, err := svc.Update(context.Background(), "user-2", p.ID, "NewName", "x")
	if !errors.Is(err, errs.ErrForbidden) {
		t.Errorf("err = %v, want ErrForbidden", err)
	}
}

// TestProjectService_Delete_OnlyOwner — even an admin member can't delete
// a project they don't own. Owners are the only delete authority.
func TestProjectService_Delete_OnlyOwner(t *testing.T) {
	projects, members := newFakeProjectStore(), newFakeMemberStore()
	svc := NewProjectService(projects, members)
	p, _ := svc.Create(context.Background(), "user-1", "P", "p", "")
	_ = svc.AddMember(context.Background(), "user-1", p.ID, "user-2", model.ProjectRoleAdmin)

	err := svc.Delete(context.Background(), "user-2", p.ID)
	if !errors.Is(err, errs.ErrForbidden) {
		t.Errorf("err = %v, want ErrForbidden", err)
	}
}


func TestProjectService_FindByID_HappyPath(t *testing.T) {
	projects, members := newFakeProjectStore(), newFakeMemberStore()
	svc := NewProjectService(projects, members)
	p, _ := svc.Create(context.Background(), "user-1", "P", "p", "")

	found, err := svc.FindByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Name != "P" {
		t.Errorf("Name = %q, want P", found.Name)
	}
}

func TestProjectService_FindByID_NotFound(t *testing.T) {
	svc := NewProjectService(newFakeProjectStore(), newFakeMemberStore())
	_, err := svc.FindByID(context.Background(), "missing")
	if !errors.Is(err, errs.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestProjectService_FindBySlug_HappyPath(t *testing.T) {
	projects, members := newFakeProjectStore(), newFakeMemberStore()
	svc := NewProjectService(projects, members)
	svc.Create(context.Background(), "user-1", "P", "my-slug", "")

	found, err := svc.FindBySlug(context.Background(), "my-slug")
	if err != nil {
		t.Fatalf("FindBySlug: %v", err)
	}
	if found.Name != "P" {
		t.Errorf("Name = %q, want P", found.Name)
	}
}

func TestProjectService_FindBySlug_NotFound(t *testing.T) {
	svc := NewProjectService(newFakeProjectStore(), newFakeMemberStore())
	_, err := svc.FindBySlug(context.Background(), "no-such-slug")
	if !errors.Is(err, errs.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestProjectService_List_HappyPath(t *testing.T) {
	projects, members := newFakeProjectStore(), newFakeMemberStore()
	svc := NewProjectService(projects, members)
	svc.Create(context.Background(), "user-1", "A", "a", "")
	svc.Create(context.Background(), "user-1", "B", "b", "")
	svc.Create(context.Background(), "user-2", "C", "c", "")

	list, err := svc.List(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("len = %d, want 2", len(list))
	}
	if list == nil {
		t.Error("List should return non-nil slice")
	}
}

func TestProjectService_List_Empty(t *testing.T) {
	svc := NewProjectService(newFakeProjectStore(), newFakeMemberStore())
	list, err := svc.List(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list == nil {
		t.Error("should return empty slice, not nil")
	}
	if len(list) != 0 {
		t.Errorf("len = %d, want 0", len(list))
	}
}

func TestProjectService_Update_HappyPath(t *testing.T) {
	projects, members := newFakeProjectStore(), newFakeMemberStore()
	svc := NewProjectService(projects, members)
	p, _ := svc.Create(context.Background(), "user-1", "Old", "old", "")

	updated, err := svc.Update(context.Background(), "user-1", p.ID, "New", "")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "New" {
		t.Errorf("Name = %q, want New", updated.Name)
	}
}

func TestProjectService_Delete_HappyPath(t *testing.T) {
	projects, members := newFakeProjectStore(), newFakeMemberStore()
	svc := NewProjectService(projects, members)
	p, _ := svc.Create(context.Background(), "user-1", "P", "p", "")

	if err := svc.Delete(context.Background(), "user-1", p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := svc.FindByID(context.Background(), p.ID)
	if !errors.Is(err, errs.ErrNotFound) {
		t.Errorf("project should be gone, got err = %v", err)
	}
}

func TestProjectService_ListMembers_HappyPath(t *testing.T) {
	projects, members := newFakeProjectStore(), newFakeMemberStore()
	svc := NewProjectService(projects, members)
	p, _ := svc.Create(context.Background(), "user-1", "P", "p", "")
	// Create auto-adds owner as admin, so starting count is 1.
	svc.AddMember(context.Background(), "user-1", p.ID, "user-2", model.ProjectRoleEngineer)
	svc.AddMember(context.Background(), "user-1", p.ID, "user-3", model.ProjectRoleViewer)

	list, err := svc.ListMembers(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	// owner (auto-admin) + user-2 + user-3 = 3
	if len(list) != 3 {
		t.Errorf("len = %d, want 3 (owner auto-admin + 2 added)", len(list))
	}
	if list == nil {
		t.Error("should return non-nil slice")
	}
}

func TestProjectService_ListMembers_ReturnsOwner(t *testing.T) {
	projects, members := newFakeProjectStore(), newFakeMemberStore()
	svc := NewProjectService(projects, members)
	p, _ := svc.Create(context.Background(), "user-1", "P", "p", "")
	// Create auto-adds owner; no additional members added.
	list, err := svc.ListMembers(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if list == nil {
		t.Error("should return non-nil slice")
	}
	if len(list) != 1 {
		t.Errorf("len = %d, want 1 (owner auto-added as admin)", len(list))
	}
}

func TestProjectService_RoleFor_DBError(t *testing.T) {
	projects, members := newFakeProjectStore(), newFakeMemberStore()
	projects.findErr = fmt.Errorf("db connection lost")
	svc := NewProjectService(projects, members)

	_, err := svc.RoleFor(context.Background(), "any-project", "any-user")
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, errs.ErrNotFound) {
		t.Error("should not wrap as ErrNotFound for generic DB error")
	}
}

func TestProjectService_FindByID_DBError(t *testing.T) {
	projects, _ := newFakeProjectStore(), newFakeMemberStore()
	projects.findErr = fmt.Errorf("db connection lost")
	svc := NewProjectService(projects, newFakeMemberStore())

	_, err := svc.FindByID(context.Background(), "any-id")
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, errs.ErrNotFound) {
		t.Error("should not be ErrNotFound for generic DB error")
	}
}

func TestProjectService_ListMembers_DBError(t *testing.T) {
	projects, members := newFakeProjectStore(), newFakeMemberStore()
	members.listErr = fmt.Errorf("db connection lost")
	svc := NewProjectService(projects, members)
	p, _ := svc.Create(context.Background(), "user-1", "P", "p", "")

	_, err := svc.ListMembers(context.Background(), p.ID)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProjectService_Delete_FindError(t *testing.T) {
	projects, members := newFakeProjectStore(), newFakeMemberStore()
	projects.findErr = fmt.Errorf("db down")
	svc := NewProjectService(projects, members)

	err := svc.Delete(context.Background(), "user-1", "any-id")
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, errs.ErrNotFound) {
		t.Error("should not be ErrNotFound for generic DB error")
	}
}
