// Package service contains business logic. Services depend on repositories and
// pkg/* utilities, never on *gin.Context or *gorm.DB directly.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/id"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/jwt"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/password"
)

// userStore is the persistence contract AuthService needs from a User repo.
// Defined here (not in repository) so tests can supply a fake without
// dragging *gorm.DB into the service test file.
type userStore interface {
	Create(ctx context.Context, u *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id string) (*model.User, error)
}

// refreshTokenStore is the persistence contract for refresh tokens.
type refreshTokenStore interface {
	GenerateToken(ctx context.Context, userID, ip, userAgent string) (string, error)
	Consume(ctx context.Context, rawToken string) (string, error)
	RevokeAllForUser(ctx context.Context, userID string) error
	LookupUserID(ctx context.Context, rawToken string) (string, error)
}

// AuthService is the entry point for authentication business logic.
type AuthService struct {
	users        userStore
	refreshTokens refreshTokenStore
	accessSecret string
	accessTTL    time.Duration
	bcryptCost   int
}

// NewAuthService wires an AuthService. Caller owns the lifetime.
func NewAuthService(users userStore, refreshTokens refreshTokenStore, accessSecret string, accessTTL time.Duration, bcryptCost int) *AuthService {
	return &AuthService{
		users:         users,
		refreshTokens: refreshTokens,
		accessSecret:  accessSecret,
		accessTTL:     accessTTL,
		bcryptCost:    bcryptCost,
	}
}

// Register creates a new user, hashes the password, persists the row, and
// returns a freshly minted access token and refresh token.
// Returns errs.ErrEmailTaken if the email is already registered.
func (s *AuthService) Register(ctx context.Context, email, plain, displayName string) (*model.User, string, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if err := validateRegisterInput(email, plain); err != nil {
		return nil, "", "", err
	}

	hash, err := password.Hash(plain, s.bcryptCost)
	if err != nil {
		return nil, "", "", fmt.Errorf("hash password: %w", err)
	}

	u := &model.User{
		ID:           id.New(),
		Email:        email,
		PasswordHash: hash,
		DisplayName:  strings.TrimSpace(displayName),
		Role:         model.RoleViewer,
		IsActive:     true,
	}
	if err := s.users.Create(ctx, u); err != nil {
		if isDuplicateEmail(err) {
			return nil, "", "", errs.ErrEmailTaken
		}
		return nil, "", "", fmt.Errorf("create user: %w", err)
	}

	accessToken, err := jwt.Mint(s.accessSecret, u.ID, u.Email, u.Role, s.accessTTL)
	if err != nil {
		return nil, "", "", fmt.Errorf("mint access token: %w", err)
	}
	refreshToken, err := s.refreshTokens.GenerateToken(ctx, u.ID, "", "")
	if err != nil {
		return nil, "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	return u, accessToken, refreshToken, nil
}

// Login verifies email + password and returns (user, access_token, refresh_token, err).
// Returns errs.ErrInvalidCredentials for: missing user, disabled user, OR wrong
// password. Collapsing all three prevents account enumeration.
func (s *AuthService) Login(ctx context.Context, email, plain string) (*model.User, string, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || plain == "" {
		return nil, "", "", errs.ErrInvalidCredentials
	}

	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, "", "", errs.ErrInvalidCredentials
		}
		return nil, "", "", fmt.Errorf("lookup user: %w", err)
	}
	if !u.IsActive {
		return nil, "", "", errs.ErrInvalidCredentials
	}
	if !password.Verify(u.PasswordHash, plain) {
		return nil, "", "", errs.ErrInvalidCredentials
	}

	accessToken, err := jwt.Mint(s.accessSecret, u.ID, u.Email, u.Role, s.accessTTL)
	if err != nil {
		return nil, "", "", fmt.Errorf("mint access token: %w", err)
	}
	refreshToken, err := s.refreshTokens.GenerateToken(ctx, u.ID, "", "")
	if err != nil {
		return nil, "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	return u, accessToken, refreshToken, nil
}

// Me returns the user identified by id (claims.UserID from middleware).
// Returns errs.ErrUserNotFound if the row vanished between auth and lookup,
// or errs.ErrUserInactive if the account was disabled — here the caller is
// already authenticated and is the legitimate owner, so disclosure is fine.
func (s *AuthService) Me(ctx context.Context, userID string) (*model.User, error) {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, errs.ErrUserNotFound
		}
		return nil, fmt.Errorf("lookup user: %w", err)
	}
	if !u.IsActive {
		return nil, errs.ErrUserInactive
	}
	return u, nil
}

// Refresh validates a refresh token, creates new tokens, then revokes the
// old token. Creating first avoids permanent session loss if the DB is
// temporarily unavailable after revocation.
func (s *AuthService) Refresh(ctx context.Context, rawToken string) (*model.User, string, string, error) {
	userID, err := s.refreshTokens.LookupUserID(ctx, rawToken)
	if err != nil {
		return nil, "", "", fmt.Errorf("%w: %v", errs.ErrInvalidCredentials, err)
	}
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, "", "", fmt.Errorf("lookup user for refresh: %w", err)
	}
	if !u.IsActive {
		return nil, "", "", errs.ErrInvalidCredentials
	}
	accessToken, err := jwt.Mint(s.accessSecret, u.ID, u.Email, u.Role, s.accessTTL)
	if err != nil {
		return nil, "", "", fmt.Errorf("mint access token: %w", err)
	}
	refreshToken, err := s.refreshTokens.GenerateToken(ctx, u.ID, "", "")
	if err != nil {
		return nil, "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	// Revoke the old token only after the new pair is successfully created.
	s.refreshTokens.Consume(ctx, rawToken)
	return u, accessToken, refreshToken, nil
}

// Logout revokes all refresh tokens belonging to userID.
func (s *AuthService) Logout(ctx context.Context, userID string) error {
	return s.refreshTokens.RevokeAllForUser(ctx, userID)
}

// validateRegisterInput enforces a minimum bar on email + password length
// AND a bcrypt-compatible upper bound. bcrypt truncates input at 72 bytes
// silently; without this cap, two passwords sharing the first 72 bytes
// hash to the same value and Verify accepts either. Reject early so callers
// get a clear 400 instead of silent truncation.
func validateRegisterInput(email, plain string) error {
	if !strings.Contains(email, "@") {
		return fmt.Errorf("%w: email must contain '@'", errs.ErrBadRequest)
	}
	if len(plain) < 8 {
		return fmt.Errorf("%w: password must be >= 8 chars", errs.ErrBadRequest)
	}
	if len(plain) > 72 {
		return fmt.Errorf("%w: password must be <= 72 chars (bcrypt limit)", errs.ErrBadRequest)
	}
	return nil
}

// isDuplicateEmail inspects a GORM error to detect the MySQL 1062 duplicate-key
// case. We avoid a hard dependency on the driver's exported type by checking
// via errors.As; falls back to false for unknown drivers.
func isDuplicateEmail(err error) bool {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number == 1062 // ER_DUP_ENTRY
	}
	return false
}