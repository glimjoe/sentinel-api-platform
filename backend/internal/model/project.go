// Package model — Project + ProjectMember (Phase 2 M1).
//
// Projects are the top-level tenant of a user's mock/test surface. Each project
// has many APIs, many mock_rules, and many project_members. The `slug` is the
// human-friendly identifier used in the public URL `GET /mock/:slug/*path`.
//
// Membership: a user can belong to many projects, with a per-project role
// (admin / engineer / viewer). The role is distinct from the global `User.role`
// in `model/user.go` — same enum values, different scope.
package model

import "time"

// ProjectRole values are kept aligned with User.Role so the RBAC middleware
// can reuse the same set of role checks at the project level.
const (
	ProjectRoleAdmin    = "admin"
	ProjectRoleEngineer = "engineer"
	ProjectRoleViewer   = "viewer"
)

// Project maps to the `projects` table. owner_id is a hard FK to users; if
// the owner is deleted, deletion is RESTRICT (so projects outlive their
// original creator until a successor is assigned — see plan §5).
type Project struct {
	ID             string    `gorm:"column:id;type:char(26);primaryKey" json:"id"`
	Name           string    `gorm:"column:name;type:varchar(128);not null" json:"name"`
	Slug           string    `gorm:"column:slug;type:varchar(64);uniqueIndex:uq_projects_slug;not null" json:"slug"`
	OwnerID        string    `gorm:"column:owner_id;type:char(26);not null;index:idx_projects_owner" json:"owner_id"`
	Description    string    `gorm:"column:description;type:varchar(512);not null;default:''" json:"description"`
	DefaultBaseURL string    `gorm:"column:default_base_url;type:varchar(512);not null;default:''" json:"default_base_url"`
	CreatedAt      time.Time `gorm:"column:created_at;type:datetime(3);not null" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;type:datetime(3);not null" json:"updated_at"`
}

// TableName fixes the table to `projects`.
func (Project) TableName() string { return "projects" }

// ProjectMember maps to the `project_members` join table. Composite PK on
// (project_id, user_id) — GORM needs a primary key for some operations, so we
// declare one even though the row identity is the pair.
type ProjectMember struct {
	ProjectID string    `gorm:"column:project_id;type:char(26);primaryKey" json:"project_id"`
	UserID    string    `gorm:"column:user_id;type:char(26);primaryKey" json:"user_id"`
	Role      string    `gorm:"column:role;type:enum('admin','engineer','viewer');not null;default:viewer" json:"role"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime(3);not null" json:"created_at"`
}

// TableName fixes the table to `project_members`.
func (ProjectMember) TableName() string { return "project_members" }
