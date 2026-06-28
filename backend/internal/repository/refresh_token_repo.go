// Package repository — RefreshTokenRepo.
package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/glimjoe/sentinel-api-platform/internal/model"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/id"
)

// RefreshTokenRepo persists refresh_tokens rows.
type RefreshTokenRepo struct {
	db *gorm.DB
}

// NewRefreshTokenRepo constructs a RefreshTokenRepo bound to db.
func NewRefreshTokenRepo(db *gorm.DB) *RefreshTokenRepo {
	return &RefreshTokenRepo{db: db}
}

// GenerateToken creates a new refresh token row and returns the raw token
// string. The caller is responsible for returning the raw string to the
// client; only the SHA-256 hash is stored in the database.
func (r *RefreshTokenRepo) GenerateToken(ctx context.Context, userID, ip, userAgent string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	rawStr := hex.EncodeToString(raw)
	hash := sha256.Sum256([]byte(rawStr))
	hashStr := hex.EncodeToString(hash[:])

	rt := &model.RefreshToken{
		ID:        id.New(),
		UserID:    userID,
		TokenHash: hashStr,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		IP:        ip,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
	}
	if err := r.db.WithContext(ctx).Create(rt).Error; err != nil {
		return "", fmt.Errorf("save refresh token: %w", err)
	}
	return rawStr, nil
}

// Consume finds a non-revoked, non-expired token by hash and atomically
// marks it revoked. Uses UPDATE WHERE to eliminate the SELECT→Save race.
func (r *RefreshTokenRepo) Consume(ctx context.Context, rawToken string) (string, error) {
	hash := sha256.Sum256([]byte(rawToken))
	hashStr := hex.EncodeToString(hash[:])

	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", hashStr, now).
		Update("revoked_at", now)
	if result.Error != nil {
		return "", fmt.Errorf("consume refresh token: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return "", fmt.Errorf("refresh_token_repo: invalid or expired token")
	}

	// Fetch the now-revoked row to get user_id.
	var rt model.RefreshToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", hashStr).First(&rt).Error; err != nil {
		return "", fmt.Errorf("fetch user_id after revoke: %w", err)
	}
	return rt.UserID, nil
}

// RevokeAllForUser revokes all active refresh tokens for a user (logout).
func (r *RefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, now).
		Update("revoked_at", now).Error
}

// LookupUserID returns the user_id for a valid (non-revoked, non-expired)
// refresh token without revoking it.
func (r *RefreshTokenRepo) LookupUserID(ctx context.Context, rawToken string) (string, error) {
	hash := sha256.Sum256([]byte(rawToken))
	hashStr := hex.EncodeToString(hash[:])

	var rt model.RefreshToken
	if err := r.db.WithContext(ctx).
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", hashStr, time.Now()).
		First(&rt).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("refresh_token_repo: invalid or expired token")
		}
		return "", fmt.Errorf("lookup refresh token: %w", err)
	}
	return rt.UserID, nil
}
