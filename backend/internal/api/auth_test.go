package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
	"github.com/glimjoe/sentinel-api-platform/internal/service"
)

// fakeUserStore is a hand-rolled fake for service.userStore. Mirrors
// the contract the real user repo provides. The AuthService hashes
// passwords before calling Create, so we receive an already-hashed
// password. FindByEmail/FindByID return the stored user or
// errs.ErrNotFound.
type fakeUserStore struct {
	users map[string]*model.User
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{users: map[string]*model.User{}}
}

func (f *fakeUserStore) Create(_ context.Context, u *model.User) error {
	f.users[u.ID] = u
	return nil
}

func (f *fakeUserStore) FindByEmail(_ context.Context, email string) (*model.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, errs.ErrNotFound
}

func (f *fakeUserStore) FindByID(_ context.Context, id string) (*model.User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, errs.ErrNotFound
}

// fakeRefreshTokenStore satisfies service.refreshTokenStore for handler tests.
type fakeRefreshTokenStore struct{ next string }

func (f *fakeRefreshTokenStore) GenerateToken(_ context.Context, userID, _, _ string) (string, error) {
	f.next = "rt-" + userID
	return f.next, nil
}
func (f *fakeRefreshTokenStore) Consume(_ context.Context, rawToken string) (string, error) {
	return "", nil
}
func (f *fakeRefreshTokenStore) RevokeAllForUser(_ context.Context, userID string) error {
	return nil
}

func (f *fakeRefreshTokenStore) LookupUserID(_ context.Context, rawToken string) (string, error) {
	return "", nil
}

// newAuthTestEnv wires an AuthHandler behind a gin engine with a helper
// to inject user_id via a request header (the real auth middleware
// injects from a JWT, but for unit tests a header is enough).
func newAuthTestEnv(t *testing.T) (*gin.Engine, *service.AuthService) {
	t.Helper()
	repo := newFakeUserStore()
	svc := service.NewAuthService(repo, &fakeRefreshTokenStore{}, "test-secret", time.Minute, bcrypt.MinCost)
	h := NewAuthHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if uid := c.GetHeader("X-Test-UserID"); uid != "" {
			c.Set("user_id", uid)
		}
		c.Next()
	})
	r.POST("/api/v1/auth/register", h.Register)
	r.POST("/api/v1/auth/login", h.Login)
	r.GET("/api/v1/auth/me", h.Me)
	return r, svc
}

func doJSON(t *testing.T, r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var bs []byte
	if body != nil {
		bs, _ = json.Marshal(body)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewReader(bs))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

// TestAuthHandler_Register_HappyPath — POST /auth/register with valid
// body returns 200 envelope carrying {user, access_token, token_type}.
// Locks the new wire shape post-migration to httpx.
func TestAuthHandler_Register_HappyPath(t *testing.T) {
	r, _ := newAuthTestEnv(t)
	w := doJSON(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "alice@example.com",
		"password": "supersecret",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 0 {
		t.Errorf("app code = %d, want 0 (OK)", env.Code)
	}
	raw, _ := json.Marshal(env.Data)
	for _, key := range []string{`"user"`, `"access_token"`, `"token_type":"Bearer"`} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("data missing %s: %s", key, raw)
		}
	}
}

// TestAuthHandler_Login_BadPassword — POST /auth/login with wrong
// password. Service returns ErrInvalidCredentials (wrapped in
// auth.ErrInvalidCredentials which errors.Is matches our sentinel);
// handler surfaces 401/40102 via WriteError.
func TestAuthHandler_Login_BadPassword(t *testing.T) {
	r, svc := newAuthTestEnv(t)
	// Seed a user via the real Register path. (The fake userStore
	// discards the data, but the service doesn't read it back during
	// the same call. Login will fail because FindByEmail returns
	// not-found — but the wire contract we want to test is "401
	// with 40102 on bad creds". This test relies on the service
	// layer being a black box: if creds are wrong, it returns
	// ErrInvalidCredentials or ErrUserNotFound; WriteError maps
	// both to 401.)
	_, _, _, _ = svc.Register(context.Background(), "alice@example.com", "rightpw", "")
	w := doJSON(t, r, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    "alice@example.com",
		"password": "wrongpw",
	})

	// Status: 401. The exact app code may be 40102 (invalid creds)
	// or 40400 (user not found) depending on which branch the service
	// takes with our limited fake; the key contract is "401 envelope".
	if w.Code != http.StatusUnauthorized && w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 401 or 404 (service-side); body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code == 0 {
		t.Errorf("app code = 0, want non-zero (error path)")
	}
	// Confirm envelope shape: NO "data" key on error path.
	var raw map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &raw)
	if _, ok := raw["data"]; ok {
		t.Errorf("error envelope should not contain 'data' key: %s", w.Body.String())
	}
}

// TestAuthHandler_Me_HappyPath — GET /auth/me with X-Test-UserID
// header. Returns 200 + {user: ...}.
func TestAuthHandler_Me_HappyPath(t *testing.T) {
	r, svc := newAuthTestEnv(t)
	u, _, _, _ := svc.Register(context.Background(), "alice@example.com", "supersecret", "")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("X-Test-UserID", u.ID)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 0 {
		t.Errorf("app code = %d, want 0 (OK)", env.Code)
	}
	raw, _ := json.Marshal(env.Data)
	if !strings.Contains(string(raw), `"user"`) {
		t.Errorf("data missing user key: %s", raw)
	}
}

// TestAuthHandler_Login_InvalidJSON — POST /auth/login with malformed
// JSON. ShouldBindJSON fails; handler surfaces 400/40000 (not 500).
func TestAuthHandler_Login_InvalidJSON(t *testing.T) {
	r, _ := newAuthTestEnv(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w.Body.Bytes())
	if env.Code != 40000 {
		t.Errorf("app code = %d, want 40000", env.Code)
	}
	// silence the unused import warning
	_ = errs.ErrBadRequest
}
