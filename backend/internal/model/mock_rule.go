// Package model — MockRule (Phase 2 M1).
//
// A MockRule is one configurable response attached to an API. The matcher
// (internal/mock/matcher.go) scores rules against an incoming request and
// picks the highest scorer. The `extractor_json` column drives variable
// extraction per ADR-0007.
//
// The `match_json` and `response_*` columns are `json.RawMessage` so callers
// can pass the bytes directly to the matcher/extractor without an extra
// marshal step.
package model

import "time"

// MockRule maps to the `mock_rules` table.
type MockRule struct {
	ID                  string    `gorm:"column:id;type:char(26);primaryKey" json:"id"`
	APIID               string    `gorm:"column:api_id;type:char(26);not null;index:idx_mock_rules_api_enabled_priority,priority:1" json:"api_id"`
	Name                string    `gorm:"column:name;type:varchar(128);not null" json:"name"`
	MatchJSON           []byte    `gorm:"column:match_json;type:json;not null" json:"match_json"`
	ResponseStatus      int       `gorm:"column:response_status;type:int;not null;default:200" json:"response_status"`
	ResponseHeadersJSON []byte    `gorm:"column:response_headers_json;type:json" json:"response_headers_json,omitempty"`
	ResponseBodyJSON    []byte    `gorm:"column:response_body_json;type:json" json:"response_body_json,omitempty"`
	ExtractorJSON       []byte    `gorm:"column:extractor_json;type:json" json:"extractor_json,omitempty"`
	Priority            int       `gorm:"column:priority;type:int;not null;default:100;index:idx_mock_rules_api_enabled_priority,priority:3" json:"priority"`
	DelayMs             int       `gorm:"column:delay_ms;type:int;not null;default:0" json:"delay_ms"`
	Enabled             bool      `gorm:"column:enabled;type:tinyint(1);not null;default:1;index:idx_mock_rules_api_enabled_priority,priority:2" json:"enabled"`
	HitCount            int64     `gorm:"column:hit_count;type:bigint;not null;default:0" json:"hit_count"`
	CreatedAt           time.Time `gorm:"column:created_at;type:datetime(3);not null" json:"created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at;type:datetime(3);not null" json:"updated_at"`
}

// TableName fixes the table to `mock_rules`.
func (MockRule) TableName() string { return "mock_rules" }
