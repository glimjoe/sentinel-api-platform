package model

import "time"

type TestRun struct {
	ID             string     `gorm:"column:id;type:char(26);primaryKey" json:"id"`
	ProjectID      string     `gorm:"column:project_id;type:char(26);not null" json:"project_id"`
	Name           string     `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Status         string     `gorm:"column:status;type:enum('queued','running','success','failed','cancelled','partial');not null;default:queued" json:"status"`
	Mode           string     `gorm:"column:mode;type:enum('sequential','parallel');not null;default:sequential" json:"mode"`
	Concurrency    int        `gorm:"column:concurrency;type:int;not null;default:1" json:"concurrency"`
	TargetBaseURL  string     `gorm:"column:target_base_url;type:varchar(512);not null;default:''" json:"target_base_url"`
	CaseFilterJSON []byte     `gorm:"column:case_filter_json;type:json" json:"case_filter_json,omitempty"`
	Total          int        `gorm:"column:total;type:int;not null;default:0" json:"total"`
	Passed         int        `gorm:"column:passed;type:int;not null;default:0" json:"passed"`
	Failed         int        `gorm:"column:failed;type:int;not null;default:0" json:"failed"`
	Errored        int        `gorm:"column:errored;type:int;not null;default:0" json:"errored"`
	Skipped        int        `gorm:"column:skipped;type:int;not null;default:0" json:"skipped"`
	StartedAt      *time.Time `gorm:"column:started_at;type:datetime(3)" json:"started_at,omitempty"`
	FinishedAt     *time.Time `gorm:"column:finished_at;type:datetime(3)" json:"finished_at,omitempty"`
	TriggeredBy    *string    `gorm:"column:triggered_by;type:char(26)" json:"triggered_by,omitempty"`
	TriggerType    string     `gorm:"column:trigger_type;type:enum('manual','schedule','api');not null;default:manual" json:"trigger_type"`
	CreatedAt      time.Time  `gorm:"column:created_at;type:datetime(3);not null" json:"created_at"`
}

func (TestRun) TableName() string { return "test_runs" }
