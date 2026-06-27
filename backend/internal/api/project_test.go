package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/httpx"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// fakeProjectService is a hand-rolled fake for the project service. We
// avoid a mocking library so the test file has no extra dependency. As
// more handler methods land, add the matching method here.
type fakeProjectService struct {
	CreateFunc  func(ctx context.Context, ownerID, name, slug, description string) (*model.Project, error)
	ListFunc    func(ctx context.Context, ownerID string) ([]*model.Project, error)
	FindByIDFunc func(ctx context.Context, id string) (*model.Project, error)
}

func (f *fakeProjectService) Create(ctx context.Context, ownerID, name, slug, description string) (*model.Project, error) {
	return f.CreateFunc(ctx, ownerID, name, slug, description)
}

func (f *fakeProjectService) List(ctx context.Context, ownerID string) ([]*model.Project, error) {
	return f.ListFunc(ctx, ownerID)
}

func (f *fakeProjectService) FindByID(ctx context.Context, id string) (*model.Project, error) {
	return f.FindByIDFunc(ctx, id)
}

// newTestEngine wires a single POST /projects endpoint behind ProjectHandler
// with the supplied fake service. Caller ID is injected by the helper
// (auth middleware is exercised separately in middleware tests).
func newTestEngine(t *testing.T, svc *fakeProjectService, callerID string) *gin.Engine {
	t.Helper()
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if callerID != "" {
			c.Set("user_id", callerID)
		}
		c.Next()
	})
	h := &ProjectHandler{svc: svc}
	r.POST("/api/v1/projects", h.CreateProject)
	r.GET("/api/v1/projects", h.ListProjects)
	r.GET("/api/v1/projects/:pid", h.GetProject)
	return r
}

// doCreateProject is a small driver: builds a JSON body, sends it, returns
// the recorded response. Body fields left blank are omitted from JSON.
func doCreateProject(t *testing.T, r *gin.Engine, name, slug, description string) *httptest.ResponseRecorder {
	t.Helper()
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if slug != "" {
		body["slug"] = slug
	}
	if description != "" {
		body["description"] = description
	}
	bs, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewReader(bs))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

// TestCreateProject_HappyPath — POST with valid body returns 200 envelope
// carrying the created project. The owner is the caller (injected user_id).
func TestCreateProject_HappyPath(t *testing.T) {
	svc := &fakeProjectService{
		CreateFunc: func(_ context.Context, ownerID, name, slug, desc string) (*model.Project, error) {
			if ownerID != "u-owner" {
				t.Errorf("ownerID = %q, want u-owner", ownerID)
			}
			if name != "Petstore" || slug != "petstore" {
				t.Errorf("name=%q slug=%q, want Petstore/petstore", name, slug)
			}
			return &model.Project{
				ID: "proj-1", Name: name, Slug: slug, OwnerID: ownerID, Description: desc,
			}, nil
		},
	}
	r := newTestEngine(t, svc, "u-owner")
	w := doCreateProject(t, r, "Petstore", "petstore", "demo project")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 0 {
		t.Errorf("app code = %d, want 0 (OK)", env.Code)
	}
	// Data carries the project — verify by re-parse.
	raw, _ := json.Marshal(env.Data)
	if !strings.Contains(string(raw), `"id":"proj-1"`) {
		t.Errorf("data missing project id: %s", raw)
	}
	if !strings.Contains(string(raw), `"owner_id":"u-owner"`) {
		t.Errorf("data missing owner_id: %s", raw)
	}
}

// TestCreateProject_ConflictSlug — service returns ErrConflict, handler
// must surface 409 via WriteError.
func TestCreateProject_ConflictSlug(t *testing.T) {
	svc := &fakeProjectService{
		CreateFunc: func(_ context.Context, _, _, _, _ string) (*model.Project, error) {
			return nil, errs.ErrConflict
		},
	}
	r := newTestEngine(t, svc, "u-owner")
	w := doCreateProject(t, r, "Petstore", "petstore", "")

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 40900 {
		t.Errorf("app code = %d, want 40900", env.Code)
	}
}

// TestCreateProject_BadRequest_EmptyName — service returns ErrBadRequest
// (e.g. empty name after trim), handler must surface 400.
func TestCreateProject_BadRequest_EmptyName(t *testing.T) {
	svc := &fakeProjectService{
		CreateFunc: func(_ context.Context, _, _, _, _ string) (*model.Project, error) {
			return nil, fmt.Errorf("name and slug are required: %w", errs.ErrBadRequest)
		},
	}
	r := newTestEngine(t, svc, "u-owner")
	w := doCreateProject(t, r, "Petstore", "petstore", "")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 40000 {
		t.Errorf("app code = %d, want 40000", env.Code)
	}
}

// TestCreateProject_BindError — malformed JSON body fails c.ShouldBindJSON,
// handler must surface 400 (not 500). Guards against the easy mistake of
// letting the bind error fall through to the default branch.
func TestCreateProject_BindError(t *testing.T) {
	svc := &fakeProjectService{
		CreateFunc: func(_ context.Context, _, _, _, _ string) (*model.Project, error) {
			t.Fatal("service.Create must not be called on bind failure")
			return nil, nil
		},
	}
	r := newTestEngine(t, svc, "u-owner")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 40000 {
		t.Errorf("app code = %d, want 40000", env.Code)
	}
}

// TestCreateProject_MissingUserID — caller forgot to mount the auth
// middleware. Must fail loud (500), not silently create a project with
// empty owner_id.
func TestCreateProject_MissingUserID(t *testing.T) {
	svc := &fakeProjectService{
		CreateFunc: func(_ context.Context, _, _, _, _ string) (*model.Project, error) {
			t.Fatal("service.Create must not be called when user_id is missing")
			return nil, nil
		},
	}
	r := newTestEngine(t, svc, "") // empty callerID → no Set
	w := doCreateProject(t, r, "Petstore", "petstore", "")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 50001 {
		t.Errorf("app code = %d, want 50001", env.Code)
	}
	if !strings.Contains(strings.ToLower(env.Message), "user_id") {
		t.Errorf("message = %q, want it to mention user_id", env.Message)
	}
}

// TestCreateProject_UnknownError — generic error → 500/50000 with generic
// message (the full wrap is in the access log, not the response body).
func TestCreateProject_UnknownError(t *testing.T) {
	svc := &fakeProjectService{
		CreateFunc: func(_ context.Context, _, _, _, _ string) (*model.Project, error) {
			return nil, errors.New("db down")
		},
	}
	r := newTestEngine(t, svc, "u-owner")
	w := doCreateProject(t, r, "Petstore", "petstore", "")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 50000 {
		t.Errorf("app code = %d, want 50000", env.Code)
	}
	if env.Message != "internal server error" {
		t.Errorf("message = %q, want generic", env.Message)
	}
}

// doListProjects is a small driver: GETs /api/v1/projects and returns
// the recorded response.
func doListProjects(t *testing.T, r *gin.Engine) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	r.ServeHTTP(w, req)
	return w
}

// TestListProjects_HappyPath — service returns 2 projects, handler
// surfaces them as 200 + array in the envelope data field. The ownerID
// argument must equal the caller's user_id from context.
func TestListProjects_HappyPath(t *testing.T) {
	svc := &fakeProjectService{
		ListFunc: func(_ context.Context, ownerID string) ([]*model.Project, error) {
			if ownerID != "u-owner" {
				t.Errorf("ownerID = %q, want u-owner", ownerID)
			}
			return []*model.Project{
				{ID: "p1", Name: "Petstore", Slug: "petstore", OwnerID: ownerID},
				{ID: "p2", Name: "Orders", Slug: "orders", OwnerID: ownerID},
			}, nil
		},
	}
	r := newTestEngine(t, svc, "u-owner")
	w := doListProjects(t, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 0 {
		t.Errorf("app code = %d, want 0 (OK)", env.Code)
	}
	// Data must be an array of 2 projects.
	raw, _ := json.Marshal(env.Data)
	if !strings.Contains(string(raw), `"id":"p1"`) {
		t.Errorf("data missing p1: %s", raw)
	}
	if !strings.Contains(string(raw), `"id":"p2"`) {
		t.Errorf("data missing p2: %s", raw)
	}
}

// TestListProjects_EmptyList — service returns an empty slice, handler
// must surface 200 + an empty array (NOT null, NOT a missing key — the
// frontend iterates without a nil check).
func TestListProjects_EmptyList(t *testing.T) {
	svc := &fakeProjectService{
		ListFunc: func(_ context.Context, _ string) ([]*model.Project, error) {
			return []*model.Project{}, nil
		},
	}
	r := newTestEngine(t, svc, "u-owner")
	w := doListProjects(t, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	raw, _ := json.Marshal(env.Data)
	// Must be an empty array, not "null".
	if string(raw) != "[]" {
		t.Errorf("data = %s, want [] (empty array, not null)", raw)
	}
}

// TestListProjects_ServiceError — unknown error → 500/50000.
func TestListProjects_ServiceError(t *testing.T) {
	svc := &fakeProjectService{
		ListFunc: func(_ context.Context, _ string) ([]*model.Project, error) {
			return nil, errors.New("db down")
		},
	}
	r := newTestEngine(t, svc, "u-owner")
	w := doListProjects(t, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 50000 {
		t.Errorf("app code = %d, want 50000", env.Code)
	}
}

// TestListProjects_MissingUserID — fail loud (500/50001) if the auth
// middleware was forgotten, same convention as CreateProject.
func TestListProjects_MissingUserID(t *testing.T) {
	svc := &fakeProjectService{
		ListFunc: func(_ context.Context, _ string) ([]*model.Project, error) {
			t.Fatal("service.List must not be called when user_id is missing")
			return nil, nil
		},
	}
	r := newTestEngine(t, svc, "")
	w := doListProjects(t, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 50001 {
		t.Errorf("app code = %d, want 50001", env.Code)
	}
}

// doGetProject is a small driver: GETs /api/v1/projects/:pid and returns
// the recorded response.
func doGetProject(t *testing.T, r *gin.Engine, pid string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+pid, nil)
	r.ServeHTTP(w, req)
	return w
}

// TestGetProject_HappyPath — service returns the project; handler
// surfaces 200 + project envelope. RBAC enforcement is the middleware's
// job (covered in middleware/rbac_test.go) — the handler assumes the
// caller is already authorized.
func TestGetProject_HappyPath(t *testing.T) {
	svc := &fakeProjectService{
		FindByIDFunc: func(_ context.Context, id string) (*model.Project, error) {
			if id != "proj-1" {
				t.Errorf("FindByID called with id = %q, want proj-1", id)
			}
			return &model.Project{ID: id, Name: "Petstore", Slug: "petstore", OwnerID: "u-owner"}, nil
		},
	}
	r := newTestEngine(t, svc, "u-owner")
	w := doGetProject(t, r, "proj-1")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 0 {
		t.Errorf("app code = %d, want 0 (OK)", env.Code)
	}
	raw, _ := json.Marshal(env.Data)
	if !strings.Contains(string(raw), `"id":"proj-1"`) {
		t.Errorf("data missing project id: %s", raw)
	}
}

// TestGetProject_NotFound — service returns ErrNotFound; handler
// surfaces 404 via WriteError. (The rbac middleware may also surface
// 404 if the project doesn't exist; the handler does not preempt that,
// but the wire contract is the same: 404/40400.)
func TestGetProject_NotFound(t *testing.T) {
	svc := &fakeProjectService{
		FindByIDFunc: func(_ context.Context, _ string) (*model.Project, error) {
			return nil, errs.ErrNotFound
		},
	}
	r := newTestEngine(t, svc, "u-owner")
	w := doGetProject(t, r, "proj-missing")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 40400 {
		t.Errorf("app code = %d, want 40400", env.Code)
	}
}

// TestGetProject_MissingUserID — fail loud (500/50001) if the auth
// middleware was forgotten.
func TestGetProject_MissingUserID(t *testing.T) {
	svc := &fakeProjectService{
		FindByIDFunc: func(_ context.Context, _ string) (*model.Project, error) {
			t.Fatal("service.FindByID must not be called when user_id is missing")
			return nil, nil
		},
	}
	r := newTestEngine(t, svc, "")
	w := doGetProject(t, r, "proj-1")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 50001 {
		t.Errorf("app code = %d, want 50001", env.Code)
	}
}
func decodeEnvelope(t *testing.T, body []byte) httpx.Envelope {
	t.Helper()
	var env httpx.Envelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode envelope: %v (body=%s)", err, body)
	}
	return env
}
