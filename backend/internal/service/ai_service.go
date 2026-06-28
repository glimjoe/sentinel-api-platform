package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/glimjoe/sentinel-api-platform/internal/ai"
	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

// AIService coordinates AI operations with RBAC and data access.
type AIService struct {
	roles       projectRoleChecker
	attributor  *ai.Attributor
	completer   *ai.Completer
	prioritizer *ai.Prioritizer
	apiStore    apiFinder
	caseStore   caseAccessor
	guard       *ai.Guard
}

// apiFinder is a subset of the API repo.
type apiFinder interface {
	ListByProject(ctx context.Context, projectID string) ([]*model.API, error)
}

// caseAccessor reads and updates test cases.
type caseAccessor interface {
	ListByProject(ctx context.Context, projectID string) ([]*model.TestCase, error)
	FindByID(ctx context.Context, id string) (*model.TestCase, error)
	Update(ctx context.Context, id string, fields map[string]any) error
}

// NewAIService constructs an AIService.
func NewAIService(roles projectRoleChecker, attributor *ai.Attributor, completer *ai.Completer, prioritizer *ai.Prioritizer, apiStore apiFinder, caseStore caseAccessor, guard *ai.Guard) *AIService {
	return &AIService{roles: roles, attributor: attributor, completer: completer, prioritizer: prioritizer, apiStore: apiStore, caseStore: caseStore, guard: guard}
}

// Attribute runs failure attribution on a test result (passed as JSON).
func (s *AIService) Attribute(ctx context.Context, callerID, projectID, resultJSON string) (*ai.AttributionResult, error) {
	role, err := s.roles.RoleFor(ctx, projectID, callerID)
	if err != nil {
		return nil, errs.ErrForbidden
	}
	if role != model.ProjectRoleAdmin && role != model.ProjectRoleEngineer {
		return nil, errs.ErrForbidden
	}
	return s.attributor.Attribute(ctx, resultJSON)
}

// Complete generates test cases from API specs in the project.
func (s *AIService) Complete(ctx context.Context, callerID, projectID string, apiID *string) ([]ai.GeneratedCase, error) {
	role, err := s.roles.RoleFor(ctx, projectID, callerID)
	if err != nil {
		return nil, errs.ErrForbidden
	}
	if role != model.ProjectRoleAdmin && role != model.ProjectRoleEngineer {
		return nil, errs.ErrForbidden
	}
	apis, err := s.apiStore.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list apis: %w", err)
	}
	if apiID != nil {
		filtered := make([]*model.API, 0)
		for _, a := range apis {
			if a.ID == *apiID {
				filtered = append(filtered, a)
				break
			}
		}
		apis = filtered
	}
	apiJSON, err := json.Marshal(apis)
	if err != nil {
		return nil, fmt.Errorf("marshal apis: %w", err)
	}
	cases, err := s.caseStore.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list existing cases: %w", err)
	}
	caseJSON, err := json.Marshal(cases)
	if err != nil {
		return nil, fmt.Errorf("marshal cases: %w", err)
	}
	return s.completer.Complete(ctx, string(apiJSON), string(caseJSON))
}

// Prioritize suggests priorities for test cases.
func (s *AIService) Prioritize(ctx context.Context, callerID, projectID string, caseIDs []string) ([]ai.PriorityItem, error) {
	role, err := s.roles.RoleFor(ctx, projectID, callerID)
	if err != nil {
		return nil, errs.ErrForbidden
	}
	if role != model.ProjectRoleAdmin && role != model.ProjectRoleEngineer {
		return nil, errs.ErrForbidden
	}
	cases, err := s.caseStore.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list cases: %w", err)
	}
	if len(caseIDs) > 0 {
		idSet := make(map[string]struct{}, len(caseIDs))
		for _, id := range caseIDs {
			idSet[id] = struct{}{}
		}
		filtered := make([]*model.TestCase, 0, len(caseIDs))
		for _, c := range cases {
			if _, ok := idSet[c.ID]; ok {
				filtered = append(filtered, c)
			}
		}
		cases = filtered
	}
	caseJSON, err := json.Marshal(cases)
	if err != nil {
		return nil, fmt.Errorf("marshal cases: %w", err)
	}
	return s.prioritizer.Prioritize(ctx, string(caseJSON))
}

// Budget returns current usage info.
func (s *AIService) Budget(ctx context.Context) map[string]any {
	daily, _ := s.guard.DailyUsage(ctx)
	monthly, _ := s.guard.MonthlyUsage(ctx)
	return map[string]any{
		"enabled": true,
		"daily":   map[string]any{"used": daily, "limit": s.guard.DailyLimit()},
		"monthly": map[string]any{"used": monthly, "limit": s.guard.MonthlyLimit()},
	}
}
