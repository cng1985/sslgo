package scheduler

import (
	"time"

	"sslplatform/internal/entity"
)

type Schedule struct {
	entity.Base
	TenantID   uint       `gorm:"not null;index" json:"tenantId"`
	Name       string     `gorm:"size:128;not null" json:"name"`
	ETLJobID   uint       `gorm:"not null;index" json:"etlJobId"`
	Cron       string     `gorm:"size:64;not null" json:"cron"`
	Enabled    bool       `json:"enabled"`
	LastRunAt  *time.Time `json:"lastRunAt,omitempty"`
	NextRunAt  *time.Time `json:"nextRunAt,omitempty"`
	LastStatus string     `gorm:"size:32" json:"lastStatus"`
}

func (Schedule) TableName() string { return "schedules" }

type CreateRequest struct {
	Name     string `json:"name"`
	ETLJobID uint   `json:"etlJobId"`
	Cron     string `json:"cron"`
	Enabled  bool   `json:"enabled"`
}

type UpdateRequest struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Cron    string `json:"cron"`
	Enabled *bool  `json:"enabled"`
}

type ListRequest struct {
	Page    int        `json:"page"`
	Size    int        `json:"size"`
	Filters ListFilter `json:"filters"`
}

type ListFilter struct {
	ETLJobID uint `json:"etlJobId"`
}

type IDRequest struct {
	ID uint `json:"id"`
}

type View struct {
	ID         uint       `json:"id"`
	Name       string     `json:"name"`
	ETLJobID   uint       `json:"etlJobId"`
	Cron       string     `json:"cron"`
	Enabled    bool       `json:"enabled"`
	LastRunAt  *time.Time `json:"lastRunAt,omitempty"`
	NextRunAt  *time.Time `json:"nextRunAt,omitempty"`
	LastStatus string     `json:"lastStatus"`
	CreatedAt  time.Time  `json:"createdAt"`
}
