//go:build integration

package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/repository"
	"github.com/glimjoe/sentinel-api-platform/internal/runner"
	"github.com/glimjoe/sentinel-api-platform/internal/service"
)

// fakeRunPub implements runner.EventPublisher for integration tests.
type fakeRunPub struct{}

func (f *fakeRunPub) Publish(_ context.Context, _ *runner.RunEvent) error { return nil }

// TestIntegration_RunnerFullFlow: create test cases, start a run, verify results.
type fakeRoleChecker struct{}
func (f *fakeRoleChecker) RoleFor(_ context.Context, _, _ string) (string, error) { return "admin", nil }
func newFakeRoleChecker() *fakeRoleChecker { return &fakeRoleChecker{} }

func TestIntegration_RunnerFullFlow(t *testing.T) {
	if testDB == nil {
		t.Skip("MySQL not available")
	}
	projID := "01JRUNNERINTEG000000001"
	caseID1 := "01JRUNNERCASE0000000001"
	caseID2 := "01JRUNNERCASE0000000002"
	runID := "01JRUNNERRUN00000000001"

	// Setup: project + mock endpoint that the runner will hit
	testDB.Exec("DELETE FROM test_results WHERE run_id = ?", runID)
	testDB.Exec("DELETE FROM test_runs WHERE id = ?", runID)
	testDB.Exec("DELETE FROM test_cases WHERE id IN (?,?)", caseID1, caseID2)
	testDB.Exec("DELETE FROM mock_rules WHERE api_id = '01JRUNNERAPI0000000001'")
	testDB.Exec("DELETE FROM apis WHERE id = '01JRUNNERAPI0000000001'")
	testDB.Exec("DELETE FROM projects WHERE id = ?", projID)

	testDB.Exec("INSERT INTO projects (id, name, slug, owner_id) VALUES (?,?,?,?)", projID, "RunnerTest", "runnertest", "u1")
	testDB.Exec("INSERT INTO apis (id, project_id, name, method, path, source) VALUES (?,?,?,?,?,?)", "01JRUNNERAPI0000000001", projID, "Test", "GET", "/api/health", "manual")
	testDB.Exec("INSERT INTO test_cases (id, project_id, name, method, path, expected_status, expected_body_match) VALUES (?,?,?,?,?,?,?)", caseID1, projID, "health-check", "GET", "/api/health", 200, "none")
	testDB.Exec("INSERT INTO test_cases (id, project_id, name, method, path, expected_status, expected_body_match) VALUES (?,?,?,?,?,?,?)", caseID2, projID, "should-fail", "GET", "/api/health", 201, "none")

	// Start a local test server as the target
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer ts.Close()

	runStore := repository.NewTestRunRepo(testDB)
	resultStore := repository.NewTestResultRepo(testDB)
	caseRepo := repository.NewTestCaseRepo(testDB)
	_ = service.NewTestRunService(runStore, caseRepo, resultStore, newFakeRoleChecker(), &fakeRunPub{})

	run := &model.TestRun{
		ID: runID, ProjectID: projID, Name: "integration-run",
		Status: "queued", Mode: "sequential", TargetBaseURL: ts.URL,
		TriggerType: "manual",
	}
	uid := "u1"
	run.TriggeredBy = &uid
	testDB.Create(run)

	ctx := context.Background()
	cases, _ := caseRepo.ListByProject(ctx, projID)
	require.NotEmpty(t, cases)

	// Run synchronously (no goroutine in integration test)
	err := runner.Run(ctx, run, cases, ts.URL, resultStore, runStore, &fakeRunPub{}, nil)
	require.NoError(t, err)

	// Verify results
	results, _ := resultStore.ListByRun(ctx, runID)
	assert.Len(t, results, 2)
	passCount := 0
	failCount := 0
	for _, r := range results {
		if r.Status == "pass" {
			passCount++
		} else if r.Status == "fail" {
			failCount++
		}
	}
	assert.Equal(t, 1, passCount, "expected 1 pass")
	assert.Equal(t, 1, failCount, "expected 1 fail for mismatched status")

	// Cleanup
	testDB.Exec("DELETE FROM test_results WHERE run_id = ?", runID)
	testDB.Exec("DELETE FROM test_runs WHERE id = ?", runID)
	testDB.Exec("DELETE FROM test_cases WHERE id IN (?,?)", caseID1, caseID2)
	testDB.Exec("DELETE FROM mock_rules WHERE api_id = '01JRUNNERAPI0000000001'")
	testDB.Exec("DELETE FROM apis WHERE id = '01JRUNNERAPI0000000001'")
	testDB.Exec("DELETE FROM projects WHERE id = ?", projID)
}
