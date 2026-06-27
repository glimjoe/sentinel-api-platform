// Package service — ProjectService (M2-E).
//
// ProjectService is the entry point for project CRUD + membership business
// logic. It also implements middleware.ProjectRoleResolver via RoleFor, so
// the RBAC middleware can ask "what role does user X have on project Y?"
// without each importing the repository layer.
//
// Design notes:
//
//   - Persistence contracts (projectStore, projectMemberStore) are declared
//     in this file (not in repository/) so tests can supply fakes without
//     dragging *gorm.DB into the service test. Matches auth_service's
//     userStore pattern (auth_service.go L24-28).
//
//   - Owner authorization is enforced in two places: Update/Delete/AddMember
//     check the caller's role (admin required), while Delete additionally
//     requires owner_id == callerID (only the owner can delete, even an
//     admin member can't). This matches the Phase 2 plan §6.2 intent.
//
//   - The "owner is implicitly admin" rule is implemented in RoleFor rather
//     than stored as a redundant project_members row. This keeps the members
//     table lean and avoids a "which row wins?" bug when an owner is also
//     added with a different role.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/id"
)

// projectStore is the persistence contract ProjectService needs from the
// projects table. Mirrors auth_service's userStore pattern: declared in the
// service file so tests can supply a fake without dragging *gorm.DB in.
type projectStore interface {
	Create(ctx context.Context, p *model.Project) error
	FindByID(ctx context.Context, id string) (*model.Project, error)
	FindBySlug(ctx context.Context, slug string) (*model.Project, error)
	ListByOwner(ctx context.Context, ownerID string) ([]*model.Project, error)
	Update(ctx context.Context, p *model.Project) error
	Delete(ctx context.Context, id string) error
}

// projectMemberStore is the persistence contract for project_members.
type projectMemberStore interface {
	Add(ctx context.Context, m *model.ProjectMember) error
	Remove(ctx context.Context, projectID, userID string) error
	ListByProject(ctx context.Context, projectID string) ([]*model.ProjectMember, error)
	Find(ctx context.Context, projectID, userID string) (*model.ProjectMember, error)
}

// ProjectService wires the project + membership business logic.
type ProjectService struct {
	projects projectStore
	members  projectMemberStore
}

// NewProjectService constructs a ProjectService. Caller owns the lifetime.
func NewProjectService(projects projectStore, members projectMemberStore) *ProjectService {
	return &ProjectService{projects: projects, members: members}
}

// Create creates a new project. Caller becomes owner and is auto-added as
// an admin member (so RoleFor returns "admin" for them even before they
// touch the members endpoint).
//
// Errors:
//   - errs.ErrBadRequest — empty name or slug
//   - errs.ErrConflict   — slug already taken
//   - generic wrap       — DB failure
func (s *ProjectService) Create(ctx context.Context, ownerID, name, slug, description string) (*model.Project, error) {
	name = strings.TrimSpace(name)
	slug = strings.TrimSpace(slug)
	if name == "" || slug == "" {
		return nil, fmt.Errorf("%w: name and slug are required", errs.ErrBadRequest)
	}
	p := &model.Project{
		ID:          id.New(),
		Name:        name,
		Slug:        slug,
		OwnerID:     ownerID,
		Description: strings.TrimSpace(description),
	}
	if err := s.projects.Create(ctx, p); err != nil {
		if errors.Is(err, errs.ErrConflict) {
			return nil, errs.ErrConflict
		}
		return nil, fmt.Errorf("create project: %w", err)
	}
	if err := s.members.Add(ctx, &model.ProjectMember{
		ProjectID: p.ID,
		UserID:    ownerID,
		Role:      model.ProjectRoleAdmin,
	}); err != nil {
		// The project row already exists at this point. Returning the wrap
		// lets the caller retry the Create; on retry, projects.Create will
		// return ErrConflict (slug already taken) so the caller can clean
		// up by attempting Delete. Avoids silent orphans.
		return nil, fmt.Errorf("add owner as admin member: %w", err)
	}
	return p, nil
}

// FindByID returns the project with the given id, or errs.ErrNotFound.
func (s *ProjectService) FindByID(ctx context.Context, pid string) (*model.Project, error) {
	p, err := s.projects.FindByID(ctx, pid)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("find project: %w", err)
	}
	return p, nil
}

// FindBySlug returns the project with the given slug, or errs.ErrNotFound.
// Used by the public /mock/:slug/*path route to resolve a project before
// serving a mock response.
func (s *ProjectService) FindBySlug(ctx context.Context, slug string) (*model.Project, error) {
	p, err := s.projects.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("find project by slug: %w", err)
	}
	return p, nil
}

// List returns all projects the caller owns. Empty slice, not nil, when none.
func (s *ProjectService) List(ctx context.Context, ownerID string) ([]*model.Project, error) {
	list, err := s.projects.ListByOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	if list == nil {
		list = []*model.Project{}
	}
	return list, nil
}

// Update changes mutable fields (name, description). Caller must be admin.
// Empty name/description arguments are treated as "no change" so a partial
// PATCH semantics work; the caller passes only the fields they want changed.
//
// Errors:
//   - errs.ErrForbidden — caller is not admin on this project
//   - errs.ErrNotFound  — project doesn't exist (also surfaced via RoleFor)
func (s *ProjectService) Update(ctx context.Context, callerID, projectID, name, description string) (*model.Project, error) {
	role, err := s.RoleFor(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if role != model.ProjectRoleAdmin {
		return nil, errs.ErrForbidden
	}
	p, err := s.projects.FindByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("find project: %w", err)
	}
	if name = strings.TrimSpace(name); name != "" {
		p.Name = name
	}
	if desc := strings.TrimSpace(description); desc != "" {
		p.Description = desc
	}
	if err := s.projects.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}
	return p, nil
}

// Delete removes the project. Only the owner can delete — admin members
// are explicitly forbidden. This prevents a co-admin from accidentally
// destroying a project the owner is still using.
//
// Errors:
//   - errs.ErrForbidden — caller is not the owner
//   - errs.ErrNotFound  — project doesn't exist
func (s *ProjectService) Delete(ctx context.Context, callerID, projectID string) error {
	p, err := s.projects.FindByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return errs.ErrNotFound
		}
		return fmt.Errorf("find project: %w", err)
	}
	if p.OwnerID != callerID {
		return errs.ErrForbidden
	}
	if err := s.projects.Delete(ctx, projectID); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}

// AddMember grants a user a role on the project. Caller must be admin.
//
// Errors:
//   - errs.ErrForbidden  — caller is not admin
//   - errs.ErrBadRequest — role is not admin/engineer/viewer
func (s *ProjectService) AddMember(ctx context.Context, callerID, projectID, userID, role string) error {
	callerRole, err := s.RoleFor(ctx, projectID, callerID)
	if err != nil {
		return err
	}
	if callerRole != model.ProjectRoleAdmin {
		return errs.ErrForbidden
	}
	if !isValidProjectRole(role) {
		return fmt.Errorf("%w: role must be admin/engineer/viewer", errs.ErrBadRequest)
	}
	return s.members.Add(ctx, &model.ProjectMember{
		ProjectID: projectID,
		UserID:    userID,
		Role:      role,
	})
}

// ListMembers returns the project's members. Empty slice, not nil, when none.
func (s *ProjectService) ListMembers(ctx context.Context, projectID string) ([]*model.ProjectMember, error) {
	list, err := s.members.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	if list == nil {
		list = []*model.ProjectMember{}
	}
	return list, nil
}

// RoleFor implements middleware.ProjectRoleResolver. Returns:
//   - "admin" if caller is the project owner (no member row needed)
//   - the member's role if caller is a project_members row
//   - ("", nil) if caller is not a member
//   - ("", errs.ErrNotFound) if the project doesn't exist (lets middleware
//     return 404 rather than 403 — clients can distinguish "wrong URL"
//     from "no access")
func (s *ProjectService) RoleFor(ctx context.Context, projectID, userID string) (string, error) {
	p, err := s.projects.FindByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return "", errs.ErrNotFound
		}
		return "", fmt.Errorf("find project for role: %w", err)
	}
	if p.OwnerID == userID {
		return model.ProjectRoleAdmin, nil
	}
	m, err := s.members.Find(ctx, projectID, userID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return "", nil
		}
		return "", fmt.Errorf("find member: %w", err)
	}
	return m.Role, nil
}

// isValidProjectRole guards against callers passing arbitrary strings into
// AddMember. The ENUM column on project_members.role would reject unknown
// values at the DB layer, but returning ErrBadRequest here gives a clearer
// 400 instead of a confusing 500 from the driver.
func isValidProjectRole(role string) bool {
	return role == model.ProjectRoleAdmin ||
		role == model.ProjectRoleEngineer ||
		role == model.ProjectRoleViewer
}