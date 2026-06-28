package mock

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

// ─── Test fakes ─────────────────────────────────────────────────────────────

type fakeProjects struct {
	bySlug map[string]*model.Project
	err    error
}

func (f *fakeProjects) FindBySlug(_ context.Context, slug string) (*model.Project, error) {
	if f.err != nil {
		return nil, f.err
	}
	p, ok := f.bySlug[slug]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return p, nil
}

type fakeAPIs struct {
	items []*model.API
}

func (f *fakeAPIs) ListByProject(_ context.Context, _ string) ([]*model.API, error) {
	return f.items, nil
}

type fakeRules struct {
	items []*model.MockRule
}

func (f *fakeRules) ListByProject(_ context.Context, _ string) ([]*model.MockRule, error) {
	return f.items, nil
}

// ─── Test helper ────────────────────────────────────────────────────────────

func newTestEngine(t *testing.T, projects ProjectFinder, apis APIFinder, rules RuleFinder) (*Engine, *RedisVarBag) {
	t.Helper()
	bag, _ := newTestBag(t)
	e := NewEngine(projects, apis, rules, bag, NewMatcher(), NewExtractor(), nil)
	return e, bag
}

// gginRequest builds a gin.Context with a request, mimicking what gin would
// do after route matching. We do this by going through gin.New() and a
// real route so URL params get populated correctly.
func gginRequest(t *testing.T, e *Engine, method, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Any("/mock/:projectSlug/*path", e.Serve)

	req := httptest.NewRequest(method, path, io.NopCloser(bytes.NewReader([]byte(body))))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ─── Tests ──────────────────────────────────────────────────────────────────

func TestEngine_ProjectNotFound_Returns404(t *testing.T) {
	e, _ := newTestEngine(t, &fakeProjects{bySlug: map[string]*model.Project{}}, &fakeAPIs{}, &fakeRules{})

	w := gginRequest(t, e, "GET", "/mock/nope/pet/findByStatus", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestEngine_NoMatchingRule_Returns404(t *testing.T) {
	proj := &model.Project{ID: "p1", Slug: "petstore"}
	e, _ := newTestEngine(t,
		&fakeProjects{bySlug: map[string]*model.Project{"petstore": proj}},
		&fakeAPIs{items: []*model.API{
			{ID: "api1", ProjectID: "p1", Method: "GET", Path: "/pet/findByStatus"},
		}},
		&fakeRules{items: []*model.MockRule{
			// Rule expects status=available, but we send status=pending →
			// match_json scores 0 and the catchall branch is not present.
			{ID: "r1", APIID: "api1", MatchJSON: []byte(`{"query":{"status":"available"}}`), ResponseStatus: 200, Enabled: true, Priority: 100},
		}},
	)

	w := gginRequest(t, e, "GET", "/mock/petstore/pet/findByStatus?status=pending", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestEngine_HappyPath_ReturnsConfiguredResponse(t *testing.T) {
	proj := &model.Project{ID: "p1", Slug: "petstore"}
	api := &model.API{ID: "api1", ProjectID: "p1", Method: "GET", Path: "/pet/findByStatus"}
	rule := &model.MockRule{
		ID:              "r1",
		APIID:           "api1",
		MatchJSON:       []byte(`{"query":{"status":"available"}}`),
		ResponseStatus:  200,
		ResponseBodyJSON: []byte(`{"pets":[],"status":"available"}`),
		Enabled:         true,
		Priority:        100,
	}
	e, _ := newTestEngine(t,
		&fakeProjects{bySlug: map[string]*model.Project{"petstore": proj}},
		&fakeAPIs{items: []*model.API{api}},
		&fakeRules{items: []*model.MockRule{rule}},
	)

	w := gginRequest(t, e, "GET", "/mock/petstore/pet/findByStatus?status=available", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"pets":[],"status":"available"}`, w.Body.String())
	assert.NotEmpty(t, w.Header().Get("X-Mock-Session"), "session header must be set")
	assert.Equal(t, "exact", w.Header().Get("X-Mock-Match-Reason"))
	assert.Equal(t, "r1", w.Header().Get("X-Mock-Rule-Id"))
}

func TestEngine_BadMatchJSON_Returns422(t *testing.T) {
	proj := &model.Project{ID: "p1", Slug: "petstore"}
	api := &model.API{ID: "api1", ProjectID: "p1", Method: "GET", Path: "/x"}
	rule := &model.MockRule{
		ID:        "r1",
		APIID:     "api1",
		MatchJSON: []byte(`{"query":{"q":{}}}`), // empty spec → 422
		Enabled:   true,
		Priority:  100,
	}
	e, _ := newTestEngine(t,
		&fakeProjects{bySlug: map[string]*model.Project{"petstore": proj}},
		&fakeAPIs{items: []*model.API{api}},
		&fakeRules{items: []*model.MockRule{rule}},
	)

	w := gginRequest(t, e, "GET", "/mock/petstore/x?q=v", "")
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestEngine_MethodMismatch_Returns404(t *testing.T) {
	proj := &model.Project{ID: "p1", Slug: "petstore"}
	api := &model.API{ID: "api1", ProjectID: "p1", Method: "GET", Path: "/x"}
	rule := &model.MockRule{
		ID:        "r1",
		APIID:     "api1",
		MatchJSON: []byte(`{}`),
		Enabled:   true,
		Priority:  100,
	}
	e, _ := newTestEngine(t,
		&fakeProjects{bySlug: map[string]*model.Project{"petstore": proj}},
		&fakeAPIs{items: []*model.API{api}},
		&fakeRules{items: []*model.MockRule{rule}},
	)

	// Request is POST but rule's API is GET → no candidate → 404.
	w := gginRequest(t, e, "POST", "/mock/petstore/x", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestEngine_ExtractionWritesToVarBag(t *testing.T) {
	proj := &model.Project{ID: "p1", Slug: "petstore"}
	api := &model.API{ID: "api1", ProjectID: "p1", Method: "POST", Path: "/login"}
	rule := &model.MockRule{
		ID:               "r1",
		APIID:            "api1",
		MatchJSON:        []byte(`{}`),
		ResponseStatus:   200,
		ResponseBodyJSON: []byte(`{"token":"tk-1","user_id":99}`),
		ExtractorJSON:    []byte(`[{"path":"token","as":"login_token","from":"response.body"}]`),
		Enabled:          true,
		Priority:         100,
	}
	e, bag := newTestEngine(t,
		&fakeProjects{bySlug: map[string]*model.Project{"petstore": proj}},
		&fakeAPIs{items: []*model.API{api}},
		&fakeRules{items: []*model.MockRule{rule}},
	)

	w := gginRequest(t, e, "POST", "/mock/petstore/login", "")
	require.Equal(t, http.StatusOK, w.Code)

	// Verify the varbag now has login_token via the engine's auto-generated session.
	session := w.Header().Get("X-Mock-Session")
	require.NotEmpty(t, session)
	got, ok, err := bag.Get(context.Background(), "petstore", session, "login_token")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "tk-1", got)
}

func TestEngine_SubstitutionAcrossRequests(t *testing.T) {
	// First request extracts login_token; second request (same session) uses
	// it in a `{{login_token}}` placeholder. Proves the ADR-0007 cross-request
	// story end-to-end.
	proj := &model.Project{ID: "p1", Slug: "petstore"}
	apiLogin := &model.API{ID: "api-login", ProjectID: "p1", Method: "POST", Path: "/login"}
	apiGet := &model.API{ID: "api-get", ProjectID: "p1", Method: "GET", Path: "/me"}
	ruleLogin := &model.MockRule{
		ID:               "r-login",
		APIID:            "api-login",
		MatchJSON:        []byte(`{}`),
		ResponseStatus:   200,
		ResponseBodyJSON: []byte(`{"token":"secret-abc"}`),
		ExtractorJSON:    []byte(`[{"path":"token","as":"login_token","from":"response.body"}]`),
		Enabled:          true,
		Priority:         100,
	}
	ruleGet := &model.MockRule{
		ID:               "r-get",
		APIID:            "api-get",
		MatchJSON:        []byte(`{}`),
		ResponseStatus:   200,
		ResponseBodyJSON: []byte(`{"auth":"Bearer {{login_token}}"}`),
		Enabled:          true,
		Priority:         100,
	}
	e, _ := newTestEngine(t,
		&fakeProjects{bySlug: map[string]*model.Project{"petstore": proj}},
		&fakeAPIs{items: []*model.API{apiLogin, apiGet}},
		&fakeRules{items: []*model.MockRule{ruleLogin, ruleGet}},
	)

	// First request: POST /login, server generates session.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Any("/mock/:projectSlug/*path", e.Serve)

	req1 := httptest.NewRequest("POST", "/mock/petstore/login", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusOK, w1.Code)
	session := w1.Header().Get("X-Mock-Session")
	require.NotEmpty(t, session)

	// Second request: GET /me with the session header.
	req2 := httptest.NewRequest("GET", "/mock/petstore/me", nil)
	req2.Header.Set("X-Mock-Session", session)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &got))
	assert.Equal(t, "Bearer secret-abc", got["auth"], "varbag substitution should have happened")
}

func TestEngine_TieBreakByID_HeaderSurfaced(t *testing.T) {
	proj := &model.Project{ID: "p1", Slug: "petstore"}
	api := &model.API{ID: "api1", ProjectID: "p1", Method: "GET", Path: "/x"}
	high := &model.MockRule{
		ID: "99", APIID: "api1", MatchJSON: []byte(`{"query":{"a":"1"}}`),
		ResponseStatus: 200, ResponseBodyJSON: []byte(`{"who":"99"}`),
		Enabled: true, Priority: 100,
	}
	low := &model.MockRule{
		ID: "01", APIID: "api1", MatchJSON: []byte(`{"query":{"a":"1"}}`),
		ResponseStatus: 200, ResponseBodyJSON: []byte(`{"who":"01"}`),
		Enabled: true, Priority: 100,
	}
	e, _ := newTestEngine(t,
		&fakeProjects{bySlug: map[string]*model.Project{"petstore": proj}},
		&fakeAPIs{items: []*model.API{api}},
		&fakeRules{items: []*model.MockRule{high, low}}, // high first
	)

	w := gginRequest(t, e, "GET", "/mock/petstore/x?a=1", "")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "01", w.Header().Get("X-Mock-Rule-Id"))
	assert.Equal(t, "tie-break-by-id", w.Header().Get("X-Mock-Match-Reason"))
}
