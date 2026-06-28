// Package model — RefreshToken.
package model

import "time"

// RefreshToken maps to the refresh_tokens table (0001_init).
type RefreshToken struct {
	ID        string     `gorm:"primaryKey;type:CHAR(26)"     json:"id"`
	UserID    string     `gorm:"type:CHAR(26);not null;index"  json:"user_id"`
	TokenHash string     `gorm:"type:VARCHAR(255);uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time  `gorm:"not null"                      json:"expires_at"`
	RevokedAt *time.Time `gorm:""                              json:"revoked_at,omitempty"`
	IP        string     `gorm:"type:VARCHAR(45)"              json:"ip"`
	UserAgent string     `gorm:"type:VARCHAR(255)"             json:"user_agent"`
	CreatedAt time.Time  `gorm:"not null"                      json:"created_at"`
}

// TableName overrides the default table name.
func (RefreshToken) TableName() string { return "refresh_tokens" }
