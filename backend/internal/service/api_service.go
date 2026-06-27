// Package service — api_service stub (M2-E TDD RED).
//
// APIService handles CRUD on the apis table, with RBAC checks delegated to
// a projectRoleChecker (which ProjectService.RoleFor implements — wired in
// main.go). Keeping RBAC behind an interface (rather than calling
// ProjectService directly) avoids a cycle: api_service doesn't need to
// import ProjectService, just an interface that resolves roles.
//
// Persistence contract (apiStore) is declared here so tests can supply a
// fake without *gorm.DB. Matches project_service's projectStore pattern.
//
// M2 TDD: this file is the RED-phase stub. Methods (Create, FindByID,
// ListByProject, Delete) land once api_service_test.go fails for the
// expected reason (undefined: APIService methods).
package service

import (
	"context"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
)

// apiStore is the persistence contract APIService needs from the apis
// table. Mirrors the four CRUD methods on repository.APIRepo.
type apiStore interface {
	Create(ctx context.Context, a *model.API) error
	FindByID(ctx context.Context, id string) (*model.API, error)
	ListByProject(ctx context.Context, projectID string) ([]*model.API, error)
	Delete(ctx context.Context, id string) error
}

// projectRoleChecker is the contract APIService needs from "something that
// can tell me the caller's project role". ProjectService.RoleFor satisfies
// it (see project_service.go). The interface stays tiny so other resolvers
// (e.g. a future in-memory cache) can plug in without touching this file.
type projectRoleChecker interface {
	RoleFor(ctx context.Context, projectID, userID string) (string, error)
}

// APIService is the entry point for API CRUD business logic.
type APIService struct{}

// NewAPIService wires an APIService. Caller owns the lifetime.
func NewAPIService(apis apiStore, roles projectRoleChecker) *APIService {
	_ = apis
	_ = roles
	return &APIService{}
}