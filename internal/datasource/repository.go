package datasource

import (
	"context"

	"gorm.io/gorm"

	"sslplatform/internal/common/apperror"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, item *DataSource) error {
	return apperror.FromDB(r.db.WithContext(ctx).Create(item).Error, "数据源不存在")
}

func (r *Repository) Save(ctx context.Context, item *DataSource) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *Repository) Get(ctx context.Context, tenantID, id uint) (*DataSource, error) {
	var item DataSource
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&item).Error
	if err != nil {
		return nil, apperror.FromDB(err, "数据源不存在")
	}
	return &item, nil
}

func (r *Repository) List(ctx context.Context, tenantID uint, page, size int, filter ListFilter) ([]DataSource, int64, error) {
	q := r.db.WithContext(ctx).Model(&DataSource{}).Where("tenant_id = ?", tenantID)
	if filter.Type != "" {
		q = q.Where("type = ?", filter.Type)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		q = q.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []DataSource
	err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

func (r *Repository) Delete(ctx context.Context, tenantID, id uint) error {
	var refs int64
	if err := r.db.WithContext(ctx).Table("certificates").
		Where("tenant_id = ? AND (acme_data_source_id = ? OR nginx_data_source_id = ?)", tenantID, id, id).
		Count(&refs).Error; err != nil {
		return err
	}
	if refs > 0 {
		return apperror.New(409, "数据源仍被证书引用")
	}
	res := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&DataSource{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperror.New(404, "数据源不存在")
	}
	return nil
}
