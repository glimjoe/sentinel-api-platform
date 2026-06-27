// Package service — APIService (M2-E).
//
// APIService handles CRUD on the apis table, with RBAC checks delegated to
// a projectRoleChecker (which ProjectService.RoleFor satisfies — wired in
// main.go). Keeping RBAC behind an interface rather than calling
// ProjectService directly avoids a service-to-service import and lets a
// future in-memory cache plug in without touching this file.
//
// Authorization:
//   - Create: admin OR engineer (engineers wire up endpoints)
//   - Delete: admin only (destructive; engineers can edit but not remove)
//   - Read (FindByID, ListByProject): no RBAC — the middleware in front of
//     the handler enforces it. Keeps this layer thin.
//   - Update (PATCH endpoint): deferred — APIRepo has no Update yet; will
//     land when M2-F wires the PATCH handler.
//
// Persistence contract (apiStore) is declared here so tests can supply a
// fake without *gorm.DB. Matches project_service's projectStore pattern.
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

// apiStore is the persistence contract APIService needs from the apis
// table. Mirrors the four CRUD methods on repository.APIRepo.
type apiStore interface {
	Create(ctx context.Context, a *model.API) error
	FindByID(ctx context.Context, id string) (*model.API, error)
	ListByProject(ctx context.Context, projectID string) ([]*model.API, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, id string, fields map[string]any) error
}

// projectRoleChecker is the contract APIService needs from "something that
// can tell me the caller's project role". ProjectService.RoleFor satisfies
// it (see project_service.go).
type projectRoleChecker interface {
	RoleFor(ctx context.Context, projectID, userID string) (string, error)
}

// validMethods mirrors the ENUM in migrations/0002_projects_and_mock.sql.
// Validated client-side so callers get a clear 400 instead of a confusing
// MySQL data-truncation error.
var validMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true,
	"PATCH": true, "DELETE": true, "HEAD": true, "OPTIONS": true,
}

// APIService wires the API CRUD business logic.
type APIService struct {
	apis  apiStore
	roles projectRoleChecker
}

// NewAPIService constructs an APIService. Caller owns the lifetime.
func NewAPIService(apis apiStore, roles projectRoleChecker) *APIService {
	return &APIService{apis: apis, roles: roles}
}

// Create registers a new API endpoint under projectID. Caller must be
// admin or engineer on the project.
//
// Errors:
//   - errs.ErrForbidden  — caller is not admin/engineer
//   - errs.ErrBadRequest — empty name/path, or method not in HTTP enum
//   - errs.ErrConflict   — (project_id, method, path) already exists
func (s *APIService) Create(ctx context.Context, callerID, projectID, name, method, path, operationID string) (*model.API, error) {
	callerRole, err := s.roles.RoleFor(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if callerRole != model.ProjectRoleAdmin && callerRole != model.ProjectRoleEngineer {
		return nil, errs.ErrForbidden
	}

	name = strings.TrimSpace(name)
	method = strings.ToUpper(strings.TrimSpace(method))
	path = strings.TrimSpace(path)
	if name == "" || path == "" {
		return nil, fmt.Errorf("%w: name and path are required", errs.ErrBadRequest)
	}
	if !validMethods[method] {
		return nil, fmt.Errorf("%w: method must be GET/POST/PUT/PATCH/DELETE/HEAD/OPTIONS", errs.ErrBadRequest)
	}

	a := &model.API{
		ID:          id.New(),
		ProjectID:   projectID,
		Name:        name,
		Method:      method,
		Path:        path,
		OperationID: strings.TrimSpace(operationID),
		Source:      "manual",
	}
	if err := s.apis.Create(ctx, a); err != nil {
		if errors.Is(err, errs.ErrConflict) {
			return nil, errs.ErrConflict
		}
		return nil, fmt.Errorf("create api: %w", err)
	}
	return a, nil
}

// FindByID returns the API with the given id, or errs.ErrNotFound.
func (s *APIService) FindByID(ctx context.Context, id string) (*model.API, error) {
	a, err := s.apis.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("find api: %w", err)
	}
	return a, nil
}

// ListByProject returns all APIs in the given project. Empty slice, not
// nil, when none — so handlers can iterate without a nil check.
func (s *APIService) ListByProject(ctx context.Context, projectID string) ([]*model.API, error) {
	list, err := s.apis.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list apis: %w", err)
	}
	if list == nil {
		list = []*model.API{}
	}
	return list, nil
}

// Delete removes an API by id. Caller must be admin on the API's project
// (engineers can't delete — destructive operation).
//
// Errors:
//   - errs.ErrNotFound  — api doesn't exist
//   - errs.ErrForbidden — caller is not admin on api.ProjectID
func (s *APIService) Delete(ctx context.Context, callerID, apiID string) error {
	a, err := s.apis.FindByID(ctx, apiID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return errs.ErrNotFound
		}
		return fmt.Errorf("find api for delete: %w", err)
	}
	callerRole, err := s.roles.RoleFor(ctx, a.ProjectID, callerID)
	if err != nil {
		return err
	}
	if callerRole != model.ProjectRoleAdmin {
		return errs.ErrForbidden
	}
	if err := s.apis.Delete(ctx, apiID); err != nil {
		return fmt.Errorf("delete api: %w", err)
	}
	return nil
}
// Update patches the API with the given id. Caller must be admin or
// engineer on the API's project.
//
// Errors:
//   - errs.ErrNotFound  — api doesn't exist
//   - errs.ErrForbidden — caller is not admin/engineer
func (s *APIService) Update(ctx context.Context, callerID, apiID string, fields map[string]any) (*model.API, error) {
	a, err := s.apis.FindByID(ctx, apiID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("find api for update: %w", err)
	}
	callerRole, err := s.roles.RoleFor(ctx, a.ProjectID, callerID)
	if err != nil {
		return nil, err
	}
	if callerRole != model.ProjectRoleAdmin && callerRole != model.ProjectRoleEngineer {
		return nil, errs.ErrForbidden
	}
	if err := s.apis.Update(ctx, apiID, fields); err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("update api: %w", err)
	}
	return s.apis.FindByID(ctx, apiID)
}
