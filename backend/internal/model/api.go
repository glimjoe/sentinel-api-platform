// Package model — API (Phase 2 M1).
//
// An API is one endpoint registered inside a project, normally parsed from an
// OpenAPI document. The (project_id, method, path) tuple is unique so two
// paths with different methods are distinct rows. The JSON columns hold the
// raw schema fragments so the test runner can validate requests/responses
// without re-parsing the source spec.
package model

import "time"

// API maps to the `apis` table. JSON columns use `json.RawMessage` so callers
// can `json.Unmarshal` on demand without a round-trip through map[string]any
// (GORM marshals RawMessage to MySQL JSON natively).
type API struct {
	ID                 string          `gorm:"column:id;type:char(26);primaryKey" json:"id"`
	ProjectID          string          `gorm:"column:project_id;type:char(26);not null;uniqueIndex:uq_apis_project_method_path,priority:1;index:idx_apis_project" json:"project_id"`
	Name               string          `gorm:"column:name;type:varchar(128);not null" json:"name"`
	Method             string          `gorm:"column:method;type:enum('GET','POST','PUT','PATCH','DELETE','HEAD','OPTIONS');not null;uniqueIndex:uq_apis_project_method_path,priority:2" json:"method"`
	Path               string          `gorm:"column:path;type:varchar(512);not null;uniqueIndex:uq_apis_project_method_path,priority:3" json:"path"`
	OperationID        string          `gorm:"column:operation_id;type:varchar(128);not null;default:''" json:"operation_id"`
	TagsJSON           []byte          `gorm:"column:tags_json;type:json" json:"tags_json,omitempty"`
	RequestSchemaJSON  []byte          `gorm:"column:request_schema_json;type:json" json:"request_schema_json,omitempty"`
	ResponseSchemaJSON []byte          `gorm:"column:response_schema_json;type:json" json:"response_schema_json,omitempty"`
	SpecJSON           []byte          `gorm:"column:spec_json;type:json" json:"spec_json,omitempty"`
	Source             string          `gorm:"column:source;type:enum('openapi','manual');not null;default:manual" json:"source"`
	SpecVersion        string          `gorm:"column:spec_version;type:varchar(32);not null;default:''" json:"spec_version"`
	Deprecated         bool            `gorm:"column:deprecated;type:tinyint(1);not null;default:0" json:"deprecated"`
	CreatedAt          time.Time       `gorm:"column:created_at;type:datetime(3);not null" json:"created_at"`
	UpdatedAt          time.Time       `gorm:"column:updated_at;type:datetime(3);not null" json:"updated_at"`
}

// TableName fixes the table to `apis`.
func (API) TableName() string { return "apis" }
