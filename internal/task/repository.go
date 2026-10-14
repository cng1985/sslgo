package task

import (
	"context"
	"time"

	"gorm.io/gorm"

	"sslplatform/internal/common/apperror"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, item *Task) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *Repository) Save(ctx context.Context, item *Task) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *Repository) Get(ctx context.Context, tenantID, id uint) (*Task, error) {
	var item Task
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&item).Error
	if err != nil {
		return nil, apperror.FromDB(err, "任务不存在")
	}
	return &item, nil
}

func (r *Repository) GetByID(ctx context.Context, id uint) (*Task, error) {
	var item Task
	err := r.db.WithContext(ctx).First(&item, id).Error
	if err != nil {
		return nil, apperror.FromDB(err, "任务不存在")
	}
	return &item, nil
}

func (r *Repository) List(ctx context.Context, tenantID uint, page, size int, filter ListFilter) ([]Task, int64, error) {
	q := r.db.WithContext(ctx).Model(&Task{}).Where("tenant_id = ?", tenantID)
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.CertificateID > 0 {
		q = q.Where("certificate_id = ?", filter.CertificateID)
	}
	if filter.ETLJobID > 0 {
		q = q.Where("etl_job_id = ?", filter.ETLJobID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Task
	err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

func (r *Repository) HasRunning(ctx context.Context, tenantID, certificateID uint) (bool, error) {
	var n int64
	cutoff := time.Now().Add(-15 * time.Minute)
	err := r.db.WithContext(ctx).Model(&Task{}).
		Where("tenant_id = ? AND certificate_id = ? AND status IN ? AND created_at >= ?", tenantID, certificateID, []string{StatusPending, StatusRunning}, cutoff).
		Count(&n).Error
	return n > 0, err
}

func (r *Repository) MarkRunning(ctx context.Context, id uint, stage string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&Task{}).Where("id = ?", id).Updates(map[string]any{
		"status":     StatusRunning,
		"stage":      stage,
		"started_at": now,
	}).Error
}

func (r *Repository) CountSince(ctx context.Context, tenantID uint, status string, since time.Time) (int64, error) {
	q := r.db.WithContext(ctx).Model(&Task{}).Where("tenant_id = ?", tenantID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if !since.IsZero() {
		q = q.Where("created_at >= ?", since)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}
