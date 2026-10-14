package etl

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

func (r *Repository) CreateCertificate(ctx context.Context, item *Certificate) error {
	return apperror.FromDB(r.db.WithContext(ctx).Create(item).Error, "证书不存在")
}

func (r *Repository) SaveCertificate(ctx context.Context, item *Certificate) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *Repository) GetCertificate(ctx context.Context, tenantID, id uint) (*Certificate, error) {
	var item Certificate
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&item).Error
	if err != nil {
		return nil, apperror.FromDB(err, "证书不存在")
	}
	return &item, nil
}

func (r *Repository) ListCertificates(ctx context.Context, tenantID uint, page, size int, filter CertificateListFilter) ([]Certificate, int64, error) {
	q := r.db.WithContext(ctx).Model(&Certificate{}).Where("tenant_id = ?", tenantID)
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		q = q.Where("name LIKE ? OR domains_json LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Certificate
	err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

func (r *Repository) DeleteCertificate(ctx context.Context, tenantID, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var jobs []ETLJob
		if err := tx.Where("tenant_id = ? AND certificate_id = ?", tenantID, id).Find(&jobs).Error; err != nil {
			return err
		}
		for _, job := range jobs {
			if err := tx.Exec("DELETE FROM schedules WHERE tenant_id = ? AND etl_job_id = ?", tenantID, job.ID).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("tenant_id = ? AND certificate_id = ?", tenantID, id).Delete(&ETLJob{}).Error; err != nil {
			return err
		}
		res := tx.Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&Certificate{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return apperror.New(404, "证书不存在")
		}
		return nil
	})
}

func (r *Repository) CreateJob(ctx context.Context, item *ETLJob) error {
	return apperror.FromDB(r.db.WithContext(ctx).Create(item).Error, "流水线不存在")
}

func (r *Repository) SaveJob(ctx context.Context, item *ETLJob) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *Repository) GetJob(ctx context.Context, tenantID, id uint) (*ETLJob, error) {
	var item ETLJob
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&item).Error
	if err != nil {
		return nil, apperror.FromDB(err, "流水线不存在")
	}
	return &item, nil
}

func (r *Repository) JobByCertificate(ctx context.Context, tenantID, certificateID uint) (*ETLJob, error) {
	var item ETLJob
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND certificate_id = ?", tenantID, certificateID).Order("id asc").First(&item).Error
	if err != nil {
		return nil, apperror.FromDB(err, "流水线不存在")
	}
	return &item, nil
}

func (r *Repository) ListJobs(ctx context.Context, tenantID uint, page, size int, filter JobListFilter) ([]ETLJob, int64, error) {
	q := r.db.WithContext(ctx).Model(&ETLJob{}).Where("tenant_id = ?", tenantID)
	if filter.CertificateID > 0 {
		q = q.Where("certificate_id = ?", filter.CertificateID)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []ETLJob
	err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

func (r *Repository) DeleteJob(ctx context.Context, tenantID, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM schedules WHERE tenant_id = ? AND etl_job_id = ?", tenantID, id).Error; err != nil {
			return err
		}
		res := tx.Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&ETLJob{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return apperror.New(404, "流水线不存在")
		}
		return nil
	})
}

func (r *Repository) CountCertificates(ctx context.Context, tenantID uint, status string) (int64, error) {
	q := r.db.WithContext(ctx).Model(&Certificate{}).Where("tenant_id = ?", tenantID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *Repository) Expiring(ctx context.Context, tenantID uint, before time.Time) ([]Certificate, error) {
	var rows []Certificate
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND not_after IS NOT NULL AND not_after <= ?", tenantID, before).
		Order("not_after asc").Limit(8).Find(&rows).Error
	return rows, err
}

func (r *Repository) CountExpiring(ctx context.Context, tenantID uint, before time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Certificate{}).
		Where("tenant_id = ? AND not_after IS NOT NULL AND not_after <= ?", tenantID, before).
		Count(&n).Error
	return n, err
}
