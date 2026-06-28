package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

// -----------------------------------------------------------------------------
// Fakes
// -----------------------------------------------------------------------------

// fakeMockRuleStore tracks rows by id and per-(api, name) uniqueness so
// tests can assert duplicate-name rejection without GORM. Update signature
// matches MockRuleRepo.Update (ctx, id, fields map).
type fakeMockRuleStore struct {
	createErr  error // injected
	rows       map[string]*model.MockRule
	namesByAPI map[string]map[string]string // apiID → name → rule id
}

func newFakeMockRuleStore() *fakeMockRuleStore {
	return &fakeMockRuleStore{
		rows:       map[string]*model.MockRule{},
		namesByAPI: map[string]map[string]string{},
	}
}

func (f *fakeMockRuleStore) Create(_ context.Context, r *model.MockRule) error {
	if f.createErr != nil {
		return f.createErr
	}
	if f.namesByAPI[r.APIID] == nil {
		f.namesByAPI[r.APIID] = map[string]string{}
	}
	if _, exists := f.namesByAPI[r.APIID][r.Name]; exists {
		return errs.ErrConflict
	}
	f.rows[r.ID] = r
	f.namesByAPI[r.APIID][r.Name] = r.ID
	return nil
}

func (f *fakeMockRuleStore) FindByID(_ context.Context, id string) (*model.MockRule, error) {
	r, ok := f.rows[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return r, nil
}

func (f *fakeMockRuleStore) ListByAPI(_ context.Context, apiID string) ([]*model.MockRule, error) {
	var list []*model.MockRule
	for _, r := range f.rows {
		if r.APIID == apiID {
			list = append(list, r)
		}
	}
	return list, nil
}

func (f *fakeMockRuleStore) Update(_ context.Context, id string, fields map[string]any) error {
	r, ok := f.rows[id]
	if !ok {
		return errs.ErrNotFound
	}
	// Apply only the well-known keys; mirrors the partial-update semantics
	// of MockRuleRepo.Update (which uses gorm Updates(map)).
	if v, ok := fields["priority"].(int); ok {
		r.Priority = v
	}
	if v, ok := fields["enabled"].(bool); ok {
		r.Enabled = v
	}
	if v, ok := fields["name"].(string); ok {
		r.Name = v
	}
	return nil
}

func (f *fakeMockRuleStore) IncrementHitCount(_ context.Context, id string) error {
	if r, ok := f.rows[id]; ok {
		r.HitCount++
	}
	return nil
}

// fakeMockRoleChecker mirrors the fakeRoleChecker from api_service_test.go.
// Re-declared here so this file can run in isolation.
type fakeMockRoleChecker struct {
	roleByUser map[string]string
}

func newFakeMockRoleChecker() *fakeMockRoleChecker {
	return &fakeMockRoleChecker{roleByUser: map[string]string{}}
}

func (f *fakeMockRoleChecker) RoleFor(_ context.Context, projectID, userID string) (string, error) {
	return f.roleByUser[projectID+":"+userID], nil
}

// -----------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------

// TestMockRuleService_Create_HappyPath — engineer creates a rule.
// NB: model.MockRule has no ProjectID field — only APIID. Project ownership
// is enforced via RBAC + the rule's API's parent project.
func TestMockRuleService_Create_HappyPath(t *testing.T) {
	rules, roles := newFakeMockRuleStore(), newFakeMockRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleEngineer
	svc := NewMockRuleService(rules, roles)

	spec := CreateRuleSpec{
		Name:           "available-pets",
		MatchJSON:      json.RawMessage(`{"query":{"status":"available"}}`),
		ResponseStatus: 200,
		ResponseBody:   json.RawMessage(`{"pets":[]}`),
		Priority:       10,
	}
	r, err := svc.Create(context.Background(), "user-1", "proj-1", "api-1", spec)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if r.Name != "available-pets" || r.APIID != "api-1" {
		t.Errorf("rule = %+v", r)
	}
	if r.Priority != 10 || r.ResponseStatus != 200 {
		t.Errorf("rule fields wrong: %+v", r)
	}
	if !r.Enabled {
		t.Error("new rules should default to enabled=true")
	}
}

// TestMockRuleService_Create_ViewerForbidden — viewer can't add rules.
func TestMockRuleService_Create_ViewerForbidden(t *testing.T) {
	rules, roles := newFakeMockRuleStore(), newFakeMockRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleViewer
	svc := NewMockRuleService(rules, roles)

	_, err := svc.Create(context.Background(), "user-1", "proj-1", "api-1", CreateRuleSpec{Name: "x"})
	if !errors.Is(err, errs.ErrForbidden) {
		t.Errorf("err = %v, want ErrForbidden", err)
	}
}

// TestMockRuleService_Create_BadMatchJSON — non-JSON match_json → ErrBadRequest
// (so the engine never sees a broken schema and can return its 422 cleanly).
func TestMockRuleService_Create_BadMatchJSON(t *testing.T) {
	rules, roles := newFakeMockRuleStore(), newFakeMockRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleAdmin
	svc := NewMockRuleService(rules, roles)

	_, err := svc.Create(context.Background(), "user-1", "proj-1", "api-1", CreateRuleSpec{
		Name:      "broken",
		MatchJSON: json.RawMessage(`not valid json`),
	})
	if !errors.Is(err, errs.ErrBadRequest) {
		t.Errorf("err = %v, want ErrBadRequest", err)
	}
}

// TestMockRuleService_Create_DuplicateNamePerAPI — second rule with same
// (api, name) → ErrConflict. Name is the human-friendly label within an
// API's rule list; uniqueness is per-API, not global.
func TestMockRuleService_Create_DuplicateNamePerAPI(t *testing.T) {
	rules, roles := newFakeMockRuleStore(), newFakeMockRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleAdmin
	svc := NewMockRuleService(rules, roles)

	spec := CreateRuleSpec{Name: "same", MatchJSON: json.RawMessage(`{}`)}
	if _, err := svc.Create(context.Background(), "user-1", "proj-1", "api-1", spec); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err := svc.Create(context.Background(), "user-1", "proj-1", "api-1", spec)
	if !errors.Is(err, errs.ErrConflict) {
		t.Errorf("err = %v, want ErrConflict", err)
	}
}

// TestMockRuleService_FindByID_NotFound — unknown id → ErrNotFound.
func TestMockRuleService_FindByID_NotFound(t *testing.T) {
	svc := NewMockRuleService(newFakeMockRuleStore(), newFakeMockRoleChecker())
	_, err := svc.FindByID(context.Background(), "missing")
	if !errors.Is(err, errs.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

// TestMockRuleService_ListByAPI_Empty — empty slice (not nil).
func TestMockRuleService_ListByAPI_Empty(t *testing.T) {
	svc := NewMockRuleService(newFakeMockRuleStore(), newFakeMockRoleChecker())
	list, err := svc.ListByAPI(context.Background(), "api-1")
	if err != nil {
		t.Fatalf("ListByAPI: %v", err)
	}
	if list == nil {
		t.Error("ListByAPI should return empty slice, not nil")
	}
}

// TestMockRuleService_Update_NonAdminForbidden — viewer trying to patch a
// rule → ErrForbidden (engineers can edit, viewers cannot).
func TestMockRuleService_Update_NonAdminForbidden(t *testing.T) {
	rules, roles := newFakeMockRuleStore(), newFakeMockRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleAdmin
	roles.roleByUser["proj-1:user-2"] = model.ProjectRoleViewer
	svc := NewMockRuleService(rules, roles)

	r, _ := svc.Create(context.Background(), "user-1", "proj-1", "api-1", CreateRuleSpec{Name: "x", MatchJSON: json.RawMessage(`{}`)})
	_, err := svc.Update(context.Background(), "user-2", "proj-1", r.ID, map[string]any{"priority": 50})
	if !errors.Is(err, errs.ErrForbidden) {
		t.Errorf("err = %v, want ErrForbidden", err)
	}
}

// TestMockRuleService_IncrementHitCount — engine records hits; counter goes up.
func TestMockRuleService_IncrementHitCount(t *testing.T) {
	rules, roles := newFakeMockRuleStore(), newFakeMockRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleAdmin
	svc := NewMockRuleService(rules, roles)

	r, _ := svc.Create(context.Background(), "user-1", "proj-1", "api-1", CreateRuleSpec{Name: "x", MatchJSON: json.RawMessage(`{}`)})
	if r.HitCount != 0 {
		t.Fatalf("initial HitCount = %d, want 0", r.HitCount)
	}
	for i := 1; i <= 3; i++ {
		if err := svc.IncrementHitCount(context.Background(), r.ID); err != nil {
			t.Fatalf("IncrementHitCount #%d: %v", i, err)
		}
	}
	r2, _ := svc.FindByID(context.Background(), r.ID)
	if r2.HitCount != 3 {
		t.Errorf("HitCount = %d, want 3", r2.HitCount)
	}
}
func (f *fakeMockRuleStore) Delete(_ context.Context, id string) error {
	if _, ok := f.rows[id]; !ok {
		return fmt.Errorf("mock_rule_repo: %w", errs.ErrNotFound)
	}
	delete(f.rows, id)
	return nil
}

func TestMockRuleService_Delete_HappyPath(t *testing.T) {
	rules, roles := newFakeMockRuleStore(), newFakeMockRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleAdmin
	svc := NewMockRuleService(rules, roles)

	r, _ := svc.Create(context.Background(), "user-1", "proj-1", "api-1", CreateRuleSpec{Name: "x", MatchJSON: json.RawMessage(`{}`)})
	err := svc.Delete(context.Background(), "user-1", "proj-1", r.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = svc.FindByID(context.Background(), r.ID)
	if !errors.Is(err, errs.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMockRuleService_Delete_ViewerForbidden(t *testing.T) {
	rules, roles := newFakeMockRuleStore(), newFakeMockRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleAdmin
	roles.roleByUser["proj-1:user-2"] = model.ProjectRoleViewer
	svc := NewMockRuleService(rules, roles)

	r, _ := svc.Create(context.Background(), "user-1", "proj-1", "api-1", CreateRuleSpec{Name: "x", MatchJSON: json.RawMessage(`{}`)})
	err := svc.Delete(context.Background(), "user-2", "proj-1", r.ID)
	if !errors.Is(err, errs.ErrForbidden) {
		t.Errorf("err = %v, want ErrForbidden", err)
	}
}

func TestMockRuleService_GetHitCount(t *testing.T) {
	rules, roles := newFakeMockRuleStore(), newFakeMockRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleAdmin
	svc := NewMockRuleService(rules, roles)

	r, _ := svc.Create(context.Background(), "user-1", "proj-1", "api-1", CreateRuleSpec{Name: "x", MatchJSON: json.RawMessage(`{}`)})
	_ = svc.IncrementHitCount(context.Background(), r.ID)
	_ = svc.IncrementHitCount(context.Background(), r.ID)

	count, err := svc.GetHitCount(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("GetHitCount: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestMockRuleService_GetHitCount_NotFound(t *testing.T) {
	svc := NewMockRuleService(newFakeMockRuleStore(), newFakeMockRoleChecker())
	_, err := svc.GetHitCount(context.Background(), "missing")
	if !errors.Is(err, errs.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestMockRuleService_Update_HappyPath(t *testing.T) {
	rules, roles := newFakeMockRuleStore(), newFakeMockRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleEngineer
	svc := NewMockRuleService(rules, roles)

	r, _ := svc.Create(context.Background(), "user-1", "proj-1", "api-1", CreateRuleSpec{Name: "old", Priority: 10, MatchJSON: json.RawMessage(`{}`)})
	updated, err := svc.Update(context.Background(), "user-1", "proj-1", r.ID, map[string]any{"name": "new", "priority": 99})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "new" {
		t.Errorf("Name = %s, want new", updated.Name)
	}
	if updated.Priority != 99 {
		t.Errorf("Priority = %d, want 99", updated.Priority)
	}
}

func TestMockRuleService_Update_NotFound(t *testing.T) {
	rules, roles := newFakeMockRuleStore(), newFakeMockRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleAdmin
	svc := NewMockRuleService(rules, roles)

	_, err := svc.Update(context.Background(), "user-1", "proj-1", "missing", map[string]any{"priority": 50})
	if !errors.Is(err, errs.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestMockRuleService_HandlerAliases(t *testing.T) {
	rules, roles := newFakeMockRuleStore(), newFakeMockRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleAdmin
	svc := NewMockRuleService(rules, roles)

	// CreateRule → Create
	r, err := svc.CreateRule(context.Background(), "user-1", "proj-1", "api-1", CreateRuleSpec{Name: "rule1", MatchJSON: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}

	// GetRule → FindByID
	r2, err := svc.GetRule(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("GetRule: %v", err)
	}
	if r2.Name != "rule1" {
		t.Errorf("GetRule Name = %s", r2.Name)
	}

	// ListRules → ListByAPI
	list, err := svc.ListRules(context.Background(), "api-1")
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("ListRules len = %d, want 1", len(list))
	}

	// UpdateRule → Update
	updated, err := svc.UpdateRule(context.Background(), "user-1", "proj-1", r.ID, map[string]any{"priority": 42})
	if err != nil {
		t.Fatalf("UpdateRule: %v", err)
	}
	if updated.Priority != 42 {
		t.Errorf("UpdateRule Priority = %d", updated.Priority)
	}

	// RecordHit → IncrementHitCount
	if err := svc.RecordHit(context.Background(), r.ID); err != nil {
		t.Fatalf("RecordHit: %v", err)
	}

	// ListHits → GetHitCount
	count, err := svc.ListHits(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("ListHits: %v", err)
	}
	if count != 1 {
		t.Errorf("ListHits = %d, want 1", count)
	}

	// DeleteRule → Delete
	if err := svc.DeleteRule(context.Background(), "user-1", "proj-1", r.ID); err != nil {
		t.Fatalf("DeleteRule: %v", err)
	}
}

func TestMockRuleService_Delete_RoleForError(t *testing.T) {
	store := newFakeMockRuleStore()
	roles := newFakeRoleChecker()
	roles.err = errs.ErrNotFound
	svc := NewMockRuleService(store, roles)

	err := svc.Delete(context.Background(), "user-1", "proj-1", "rule-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, errs.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestMockRuleService_Create_WithEmptyMatchJSON(t *testing.T) {
	store := newFakeMockRuleStore()
	roles := newFakeRoleChecker()
	roles.roleByUser["proj-1:user-1"] = model.ProjectRoleAdmin
	svc := NewMockRuleService(store, roles)

	rule, err := svc.Create(context.Background(), "user-1", "proj-1", "api-1", CreateRuleSpec{
		Name: "test-rule", MatchJSON: json.RawMessage("{}"), ResponseStatus: 200, Priority: 100,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rule == nil {
		t.Fatal("rule should not be nil")
	}
}

func TestMockRuleService_Create_RoleForError(t *testing.T) {
	store := newFakeMockRuleStore()
	roles := newFakeRoleChecker()
	roles.err = errs.ErrForbidden
	svc := NewMockRuleService(store, roles)

	_, err := svc.Create(context.Background(), "user-1", "proj-1", "api-1", CreateRuleSpec{
		Name: "test", ResponseStatus: 200, Priority: 100, MatchJSON: json.RawMessage(`{}`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, errs.ErrForbidden) {
		t.Errorf("err = %v, want ErrForbidden", err)
	}
}

func TestMockRuleService_Update_RoleForError(t *testing.T) {
	store := newFakeMockRuleStore()
	roles := newFakeRoleChecker()
	roles.err = errs.ErrNotFound
	svc := NewMockRuleService(store, roles)

	_, err := svc.Update(context.Background(), "user-1", "proj-1", "rule-1", map[string]any{"name": "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, errs.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
