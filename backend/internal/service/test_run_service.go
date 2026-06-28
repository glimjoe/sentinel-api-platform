package service

import (
	"context"
	"fmt"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/id"
	"github.com/glimjoe/sentinel-api-platform/internal/runner"
)

type testRunStore interface {
	Create(ctx context.Context, run *model.TestRun) error
	FindByID(ctx context.Context, id string) (*model.TestRun, error)
	ListByProject(ctx context.Context, projectID string) ([]*model.TestRun, error)
	Update(ctx context.Context, id string, fields map[string]any) error
}

type TestRunService struct {
	store       testRunStore
	caseStore   testCaseStore
	resultStore runner.ResultPersister
	roles       projectRoleChecker
	publisher   runner.EventPublisher
}

func NewTestRunService(store testRunStore, caseStore testCaseStore, resultStore runner.ResultPersister, roles projectRoleChecker, pub runner.EventPublisher) *TestRunService {
	return &TestRunService{store: store, caseStore: caseStore, resultStore: resultStore, roles: roles, publisher: pub}
}

func (s *TestRunService) Create(ctx context.Context, callerID, projectID, name, targetBaseURL, mode string) (*model.TestRun, error) {
	role, err := s.roles.RoleFor(ctx, projectID, callerID)
	if err != nil {
		return nil, fmt.Errorf("check role: %w", err)
	}
	if role != model.ProjectRoleAdmin && role != model.ProjectRoleEngineer {
		return nil, errs.ErrForbidden
	}
	run := &model.TestRun{
		ID:            id.New(),
		ProjectID:     projectID,
		Name:          name,
		TargetBaseURL: targetBaseURL,
		Mode:          mode,
		Status:        "queued",
		TriggeredBy:   &callerID,
		TriggerType:   "manual",
	}
	if err := s.store.Create(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

func (s *TestRunService) Start(ctx context.Context, callerID, runID string) (*model.TestRun, error) {
	run, err := s.store.FindByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run.Status != "queued" {
		return nil, fmt.Errorf("run %s is already %s", runID, run.Status)
	}
	cases, err := s.caseStore.ListByProject(ctx, run.ProjectID)
	if err != nil {
		return nil, err
	}
	if len(cases) == 0 {
		return nil, errs.ErrBadRequest // no test cases to run
	}

	// Run synchronously for now (async via goroutine in handler)
	if err := runner.Run(ctx, run, cases, run.TargetBaseURL, s.resultStore, s.store, s.publisher); err != nil {
		return nil, err
	}
	return s.store.FindByID(ctx, runID)
}

func (s *TestRunService) FindByID(ctx context.Context, id string) (*model.TestRun, error) {
	return s.store.FindByID(ctx, id)
}

func (s *TestRunService) ListByProject(ctx context.Context, projectID string) ([]*model.TestRun, error) {
	return s.store.ListByProject(ctx, projectID)
}
