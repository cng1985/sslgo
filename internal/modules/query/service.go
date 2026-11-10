package query

import "context"

type Resource struct {
	repo *Repository
}

func NewResource(repo *Repository) *Resource {
	return &Resource{repo: repo}
}

func (s *Resource) search(ctx context.Context, tenantID uint, req SearchRequest) ([]map[string]any, int, int, int64, error) {
	return s.repo.Search(ctx, tenantID, req)
}

func (s *Resource) list(ctx context.Context, tenantID uint, req ListRequest) ([]map[string]any, int, int, int64, error) {
	return s.repo.List(ctx, tenantID, req)
}

func (s *Resource) one(ctx context.Context, tenantID uint, req OneRequest) (map[string]any, error) {
	return s.repo.One(ctx, tenantID, req)
}
