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
	"gorm.io/gorm"

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

// AuthService is the entry point for authentication business logic.
type AuthService struct {
	users        userStore
	accessSecret string
	accessTTL    time.Duration
	bcryptCost   int
}

// NewAuthService wires an AuthService. Caller owns the lifetime.
func NewAuthService(users userStore, accessSecret string, accessTTL time.Duration, bcryptCost int) *AuthService {
	return &AuthService{
		users:        users,
		accessSecret: accessSecret,
		accessTTL:    accessTTL,
		bcryptCost:   bcryptCost,
	}
}

// Register creates a new user, hashes the password, persists the row, and
// returns a freshly minted access token. Returns errs.ErrEmailTaken if the
// email is already registered.
func (s *AuthService) Register(ctx context.Context, email, plain, displayName string) (*model.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if err := validateRegisterInput(email, plain); err != nil {
		return nil, "", err
	}

	hash, err := password.Hash(plain, s.bcryptCost)
	if err != nil {
		return nil, "", fmt.Errorf("hash password: %w", err)
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
			return nil, "", errs.ErrEmailTaken
		}
		return nil, "", fmt.Errorf("create user: %w", err)
	}

	token, err := jwt.Mint(s.accessSecret, u.ID, u.Email, u.Role, s.accessTTL)
	if err != nil {
		return nil, "", fmt.Errorf("mint access token: %w", err)
	}
	return u, token, nil
}

// Login verifies email + password and returns (user, access_token).
// Returns errs.ErrInvalidCredentials when the user is missing OR the password
// does not match, to avoid leaking which emails are registered.
func (s *AuthService) Login(ctx context.Context, email, plain string) (*model.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || plain == "" {
		return nil, "", errs.ErrInvalidCredentials
	}

	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", errs.ErrInvalidCredentials
		}
		return nil, "", fmt.Errorf("lookup user: %w", err)
	}
	if !u.IsActive {
		return nil, "", errs.ErrUserInactive
	}
	if !password.Verify(u.PasswordHash, plain) {
		return nil, "", errs.ErrInvalidCredentials
	}

	token, err := jwt.Mint(s.accessSecret, u.ID, u.Email, u.Role, s.accessTTL)
	if err != nil {
		return nil, "", fmt.Errorf("mint access token: %w", err)
	}
	return u, token, nil
}

// Me returns the user identified by id (claims.UserID from middleware).
// Returns errs.ErrUserNotFound if the row vanished between auth and lookup,
// or errs.ErrUserInactive if the account was disabled.
func (s *AuthService) Me(ctx context.Context, userID string) (*model.User, error) {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrUserNotFound
		}
		return nil, fmt.Errorf("lookup user: %w", err)
	}
	if !u.IsActive {
		return nil, errs.ErrUserInactive
	}
	return u, nil
}

// validateRegisterInput enforces a minimum bar on email + password length.
// Tighter rules (regex, complexity) can land in a follow-up.
func validateRegisterInput(email, plain string) error {
	if !strings.Contains(email, "@") {
		return fmt.Errorf("%w: email must contain '@'", errs.ErrBadRequest)
	}
	if len(plain) < 8 {
		return fmt.Errorf("%w: password must be >= 8 chars", errs.ErrBadRequest)
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