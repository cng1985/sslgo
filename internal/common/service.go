package common

import "context"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Search(ctx context.Context, tenantID uint, req SearchRequest) ([]map[string]any, int, int, int64, error) {
	return s.repo.Search(ctx, tenantID, req)
}

func (s *Service) List(ctx context.Context, tenantID uint, req ListRequest) ([]map[string]any, int, int, int64, error) {
	return s.repo.List(ctx, tenantID, req)
}

func (s *Service) One(ctx context.Context, tenantID uint, req OneRequest) (map[string]any, error) {
	return s.repo.One(ctx, tenantID, req)
}
