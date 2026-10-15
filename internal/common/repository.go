package common

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"gorm.io/gorm"

	"sslplatform/internal/common/apperror"
	"sslplatform/internal/datasource"
	"sslplatform/internal/paging"
)

type modelSpec struct {
	Table    string
	Columns  []string
	Filters  map[string]string
	Search   []string
	Sanitize func(map[string]any)
}

type Repository struct {
	db    *gorm.DB
	specs map[string]modelSpec
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db, specs: registry()}
}

func (r *Repository) Search(ctx context.Context, tenantID uint, req SearchRequest) ([]map[string]any, int, int, int64, error) {
	return r.query(ctx, tenantID, req.Model, req.Page, req.Size, req.Keyword, req.Filters, false)
}

func (r *Repository) List(ctx context.Context, tenantID uint, req ListRequest) ([]map[string]any, int, int, int64, error) {
	return r.query(ctx, tenantID, req.Model, req.Page, req.Size, "", req.Filters, false)
}

func (r *Repository) One(ctx context.Context, tenantID uint, req OneRequest) (map[string]any, error) {
	rows, _, _, _, err := r.query(ctx, tenantID, req.Model, 1, 1, "", req.Filters, true)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, apperror.New(404, "记录不存在")
	}
	return rows[0], nil
}

func (r *Repository) query(ctx context.Context, tenantID uint, model string, page, size int, keyword string, filters map[string]any, one bool) ([]map[string]any, int, int, int64, error) {
	spec, ok := r.specs[canonicalModel(model)]
	if !ok {
		return nil, 0, 0, 0, apperror.New(400, "不支持的查询模型")
	}
	page, size = paging.Norm(page, size)
	if one {
		page, size = 1, 1
	}
	build := func() (*gorm.DB, error) {
		q := r.db.WithContext(ctx).Table(spec.Table).Where("tenant_id = ?", tenantID)
		var err error
		q, err = applyFilters(q, spec, filters)
		if err != nil {
			return nil, err
		}
		q, err = applyKeyword(q, spec, keyword)
		return q, err
	}
	countQuery, err := build()
	if err != nil {
		return nil, 0, 0, 0, err
	}
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, 0, 0, err
	}
	findQuery, err := build()
	if err != nil {
		return nil, 0, 0, 0, err
	}
	var rows []map[string]any
	err = findQuery.Select(spec.Columns).Order("id desc").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	if err != nil {
		return nil, 0, 0, 0, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		mapped := map[string]any{}
		for key, value := range row {
			mapped[camel(key)] = normalize(value)
		}
		if spec.Sanitize != nil {
			spec.Sanitize(mapped)
		}
		out = append(out, mapped)
	}
	return out, page, size, total, nil
}

func applyFilters(db *gorm.DB, spec modelSpec, filters map[string]any) (*gorm.DB, error) {
	for key, value := range filters {
		column, ok := spec.Filters[key]
		if !ok {
			return nil, apperror.New(400, "不支持的过滤字段 "+key)
		}
		switch typed := value.(type) {
		case map[string]any:
			for op, raw := range typed {
				switch op {
				case "eq":
					db = db.Where(column+" = ?", raw)
				case "like":
					text, ok := raw.(string)
					if !ok {
						return nil, apperror.New(400, key+" like 需要字符串")
					}
					db = db.Where(column+" LIKE ? ESCAPE '\\'", likeValue(text))
				case "in":
					db = db.Where(column+" IN ?", raw)
				case "gt":
					db = db.Where(column+" > ?", raw)
				case "gte":
					db = db.Where(column+" >= ?", raw)
				case "lt":
					db = db.Where(column+" < ?", raw)
				case "lte":
					db = db.Where(column+" <= ?", raw)
				default:
					return nil, apperror.New(400, "不支持的过滤操作 "+op)
				}
			}
		default:
			db = db.Where(column+" = ?", value)
		}
	}
	return db, nil
}

func applyKeyword(db *gorm.DB, spec modelSpec, keyword string) (*gorm.DB, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return db, nil
	}
	if len(spec.Search) == 0 {
		return nil, apperror.New(400, "该模型不支持关键字搜索")
	}
	parts := make([]string, 0, len(spec.Search))
	args := make([]any, 0, len(spec.Search))
	for _, column := range spec.Search {
		parts = append(parts, column+" LIKE ? ESCAPE '\\'")
		args = append(args, likeValue(keyword))
	}
	return db.Where(strings.Join(parts, " OR "), args...), nil
}

func likeValue(s string) string {
	s = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
	return "%" + s + "%"
}

func camel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}

func normalize(value any) any {
	switch typed := value.(type) {
	case []byte:
		return string(typed)
	case time.Time:
		return typed.Format(time.RFC3339)
	default:
		return value
	}
}

func canonicalModel(model string) string {
	aliases := map[string]string{
		"pageETLJob":      "etlJob",
		"pageCertificate": "certificate",
		"pageDatasource":  "datasource",
		"pageTask":        "task",
		"pageUser":        "user",
		"pageSchedule":    "schedule",
		"pageRole":        "role",
	}
	if mapped, ok := aliases[model]; ok {
		return mapped
	}
	return model
}

func registry() map[string]modelSpec {
	return map[string]modelSpec{
		"certificate": {
			Table:   "certificates",
			Columns: []string{"id", "tenant_id", "name", "domains_json", "email", "challenge_type", "key_type", "acme_data_source_id", "nginx_data_source_id", "renew_before_days", "auto_renew", "agree_tos", "status", "not_before", "not_after", "fingerprint", "last_error", "last_issued_at", "last_deployed_at", "created_at", "updated_at"},
			Filters: map[string]string{"id": "id", "name": "name", "status": "status", "autoRenew": "auto_renew", "fingerprint": "fingerprint"},
			Search:  []string{"name", "domains_json", "fingerprint"},
			Sanitize: func(row map[string]any) {
				raw, _ := row["domainsJson"].(string)
				var domains []string
				if json.Unmarshal([]byte(raw), &domains) != nil || domains == nil {
					domains = []string{}
				}
				row["domains"] = domains
			},
		},
		"etlJob": {
			Table:   "etl_jobs",
			Columns: []string{"id", "tenant_id", "name", "certificate_id", "pipeline", "status", "created_at", "updated_at"},
			Filters: map[string]string{"id": "id", "name": "name", "certificateId": "certificate_id", "status": "status", "pipeline": "pipeline"},
			Search:  []string{"name", "pipeline"},
		},
		"task": {
			Table:   "tasks",
			Columns: []string{"id", "tenant_id", "user_id", "etl_job_id", "certificate_id", "trigger", "status", "stage", "force", "deploy", "error", "started_at", "finished_at", "created_at", "updated_at"},
			Filters: map[string]string{"id": "id", "status": "status", "certificateId": "certificate_id", "etlJobId": "etl_job_id", "trigger": "trigger"},
			Search:  []string{"status", "error", "stage"},
		},
		"datasource": {
			Table:   "datasources",
			Columns: []string{"id", "tenant_id", "name", "type", "description", "config_json", "status", "last_test_ok", "last_test_msg", "last_test_at", "created_at", "updated_at"},
			Filters: map[string]string{"id": "id", "name": "name", "type": "type", "status": "status"},
			Search:  []string{"name", "description", "type"},
			Sanitize: func(row map[string]any) {
				raw, _ := row["configJson"].(string)
				row["configJson"] = datasource.SanitizeConfigJSON(raw)
			},
		},
		"user": {
			Table:   "users",
			Columns: []string{"id", "tenant_id", "username", "display_name", "status", "must_change_password", "created_at", "updated_at"},
			Filters: map[string]string{"id": "id", "username": "username", "status": "status", "displayName": "display_name"},
			Search:  []string{"username", "display_name"},
		},
		"schedule": {
			Table:   "schedules",
			Columns: []string{"id", "tenant_id", "name", "etl_job_id", "cron", "enabled", "last_run_at", "next_run_at", "last_status", "created_at", "updated_at"},
			Filters: map[string]string{"id": "id", "name": "name", "etlJobId": "etl_job_id", "enabled": "enabled", "lastStatus": "last_status"},
			Search:  []string{"name", "cron"},
		},
		"role": {
			Table:   "roles",
			Columns: []string{"id", "tenant_id", "code", "name", "description", "built_in", "created_at", "updated_at"},
			Filters: map[string]string{"id": "id", "code": "code", "name": "name", "builtIn": "built_in"},
			Search:  []string{"code", "name", "description"},
		},
	}
}
