package model

import "time"

type TestResult struct {
	ID                     string    `gorm:"column:id;type:char(26);primaryKey" json:"id"`
	RunID                  string    `gorm:"column:run_id;type:char(26);not null" json:"run_id"`
	CaseID                 string    `gorm:"column:case_id;type:char(26);not null" json:"case_id"`
	Status                 string    `gorm:"column:status;type:enum('pass','fail','error','skip');not null" json:"status"`
	ActualStatus           *int      `gorm:"column:actual_status;type:int" json:"actual_status,omitempty"`
	ActualHeadersJSON      []byte    `gorm:"column:actual_headers_json;type:json" json:"actual_headers_json,omitempty"`
	ActualBodyJSON         []byte    `gorm:"column:actual_body_json;type:json" json:"actual_body_json,omitempty"`
	AssertionFailuresJSON  []byte    `gorm:"column:assertion_failures_json;type:json" json:"assertion_failures_json,omitempty"`
	DurationMs             int       `gorm:"column:duration_ms;type:int;not null;default:0" json:"duration_ms"`
	ErrorMsg               string    `gorm:"column:error_msg;type:varchar(2048);not null;default:''" json:"error_msg"`
	AIAttributionJSON      []byte    `gorm:"column:ai_attribution_json;type:json" json:"ai_attribution_json,omitempty"`
	Attempt                int       `gorm:"column:attempt;type:int;not null;default:1" json:"attempt"`
	CreatedAt              time.Time `gorm:"column:created_at;type:datetime(3);not null" json:"created_at"`
}

func (TestResult) TableName() string { return "test_results" }
