// Package service — project_service stub (M2-E TDD RED).
//
// ProjectService is the entry point for project CRUD + membership business
// logic. It also implements middleware.ProjectRoleResolver via RoleFor, so
// the RBAC middleware and other services can ask "what role does user X
// have on project Y?" without each importing the repository layer.
//
// Persistence contracts (projectStore, projectMemberStore) are declared in
// this file (not in repository/) so tests can supply fakes without dragging
// *gorm.DB into the service test. The pattern matches auth_service's
// userStore (auth_service.go L24-28).
//
// M2 TDD: this file is the RED-phase stub. The methods (Create, FindByID,
// FindBySlug, List, Update, Delete, AddMember, ListMembers, RoleFor) land
// once project_service_test.go fails for the expected reason (undefined:
// ProjectService methods).
package service

import (
	"context"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

// projectStore is the persistence contract ProjectService needs from the
// projects table. Mirrors the pattern used by auth_service's userStore.
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

// ProjectService is the entry point for project + member business logic.
// Methods will be defined in the GREEN phase.
type ProjectService struct{}

// NewProjectService wires a ProjectService. Returns *ProjectService so it
// composes with other services via interfaces. Caller owns the lifetime.
func NewProjectService(projects projectStore, members projectMemberStore) *ProjectService {
	_ = projects
	_ = members
	return &ProjectService{}
}