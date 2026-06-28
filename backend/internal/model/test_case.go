package model

import "time"

type TestCase struct {
	ID                   string    `gorm:"column:id;type:char(26);primaryKey" json:"id"`
	ProjectID            string    `gorm:"column:project_id;type:char(26);not null" json:"project_id"`
	APIID                *string   `gorm:"column:api_id;type:char(26)" json:"api_id,omitempty"`
	Name                 string    `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Description          string    `gorm:"column:description;type:varchar(1024);not null;default:''" json:"description"`
	Method               string    `gorm:"column:method;type:varchar(16);not null" json:"method"`
	Path                 string    `gorm:"column:path;type:varchar(512);not null" json:"path"`
	HeadersJSON          []byte    `gorm:"column:headers_json;type:json" json:"headers_json,omitempty"`
	QueryJSON            []byte    `gorm:"column:query_json;type:json" json:"query_json,omitempty"`
	BodyJSON             []byte    `gorm:"column:body_json;type:json" json:"body_json,omitempty"`
	ExpectedStatus       int       `gorm:"column:expected_status;type:int;not null;default:200" json:"expected_status"`
	ExpectedBodyJSON     []byte    `gorm:"column:expected_body_json;type:json" json:"expected_body_json,omitempty"`
	ExpectedBodyMatch    string    `gorm:"column:expected_body_match;type:enum('exact','contains','jsonpath','schema','regex','none');not null;default:none" json:"expected_body_match"`
	ExpectedBodyPattern  string    `gorm:"column:expected_body_pattern;type:text" json:"expected_body_pattern,omitempty"`
	AssertionsJSON       []byte    `gorm:"column:assertions_json;type:json" json:"assertions_json,omitempty"`
	TagsJSON             []byte    `gorm:"column:tags_json;type:json" json:"tags_json,omitempty"`
	Priority             string    `gorm:"column:priority;type:enum('p0','p1','p2','p3');not null;default:p2" json:"priority"`
	AIGenerated          bool      `gorm:"column:ai_generated;type:tinyint(1);not null;default:0" json:"ai_generated"`
	CreatedBy            *string   `gorm:"column:created_by;type:char(26)" json:"created_by,omitempty"`
	CreatedAt            time.Time `gorm:"column:created_at;type:datetime(3);not null" json:"created_at"`
	UpdatedAt            time.Time `gorm:"column:updated_at;type:datetime(3);not null" json:"updated_at"`
}

func (TestCase) TableName() string { return "test_cases" }
