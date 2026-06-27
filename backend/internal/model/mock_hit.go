// Package model — MockHit (Phase 2 M1).
//
// A MockHit is a record of one matched mock request. The write path is wired
// in M2 (recorder.go). The table is created now so the engine can insert
// later without a migration. For M1 the engine skips this write — see
// engine.go's TODO comment.
package model

import "time"

// MockHit maps to the `mock_hits` table. Service layer is responsible for
// truncating `request_body_json` to 8KB before insert (the column type is
// JSON and GORM has no automatic truncation for it).
type MockHit struct {
	ID                string    `gorm:"column:id;type:char(26);primaryKey" json:"id"`
	MockRuleID        string    `gorm:"column:mock_rule_id;type:char(26);not null;index:idx_mock_hits_rule_created,priority:1" json:"mock_rule_id"`
	RequestMethod     string    `gorm:"column:request_method;type:varchar(16);not null" json:"request_method"`
	RequestPath       string    `gorm:"column:request_path;type:varchar(512);not null" json:"request_path"`
	RequestHeadersJSON []byte   `gorm:"column:request_headers_json;type:json" json:"request_headers_json,omitempty"`
	RequestBodyJSON   []byte    `gorm:"column:request_body_json;type:json" json:"request_body_json,omitempty"`
	ResponseStatus    int       `gorm:"column:response_status;type:int;not null" json:"response_status"`
	ResponseBodyJSON  []byte    `gorm:"column:response_body_json;type:json" json:"response_body_json,omitempty"`
	DurationMs        int       `gorm:"column:duration_ms;type:int;not null" json:"duration_ms"`
	CreatedAt         time.Time `gorm:"column:created_at;type:datetime(3);not null;index:idx_mock_hits_rule_created,priority:2,sort:desc" json:"created_at"`
}

// TableName fixes the table to `mock_hits`.
func (MockHit) TableName() string { return "mock_hits" }
