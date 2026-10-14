package task

import (
	"time"

	"sslplatform/internal/entity"
)

const (
	StatusPending = "pending"
	StatusRunning = "running"
	StatusSuccess = "success"
	StatusFailed  = "failed"
	StatusSkipped = "skipped"

	TriggerManual   = "manual"
	TriggerSchedule = "schedule"
)

type Task struct {
	entity.Base
	TenantID      uint       `gorm:"index;not null" json:"tenantId"`
	UserID        uint       `gorm:"index" json:"userId"`
	ETLJobID      uint       `gorm:"index" json:"etlJobId"`
	CertificateID uint       `gorm:"index" json:"certificateId"`
	Trigger       string     `gorm:"size:32;not null" json:"trigger"`
	Status        string     `gorm:"size:32;not null;index" json:"status"`
	Stage         string     `gorm:"size:32" json:"stage"`
	Force         bool       `json:"force"`
	ResultJSON    string     `gorm:"type:text" json:"-"`
	Error         string     `gorm:"type:text" json:"error"`
	StartedAt     *time.Time `json:"startedAt,omitempty"`
	FinishedAt    *time.Time `json:"finishedAt,omitempty"`
}

func (Task) TableName() string { return "tasks" }

type ListRequest struct {
	Page    int        `json:"page"`
	Size    int        `json:"size"`
	Filters ListFilter `json:"filters"`
}

type ListFilter struct {
	Status        string `json:"status"`
	CertificateID uint   `json:"certificateId"`
	ETLJobID      uint   `json:"etlJobId"`
}

type DetailRequest struct {
	ID uint `json:"id"`
}

type View struct {
	ID            uint           `json:"id"`
	ETLJobID      uint           `json:"etlJobId"`
	CertificateID uint           `json:"certificateId"`
	UserID        uint           `json:"userId"`
	Trigger       string         `json:"trigger"`
	Status        string         `json:"status"`
	Stage         string         `json:"stage"`
	Force         bool           `json:"force"`
	Error         string         `json:"error"`
	Result        map[string]any `json:"result"`
	StartedAt     *time.Time     `json:"startedAt,omitempty"`
	FinishedAt    *time.Time     `json:"finishedAt,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
}
