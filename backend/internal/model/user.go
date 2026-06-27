// Package model holds GORM-backed domain entities for the Sentinel API.
//
// ID format: 26-char ULID strings stored as CHAR(26) (see migrations/0001_init.sql).
// Timestamps are DATETIME(3) UTC; GORM auto-fills CreatedAt/UpdatedAt.
package model

import (
	"time"
)

// Role enumerates the three account levels recognised by Sentinel.
// Persisted as ENUM in MySQL; keep values aligned with migrations/0001_init.sql.
const (
	RoleAdmin    = "admin"
	RoleEngineer = "engineer"
	RoleViewer   = "viewer"
)

// User maps to the `users` table. Field tags are aligned with migrations/0001_init.sql.
type User struct {
	ID           string    `gorm:"column:id;type:char(26);primaryKey" json:"id"`
	Email        string    `gorm:"column:email;type:varchar(255);uniqueIndex:uq_users_email;not null" json:"email"`
	PasswordHash string    `gorm:"column:password_hash;type:varchar(255);not null" json:"-"`
	DisplayName  string    `gorm:"column:display_name;type:varchar(64);not null;default:''" json:"display_name"`
	Role         string    `gorm:"column:role;type:enum('admin','engineer','viewer');not null;default:viewer;index:idx_users_role" json:"role"`
	IsActive     bool      `gorm:"column:is_active;type:tinyint(1);not null;default:1" json:"is_active"`
	CreatedAt    time.Time `gorm:"column:created_at;type:datetime(3);not null" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:datetime(3);not null" json:"updated_at"`
}

// TableName fixes the table to `users` (GORM default would pluralise to `users` anyway,
// but explicit is better than implicit for migrations/replication tooling).
func (User) TableName() string { return "users" }