package scheduler

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

func (r *Repository) Create(ctx context.Context, item *Schedule) error {
	return apperror.FromDB(r.db.WithContext(ctx).Create(item).Error, "调度不存在")
}

func (r *Repository) Save(ctx context.Context, item *Schedule) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *Repository) Get(ctx context.Context, tenantID, id uint) (*Schedule, error) {
	var item Schedule
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&item).Error
	if err != nil {
		return nil, apperror.FromDB(err, "调度不存在")
	}
	return &item, nil
}

func (r *Repository) GetByID(ctx context.Context, id uint) (*Schedule, error) {
	var item Schedule
	err := r.db.WithContext(ctx).First(&item, id).Error
	if err != nil {
		return nil, apperror.FromDB(err, "调度不存在")
	}
	return &item, nil
}

func (r *Repository) List(ctx context.Context, tenantID uint, page, size int, filter ListFilter) ([]Schedule, int64, error) {
	q := r.db.WithContext(ctx).Model(&Schedule{}).Where("tenant_id = ?", tenantID)
	if filter.ETLJobID > 0 {
		q = q.Where("etl_job_id = ?", filter.ETLJobID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Schedule
	err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

func (r *Repository) ListEnabled(ctx context.Context) ([]Schedule, error) {
	var rows []Schedule
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&rows).Error
	return rows, err
}

func (r *Repository) Delete(ctx context.Context, tenantID, id uint) error {
	res := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&Schedule{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperror.New(404, "调度不存在")
	}
	return nil
}

func (r *Repository) MarkRun(ctx context.Context, id uint, last time.Time, next *time.Time, status string) error {
	return r.db.WithContext(ctx).Model(&Schedule{}).Where("id = ?", id).Updates(map[string]any{
		"last_run_at": last,
		"next_run_at": next,
		"last_status": status,
	}).Error
}
