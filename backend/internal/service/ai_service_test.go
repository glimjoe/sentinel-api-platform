package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/ai"
	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

type fakeGuardStore struct {
	daily   float64
	monthly float64
}

func (f *fakeGuardStore) RecordUsage(ctx context.Context, m *model.AiUsage) error {
	return nil
}
func (f *fakeGuardStore) GetDailyUsage(ctx context.Context) (float64, error)   { return f.daily, nil }
func (f *fakeGuardStore) GetMonthlyUsage(ctx context.Context) (float64, error) { return f.monthly, nil }

type fakeAIAPIStore struct {
	apis []*model.API
}

func (f *fakeAIAPIStore) ListByProject(ctx context.Context, projectID string) ([]*model.API, error) {
	return f.apis, nil
}

func newTestAIService(t *testing.T) *AIService {
	t.Helper()
	engine := ai.NewEngine(&ai.MockProvider{}, nil, ai.NewGuard(&fakeGuardStore{}, 100, 500), 1024, 0.3)
	roles := newFakeRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleEngineer
	roles.roleByUser["proj-1:user-2"] = model.ProjectRoleViewer
	return NewAIService(roles,
		ai.NewAttributor(engine),
		ai.NewCompleter(engine),
		ai.NewPrioritizer(engine),
		&fakeAIAPIStore{apis: []*model.API{{ID: "api-1", Path: "/api/test", Method: "GET"}}},
		newFakeCaseStore(),
		ai.NewGuard(&fakeGuardStore{}, 100, 500),
	)
}

func TestAIService_Attribute(t *testing.T) {
	svc := newTestAIService(t)

	result, err := svc.Attribute(context.Background(), "user-1", "proj-1", `{"status":"fail"}`)
	require.NoError(t, err)
	assert.NotZero(t, result.Confidence)
	assert.NotEmpty(t, result.RootCause)
}

func TestAIService_Attribute_Forbidden(t *testing.T) {
	svc := newTestAIService(t)
	_, err := svc.Attribute(context.Background(), "user-2", "proj-1", `{"status":"fail"}`)
	require.Error(t, err)
	assert.ErrorIs(t, err, errs.ErrForbidden)
}

func TestAIService_Complete(t *testing.T) {
	svc := newTestAIService(t)

	cases, err := svc.Complete(context.Background(), "user-1", "proj-1", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, cases)
}

func TestAIService_Complete_Forbidden(t *testing.T) {
	svc := newTestAIService(t)
	_, err := svc.Complete(context.Background(), "user-2", "proj-1", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, errs.ErrForbidden)
}

func TestAIService_Prioritize(t *testing.T) {
	svc := newTestAIService(t)

	items, err := svc.Prioritize(context.Background(), "user-1", "proj-1", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, items)
}

func TestAIService_Prioritize_Forbidden(t *testing.T) {
	svc := newTestAIService(t)
	_, err := svc.Prioritize(context.Background(), "user-2", "proj-1", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, errs.ErrForbidden)
}

func TestAIService_Budget(t *testing.T) {
	svc := newTestAIService(t)

	budget := svc.Budget(context.Background())
	assert.Equal(t, true, budget["enabled"])
	daily := budget["daily"].(map[string]any)
	assert.Equal(t, float64(0), daily["used"])
	assert.Equal(t, float64(100), daily["limit"])
}

func TestAIService_Complete_WithAPIIDFilter(t *testing.T) {
	svc := newTestAIService(t)

	apiID := "api-1"
	cases, err := svc.Complete(context.Background(), "user-1", "proj-1", &apiID)
	require.NoError(t, err)
	assert.NotEmpty(t, cases)
}

func TestAIService_Prioritize_WithCaseIDs(t *testing.T) {
	svc := newTestAIService(t)

	// Add a test case to filter by
	store := svc.caseStore.(*fakeCaseStore)
	store.rows["c1"] = &model.TestCase{ID: "c1", ProjectID: "proj-1", Name: "tc1"}

	items, err := svc.Prioritize(context.Background(), "user-1", "proj-1", []string{"c1"})
	require.NoError(t, err)
	assert.NotEmpty(t, items)
}

func TestAIService_Attribute_RoleForError(t *testing.T) {
	engine := ai.NewEngine(&ai.MockProvider{}, nil, ai.NewGuard(&fakeGuardStore{}, 100, 500), 1024, 0.3)
	roles := newFakeRoleChecker()
	roles.err = errs.ErrNotFound // simulate project not found
	svc := NewAIService(roles,
		ai.NewAttributor(engine),
		ai.NewCompleter(engine),
		ai.NewPrioritizer(engine),
		&fakeAIAPIStore{apis: []*model.API{{ID: "api-1", Path: "/api/test", Method: "GET"}}},
		newFakeCaseStore(),
		ai.NewGuard(&fakeGuardStore{}, 100, 500),
	)

	_, err := svc.Attribute(context.Background(), "user-1", "proj-1", `{"status":"fail"}`)
	require.Error(t, err)
	assert.ErrorIs(t, err, errs.ErrForbidden)
}
