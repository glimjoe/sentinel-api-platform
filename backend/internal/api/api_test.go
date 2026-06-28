// Package api — API handler tests (RED → GREEN, M2-F.C).
//
// 7 endpoints under test:
//   POST   /api/v1/projects/:pid/apis              CreateAPI
//   GET    /api/v1/projects/:pid/apis              ListAPIs
//   GET    /api/v1/projects/:pid/apis/:apiId       GetAPI
//   PATCH  /api/v1/projects/:pid/apis/:apiId       UpdateAPI
//   DELETE /api/v1/projects/:pid/apis/:apiId       DeleteAPI
//   POST   /api/v1/projects/:pid/apis/:apiId/tags   AddTag
//   DELETE /api/v1/projects/:pid/apis/:apiId/tags/:tag RemoveTag
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

// ---------------------------------------------------------------------------
// fakeAPIService — hand-rolled stub implementing the apiService interface
// from api.go. Each test sets the closure it needs.
// ---------------------------------------------------------------------------

type fakeAPIService struct {
	CreateFn       func(ctx context.Context, callerID, projectID, name, method, path, operationID string) (*model.API, error)
	ListByProjectFn func(ctx context.Context, projectID string) ([]*model.API, error)
	FindByIDFn     func(ctx context.Context, id string) (*model.API, error)
	UpdateFn       func(ctx context.Context, callerID, apiID string, fields map[string]any) (*model.API, error)
	DeleteFn       func(ctx context.Context, callerID, apiID string) error
}

func (f *fakeAPIService) Create(ctx context.Context, callerID, projectID, name, method, path, operationID string) (*model.API, error) {
	return f.CreateFn(ctx, callerID, projectID, name, method, path, operationID)
}

func (f *fakeAPIService) ListByProject(ctx context.Context, projectID string) ([]*model.API, error) {
	return f.ListByProjectFn(ctx, projectID)
}

func (f *fakeAPIService) FindByID(ctx context.Context, id string) (*model.API, error) {
	return f.FindByIDFn(ctx, id)
}

func (f *fakeAPIService) Update(ctx context.Context, callerID, apiID string, fields map[string]any) (*model.API, error) {
	return f.UpdateFn(ctx, callerID, apiID, fields)
}

func (f *fakeAPIService) Delete(ctx context.Context, callerID, apiID string) error {
	return f.DeleteFn(ctx, callerID, apiID)
}

// ---------------------------------------------------------------------------
// test harness
// ---------------------------------------------------------------------------

func newAPITestEngine(t *testing.T, svc *fakeAPIService, callerID string) *gin.Engine {
	t.Helper()
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if callerID != "" {
			c.Set("user_id", callerID)
		}
		c.Next()
	})
	h := &APIHandler{svc: svc}
	r.POST("/api/v1/projects/:pid/apis", h.CreateAPI)
	r.GET("/api/v1/projects/:pid/apis", h.ListAPIs)
	r.GET("/api/v1/projects/:pid/apis/:apiId", h.GetAPI)
	r.PATCH("/api/v1/projects/:pid/apis/:apiId", h.UpdateAPI)
	r.DELETE("/api/v1/projects/:pid/apis/:apiId", h.DeleteAPI)
	r.POST("/api/v1/projects/:pid/apis/:apiId/tags", h.AddTag)
	r.DELETE("/api/v1/projects/:pid/apis/:apiId/tags/:tag", h.RemoveTag)
	return r
}

// decodeEnvelope is reused from project_test.go (same package).

// ---------------------------------------------------------------------------
// CreateAPI
// ---------------------------------------------------------------------------

func TestCreateAPI_HappyPath(t *testing.T) {
	svc := &fakeAPIService{
		CreateFn: func(_ context.Context, callerID, projectID, name, method, path, operationID string) (*model.API, error) {
			if callerID != "u-engineer" {
				t.Errorf("callerID = %q, want u-engineer", callerID)
			}
			if projectID != "01HP" {
				t.Errorf("projectID = %q, want 01HP", projectID)
			}
			if name != "FindPets" {
				t.Errorf("name = %q, want FindPets", name)
			}
			return &model.API{ID: "01HA", ProjectID: projectID, Name: name, Method: method, Path: path}, nil
		},
	}
	r := newAPITestEngine(t, svc, "u-engineer")
	body := map[string]string{
		"name":   "FindPets",
		"method": "GET",
		"path":   "/pet/findByStatus",
	}
	bs, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/01HP/apis", bytes.NewReader(bs))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestCreateAPI_Forbidden(t *testing.T) {
	svc := &fakeAPIService{
		CreateFn: func(_ context.Context, _, _, _, _, _, _ string) (*model.API, error) {
			return nil, errs.ErrForbidden
		},
	}
	r := newAPITestEngine(t, svc, "u-viewer")
	body := map[string]string{"name": "X", "method": "GET", "path": "/x"}
	bs, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/01HP/apis", bytes.NewReader(bs))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

func TestCreateAPI_MissingUserID(t *testing.T) {
	svc := &fakeAPIService{
		CreateFn: func(_ context.Context, _, _, _, _, _, _ string) (*model.API, error) {
			t.Error("Create must not be called when user_id is missing")
			return nil, nil
		},
	}
	r := newAPITestEngine(t, svc, "") // no caller
	body := map[string]string{"name": "X", "method": "GET", "path": "/x"}
	bs, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/01HP/apis", bytes.NewReader(bs))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// ListAPIs
// ---------------------------------------------------------------------------

func TestListAPIs_HappyPath(t *testing.T) {
	svc := &fakeAPIService{
		ListByProjectFn: func(_ context.Context, projectID string) ([]*model.API, error) {
			if projectID != "01HP" {
				t.Errorf("projectID = %q, want 01HP", projectID)
			}
			return []*model.API{
				{ID: "01HA-1", ProjectID: projectID, Name: "FindPets", Method: "GET", Path: "/pets"},
				{ID: "01HA-2", ProjectID: projectID, Name: "AddPet", Method: "POST", Path: "/pets"},
			}, nil
		},
	}
	r := newAPITestEngine(t, svc, "u-viewer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/01HP/apis", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestListAPIs_EmptyList(t *testing.T) {
	svc := &fakeAPIService{
		ListByProjectFn: func(_ context.Context, _ string) ([]*model.API, error) {
			return []*model.API{}, nil
		},
	}
	r := newAPITestEngine(t, svc, "u-viewer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/01HP/apis", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestListAPIs_MissingUserID(t *testing.T) {
	svc := &fakeAPIService{
		ListByProjectFn: func(_ context.Context, _ string) ([]*model.API, error) {
			t.Error("ListByProject must not be called when user_id is missing")
			return nil, nil
		},
	}
	r := newAPITestEngine(t, svc, "")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/01HP/apis", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GetAPI
// ---------------------------------------------------------------------------

func TestGetAPI_HappyPath(t *testing.T) {
	svc := &fakeAPIService{
		FindByIDFn: func(_ context.Context, id string) (*model.API, error) {
			if id != "01HA" {
				t.Errorf("id = %q, want 01HA", id)
			}
			return &model.API{ID: id, Name: "FindPets", Method: "GET", Path: "/pets"}, nil
		},
	}
	r := newAPITestEngine(t, svc, "u-viewer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/01HP/apis/01HA", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestGetAPI_NotFound(t *testing.T) {
	svc := &fakeAPIService{
		FindByIDFn: func(_ context.Context, _ string) (*model.API, error) {
			return nil, errs.ErrNotFound
		},
	}
	r := newAPITestEngine(t, svc, "u-viewer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/01HP/apis/missing", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestGetAPI_MissingUserID(t *testing.T) {
	svc := &fakeAPIService{
		FindByIDFn: func(_ context.Context, _ string) (*model.API, error) {
			t.Error("FindByID must not be called when user_id is missing")
			return nil, nil
		},
	}
	r := newAPITestEngine(t, svc, "")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/01HP/apis/01HA", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// UpdateAPI
// ---------------------------------------------------------------------------

func TestUpdateAPI_HappyPath(t *testing.T) {
	svc := &fakeAPIService{
		UpdateFn: func(_ context.Context, callerID, apiID string, fields map[string]any) (*model.API, error) {
			if callerID != "u-engineer" {
				t.Errorf("callerID = %q, want u-engineer", callerID)
			}
			if apiID != "01HA" {
				t.Errorf("apiID = %q, want 01HA", apiID)
			}
			if v, ok := fields["name"].(string); !ok || v != "Renamed" {
				t.Errorf("fields[name] = %v, want Renamed", fields["name"])
			}
			return &model.API{ID: apiID, Name: "Renamed"}, nil
		},
	}
	r := newAPITestEngine(t, svc, "u-engineer")
	body := map[string]any{"name": "Renamed"}
	bs, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/01HP/apis/01HA", bytes.NewReader(bs))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateAPI_Forbidden(t *testing.T) {
	svc := &fakeAPIService{
		UpdateFn: func(_ context.Context, _, _ string, _ map[string]any) (*model.API, error) {
			return nil, errs.ErrForbidden
		},
	}
	r := newAPITestEngine(t, svc, "u-viewer")
	body := map[string]any{"name": "X"}
	bs, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/01HP/apis/01HA", bytes.NewReader(bs))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateAPI_MissingUserID(t *testing.T) {
	svc := &fakeAPIService{
		UpdateFn: func(_ context.Context, _, _ string, _ map[string]any) (*model.API, error) {
			t.Error("Update must not be called when user_id is missing")
			return nil, nil
		},
	}
	r := newAPITestEngine(t, svc, "")
	body := map[string]any{"name": "X"}
	bs, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/01HP/apis/01HA", bytes.NewReader(bs))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// DeleteAPI
// ---------------------------------------------------------------------------

func TestDeleteAPI_HappyPath(t *testing.T) {
	called := false
	svc := &fakeAPIService{
		DeleteFn: func(_ context.Context, callerID, apiID string) error {
			called = true
			if callerID != "u-admin" {
				t.Errorf("callerID = %q, want u-admin", callerID)
			}
			if apiID != "01HA" {
				t.Errorf("apiID = %q, want 01HA", apiID)
			}
			return nil
		},
	}
	r := newAPITestEngine(t, svc, "u-admin")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/01HP/apis/01HA", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if !called {
		t.Error("Delete not called")
	}
}

func TestDeleteAPI_Forbidden(t *testing.T) {
	svc := &fakeAPIService{
		DeleteFn: func(_ context.Context, _, _ string) error {
			return errs.ErrForbidden
		},
	}
	r := newAPITestEngine(t, svc, "u-viewer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/01HP/apis/01HA", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteAPI_MissingUserID(t *testing.T) {
	svc := &fakeAPIService{
		DeleteFn: func(_ context.Context, _, _ string) error {
			t.Error("Delete must not be called when user_id is missing")
			return nil
		},
	}
	r := newAPITestEngine(t, svc, "")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/01HP/apis/01HA", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// AddTag
// ---------------------------------------------------------------------------

func TestAddTag_HappyPath(t *testing.T) {
	svc := &fakeAPIService{
		FindByIDFn: func(_ context.Context, id string) (*model.API, error) {
			return &model.API{ID: id, TagsJSON: []byte(`["v1"]`)}, nil
		},
		UpdateFn: func(_ context.Context, callerID, apiID string, fields map[string]any) (*model.API, error) {
			raw, ok := fields["tags_json"].([]byte)
			if !ok {
				t.Fatal("fields[tags_json] is not []byte")
			}
			var tags []string
			json.Unmarshal(raw, &tags)
			if len(tags) != 2 || tags[1] != "v2" {
				t.Errorf("tags = %v, want [v1 v2]", tags)
			}
			return &model.API{ID: apiID, TagsJSON: raw}, nil
		},
	}
	r := newAPITestEngine(t, svc, "u-engineer")
	body := map[string]string{"tag": "v2"}
	bs, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/01HP/apis/01HA/tags", bytes.NewReader(bs))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestAddTag_MissingBody(t *testing.T) {
	svc := &fakeAPIService{}
	r := newAPITestEngine(t, svc, "u-engineer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/01HP/apis/01HA/tags", nil)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

func TestAddTag_APINotFound(t *testing.T) {
	svc := &fakeAPIService{
		FindByIDFn: func(_ context.Context, _ string) (*model.API, error) {
			return nil, errs.ErrNotFound
		},
	}
	r := newAPITestEngine(t, svc, "u-engineer")
	body := map[string]string{"tag": "v2"}
	bs, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/01HP/apis/01HA/tags", bytes.NewReader(bs))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// RemoveTag
// ---------------------------------------------------------------------------

func TestRemoveTag_HappyPath(t *testing.T) {
	svc := &fakeAPIService{
		FindByIDFn: func(_ context.Context, id string) (*model.API, error) {
			return &model.API{ID: id, TagsJSON: []byte(`["v1","v2"]`)}, nil
		},
		UpdateFn: func(_ context.Context, callerID, apiID string, fields map[string]any) (*model.API, error) {
			raw := fields["tags_json"].([]byte)
			var tags []string
			json.Unmarshal(raw, &tags)
			if len(tags) != 1 || tags[0] != "v1" {
				t.Errorf("tags = %v, want [v1]", tags)
			}
			return &model.API{ID: apiID, TagsJSON: raw}, nil
		},
	}
	r := newAPITestEngine(t, svc, "u-engineer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/01HP/apis/01HA/tags/v2", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestRemoveTag_APINotFound(t *testing.T) {
	svc := &fakeAPIService{
		FindByIDFn: func(_ context.Context, _ string) (*model.API, error) {
			return nil, errs.ErrNotFound
		},
	}
	r := newAPITestEngine(t, svc, "u-engineer")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/01HP/apis/01HA/tags/ghost", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}
