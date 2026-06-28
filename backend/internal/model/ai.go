// Package model — AI usage tracking.
package model

import "time"

// AiUsage records one LLM call for cost tracking and budget enforcement.
type AiUsage struct {
	ID               string    `gorm:"primaryKey;type:CHAR(26)" json:"id"`
	Model            string    `gorm:"type:VARCHAR(64);not null"  json:"model"`
	Function         string    `gorm:"type:ENUM('attribution','completion','prioritization');not null" json:"function"`
	PromptTokens     int       `gorm:"not null;default:0"          json:"prompt_tokens"`
	CompletionTokens int       `gorm:"not null;default:0"          json:"completion_tokens"`
	CostUSD          float64   `gorm:"type:DECIMAL(12,8);not null;default:0" json:"cost_usd"`
	ProjectID        *string   `gorm:"type:CHAR(26);index"         json:"project_id,omitempty"`
	CreatedAt        time.Time `gorm:"not null"                   json:"created_at"`
}

// TableName overrides the default table name.
func (AiUsage) TableName() string { return "ai_usage" }
