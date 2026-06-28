package service

import (
	"context"
	"fmt"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/id"
)

type testCaseStore interface {
	Create(ctx context.Context, tc *model.TestCase) error
	FindByID(ctx context.Context, id string) (*model.TestCase, error)
	ListByProject(ctx context.Context, projectID string) ([]*model.TestCase, error)
	Update(ctx context.Context, id string, fields map[string]any) error
	Delete(ctx context.Context, id string) error
}

type TestCaseService struct {
	store testCaseStore
	roles projectRoleChecker
}

func NewTestCaseService(store testCaseStore, roles projectRoleChecker) *TestCaseService {
	return &TestCaseService{store: store, roles: roles}
}

func (s *TestCaseService) Create(ctx context.Context, callerID, projectID string, tc *model.TestCase) (*model.TestCase, error) {
	role, err := s.roles.RoleFor(ctx, projectID, callerID)
	if err != nil {
		return nil, fmt.Errorf("check role: %w", err)
	}
	if role != model.ProjectRoleAdmin && role != model.ProjectRoleEngineer {
		return nil, errs.ErrForbidden
	}
	tc.ID = id.New()
	tc.ProjectID = projectID
	if err := s.store.Create(ctx, tc); err != nil {
		return nil, err
	}
	return tc, nil
}

func (s *TestCaseService) ListByProject(ctx context.Context, projectID string) ([]*model.TestCase, error) {
	return s.store.ListByProject(ctx, projectID)
}

func (s *TestCaseService) FindByID(ctx context.Context, id string) (*model.TestCase, error) {
	return s.store.FindByID(ctx, id)
}

func (s *TestCaseService) Update(ctx context.Context, callerID, projectID, caseID string, fields map[string]any) (*model.TestCase, error) {
	role, err := s.roles.RoleFor(ctx, projectID, callerID)
	if err != nil {
		return nil, fmt.Errorf("check role: %w", err)
	}
	if role != model.ProjectRoleAdmin && role != model.ProjectRoleEngineer {
		return nil, errs.ErrForbidden
	}
	if err := s.store.Update(ctx, caseID, fields); err != nil {
		return nil, err
	}
	return s.store.FindByID(ctx, caseID)
}

func (s *TestCaseService) Delete(ctx context.Context, callerID, projectID, caseID string) error {
	role, err := s.roles.RoleFor(ctx, projectID, callerID)
	if err != nil {
		return fmt.Errorf("check role: %w", err)
	}
	if role != model.ProjectRoleAdmin && role != model.ProjectRoleEngineer {
		return errs.ErrForbidden
	}
	return s.store.Delete(ctx, caseID)
}
