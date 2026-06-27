// Package repository holds GORM-backed persistence methods. One file per
// aggregate root. Service code never touches *gorm.DB directly.
package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

// UserRepo persists users against MySQL via GORM.
type UserRepo struct {
	db *gorm.DB
}

// NewUserRepo constructs a UserRepo bound to db.
func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

// Create inserts a new user row. Returns a wrapped gorm.ErrDuplicatedKey if
// the email is already taken; service layer maps that to errs.ErrEmailTaken.
func (r *UserRepo) Create(ctx context.Context, u *model.User) error {
	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// FindByEmail returns the user with the given email, or errs.ErrNotFound
// wrapped if no row matches. Service layer uses errors.Is(err, errs.ErrNotFound)
// and decides whether to surface as ErrUserNotFound or collapse into
// ErrInvalidCredentials (login flow).
func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user_repo: %w", errs.ErrNotFound)
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return &u, nil
}

// FindByID returns the user with the given id, or errs.ErrNotFound wrapped.
func (r *UserRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
	var u model.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user_repo: %w", errs.ErrNotFound)
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return &u, nil
}