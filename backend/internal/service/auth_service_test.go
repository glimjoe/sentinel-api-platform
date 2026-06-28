package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

// fakeUserStore is an in-memory implementation of userStore used only by tests.
type fakeUserStore struct {
	byID    map[string]*model.User
	byEmail map[string]*model.User
	// createErr overrides Create for negative-path tests (e.g. dup-key simulation).
	createErr error
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{
		byID:    make(map[string]*model.User),
		byEmail: make(map[string]*model.User),
	}
}

func (f *fakeUserStore) Create(_ context.Context, u *model.User) error {
	if f.createErr != nil {
		return f.createErr
	}
	if _, exists := f.byEmail[u.Email]; exists {
		// Mimic the MySQL 1062 ER_DUP_ENTRY error so isDuplicateEmail matches.
		return &mysql.MySQLError{Number: 1062, Message: "Duplicate entry '" + u.Email + "' for key 'uq_users_email'"}
	}
	f.byID[u.ID] = u
	f.byEmail[u.Email] = u
	return nil
}

func (f *fakeUserStore) FindByEmail(_ context.Context, email string) (*model.User, error) {
	if u, ok := f.byEmail[email]; ok {
		return u, nil
	}
	return nil, fmt.Errorf("fake_user_store: %w", errs.ErrNotFound)
}

func (f *fakeUserStore) FindByID(_ context.Context, id string) (*model.User, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, fmt.Errorf("fake_user_store: %w", errs.ErrNotFound)
}

// fakeRefreshTokenStore satisfies refreshTokenStore for tests.
type fakeRefreshTokenStore struct {
	tokens map[string]string // raw → userID
}

func newFakeRefreshTokenStore() *fakeRefreshTokenStore {
	return &fakeRefreshTokenStore{tokens: make(map[string]string)}
}

func (f *fakeRefreshTokenStore) GenerateToken(_ context.Context, userID, _, _ string) (string, error) {
	raw := fmt.Sprintf("rt-%s-%d", userID, len(f.tokens)+1)
	f.tokens[raw] = userID
	return raw, nil
}

func (f *fakeRefreshTokenStore) Consume(_ context.Context, rawToken string) (string, error) {
	uid, ok := f.tokens[rawToken]
	if !ok {
		return "", fmt.Errorf("invalid token")
	}
	delete(f.tokens, rawToken)
	return uid, nil
}

func (f *fakeRefreshTokenStore) RevokeAllForUser(_ context.Context, userID string) error {
	for raw, uid := range f.tokens {
		if uid == userID {
			delete(f.tokens, raw)
		}
	}
	return nil
}

func (f *fakeRefreshTokenStore) LookupUserID(_ context.Context, rawToken string) (string, error) {
	uid, ok := f.tokens[rawToken]
	if !ok {
		return "", fmt.Errorf("invalid token")
	}
	return uid, nil
}

// newTestService wires AuthService against the fake with sane defaults.
func newTestService(users userStore) *AuthService {
	return NewAuthService(users, newFakeRefreshTokenStore(), "test-secret-must-be-long-enough-for-hs256",
		15*time.Minute, 4) // cost=4 keeps tests fast
}

func TestAuthService_Register_Success(t *testing.T) {
	users := newFakeUserStore()
	svc := newTestService(users)

	u, token, _, err := svc.Register(context.Background(),
		"alice@example.com", "hunter2hunter", "Alice")

	require.NoError(t, err)
	require.NotNil(t, u)
	require.NotEmpty(t, token)
	assert.Equal(t, "alice@example.com", u.Email)
	assert.Equal(t, "Alice", u.DisplayName)
	assert.Equal(t, model.RoleViewer, u.Role)
	assert.True(t, u.IsActive)
	assert.NotEmpty(t, u.ID)
	assert.NotEmpty(t, u.PasswordHash, "password must be hashed, not stored plain")
	assert.NotEqual(t, "hunter2hunter", u.PasswordHash)
	assert.Len(t, strings.Split(token, "."), 3, "JWT must have 3 dot-separated segments")
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	users := newFakeUserStore()
	svc := newTestService(users)

	// First registration succeeds.
	_, _, _, err := svc.Register(context.Background(),
		"alice@example.com", "hunter2hunter", "Alice")
	require.NoError(t, err)

	// Second registration with same email triggers fakeUserStore's MySQL 1062 path.
	_, _, _, err = svc.Register(context.Background(),
		"alice@example.com", "differentpass123", "Alice2")
	assert.ErrorIs(t, err, errs.ErrEmailTaken)
}

func TestAuthService_Register_InvalidInput(t *testing.T) {
	svc := newTestService(newFakeUserStore())
	tests := []struct {
		name, email, plain string
	}{
		{"no-at-sign", "alice", "hunter2hunter"},
		{"short-password", "alice@example.com", "short"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := svc.Register(context.Background(), tc.email, tc.plain, "")
			require.Error(t, err)
			assert.True(t, errors.Is(err, errs.ErrBadRequest),
				"expected ErrBadRequest, got %v", err)
		})
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	users := newFakeUserStore()
	svc := newTestService(users)

	// Seed via Register so the bcrypt hash is produced the same way prod does.
	_, _, _, err := svc.Register(context.Background(),
		"alice@example.com", "hunter2hunter", "")
	require.NoError(t, err)

	u, token, _, err := svc.Login(context.Background(),
		"ALICE@example.com", "hunter2hunter") // also verifies email lowercase normalisation

	require.NoError(t, err)
	require.NotNil(t, u)
	require.NotEmpty(t, token)
	assert.Equal(t, "alice@example.com", u.Email)
}

func TestAuthService_Login_RejectsBadCredentials(t *testing.T) {
	users := newFakeUserStore()
	svc := newTestService(users)
	_, _, _, _ = svc.Register(context.Background(),
		"alice@example.com", "hunter2hunter", "")

	tests := []struct {
		name, email, plain string
	}{
		{"wrong-password", "alice@example.com", "WRONG"},
		{"unknown-email", "bob@example.com", "hunter2hunter"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := svc.Login(context.Background(), tc.email, tc.plain)
			require.Error(t, err)
			assert.ErrorIs(t, err, errs.ErrInvalidCredentials)
		})
	}
}

func TestAuthService_Login_InactiveUser(t *testing.T) {
	users := newFakeUserStore()
	svc := newTestService(users)

	u, _, _, err := svc.Register(context.Background(),
		"alice@example.com", "hunter2hunter", "")
	require.NoError(t, err)
	users.byID[u.ID].IsActive = false

	// Inactive accounts return ErrInvalidCredentials (not ErrUserInactive) to
	// prevent account enumeration — Login's caller can't distinguish this from
	// unknown-email / wrong-password. ErrUserInactive is reserved for Me.
	_, _, _, err = svc.Login(context.Background(),
		"alice@example.com", "hunter2hunter")
	require.Error(t, err)
	assert.ErrorIs(t, err, errs.ErrInvalidCredentials)
}

func TestAuthService_Me(t *testing.T) {
	users := newFakeUserStore()
	svc := newTestService(users)
	seeded, _, _, err := svc.Register(context.Background(),
		"alice@example.com", "hunter2hunter", "Alice")
	require.NoError(t, err)

	u, err := svc.Me(context.Background(), seeded.ID)
	require.NoError(t, err)
	assert.Equal(t, seeded.ID, u.ID)
}

func TestAuthService_Me_NotFound(t *testing.T) {
	svc := newTestService(newFakeUserStore())
	_, err := svc.Me(context.Background(), "01HXXXXXXXXXXXXXXXXXXXXXXXX")
	require.Error(t, err)
	assert.ErrorIs(t, err, errs.ErrUserNotFound)
}