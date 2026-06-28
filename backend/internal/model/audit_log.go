package model

import "time"

// AuditLog maps to the `audit_logs` table.
type AuditLog struct {
	ID           string    `gorm:"column:id;type:char(26);primaryKey" json:"id"`
	UserID       string    `gorm:"column:user_id;type:char(26)" json:"user_id,omitempty"`
	Action       string    `gorm:"column:action;type:varchar(64);not null" json:"action"`
	ResourceType string    `gorm:"column:resource_type;type:varchar(64);not null" json:"resource_type"`
	ResourceID   string    `gorm:"column:resource_id;type:varchar(64);not null;default:''" json:"resource_id"`
	ProjectID    string    `gorm:"column:project_id;type:char(26)" json:"project_id,omitempty"`
	PayloadJSON  []byte    `gorm:"column:payload_json;type:json" json:"payload_json,omitempty"`
	IP           string    `gorm:"column:ip;type:varchar(45);not null;default:''" json:"ip"`
	UserAgent    string    `gorm:"column:user_agent;type:varchar(512);not null;default:''" json:"user_agent"`
	CreatedAt    time.Time `gorm:"column:created_at;type:datetime(3);not null" json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }
