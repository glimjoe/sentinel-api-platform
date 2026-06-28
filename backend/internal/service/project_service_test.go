package service

import (
	"context"
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
	p, ok := f.projects[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return p, nil
}

func (f *fakeProjectStore) FindBySlug(_ context.Context, slug string) (*model.Project, error) {
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
	members map[string]*model.ProjectMember
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