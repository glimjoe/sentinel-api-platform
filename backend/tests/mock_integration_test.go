//go:build integration

package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mockPkg "github.com/glimjoe/sentinel-api-platform/internal/mock"
	"github.com/glimjoe/sentinel-api-platform/internal/repository"
)

// TestIntegration_MockMatcher: full mock engine flow with real DB — create project,
// register APIs, add rules, then hit the mock endpoint and verify matching.

// dummyVarBag is a no-op var bag for integration testing.
type dummyVarBag struct{}

func (d *dummyVarBag) Set(_ context.Context, _, _, _, _ string) error { return nil }
func (d *dummyVarBag) Get(_ context.Context, _, _, _ string) (string, bool, error) { return "", false, nil }
func (d *dummyVarBag) GetAll(_ context.Context, _, _ string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (d *dummyVarBag) Substitute(_ context.Context, _, _, body string) (string, error) { return body, nil }

func TestIntegration_MockMatcher(t *testing.T) {
	if testDB == nil {
		t.Skip("MySQL not available")
	}
	// Seed: project + API + rules
	projID := "01JMOCKINTEGRATION0000001"
	apiID := "01JMOCKINTEGRATIONAPI0001"
	ruleID1 := "01JMOCKRULE00000000001"
	ruleID2 := "01JMOCKRULE00000000002"

	testDB.Exec("DELETE FROM mock_rules WHERE id IN (?,?)", ruleID1, ruleID2)
	testDB.Exec("DELETE FROM apis WHERE id = ?", apiID)
	testDB.Exec("DELETE FROM projects WHERE id = ?", projID)

	testDB.Exec("INSERT INTO projects (id, name, slug, owner_id) VALUES (?,?,?,?)", projID, "MockTest", "mocktest", "u1")
	testDB.Exec("INSERT INTO apis (id, project_id, name, method, path, source) VALUES (?,?,?,?,?,?)", apiID, projID, "TestAPI", "GET", "/api/test", "manual")
	testDB.Exec("INSERT INTO mock_rules (id, api_id, name, match_json, response_status, response_body_json, priority, enabled) VALUES (?,?,?,?,?,?,?,?)", ruleID1, apiID, "rule1", json.RawMessage(`{"query":{"q":"hello"}}`), 200, json.RawMessage(`{"ok":true}`), 100, true)
	testDB.Exec("INSERT INTO mock_rules (id, api_id, name, match_json, response_status, priority, enabled) VALUES (?,?,?,?,?,?,?)", ruleID2, apiID, "catchall", json.RawMessage(`{}`), 404, 200, true)

	projRepo := repository.NewProjectRepo(testDB)
	mockRuleRepo := repository.NewMockRuleRepo(testDB)
	mockHitRepo := repository.NewMockHitRepo(testDB)
	apiRepo := repository.NewAPIRepo(testDB)
	matcher := mockPkg.NewMatcher()
	extractor := mockPkg.NewExtractor()
	engine := mockPkg.NewEngine(projRepo, apiRepo, mockRuleRepo, &dummyVarBag{}, matcher, extractor, mockHitRepo)

	r := gin.New()
	r.Any("/mock/:slug/*path", engine.Serve)

	// Hit with matching query param — should match rule1 (higher priority)
	req := httptest.NewRequest(http.MethodGet, "/mock/mocktest/api/test?q=hello", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, true, resp["ok"])

	// Hit without query — should match catchall
	req = httptest.NewRequest(http.MethodGet, "/mock/mocktest/api/test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)

	// Cleanup
	testDB.Exec("DELETE FROM mock_hits WHERE mock_rule_id IN (?,?)", ruleID1, ruleID2)
	testDB.Exec("DELETE FROM mock_rules WHERE id IN (?,?)", ruleID1, ruleID2)
	testDB.Exec("DELETE FROM apis WHERE id = ?", apiID)
	testDB.Exec("DELETE FROM projects WHERE id = ?", projID)
}
