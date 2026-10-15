package task

import (
	"context"
	"encoding/json"

	"sslplatform/internal/paging"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, tenantID uint, req ListRequest) ([]View, int, int, int64, error) {
	page, size := paging.Norm(req.Page, req.Size)
	rows, total, err := s.repo.List(ctx, tenantID, page, size, req.Filters)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	views := make([]View, 0, len(rows))
	for i := range rows {
		views = append(views, toView(&rows[i]))
	}
	return views, page, size, total, nil
}

func (s *Service) Detail(ctx context.Context, tenantID, id uint) (*View, error) {
	item, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	view := toView(item)
	return &view, nil
}

// ViewFrom 供其他模块把任务记录转换成响应。
func ViewFrom(item *Task) View { return toView(item) }

func toView(item *Task) View {
	result := map[string]any{}
	if item.ResultJSON != "" {
		_ = json.Unmarshal([]byte(item.ResultJSON), &result)
	}
	return View{
		ID: item.ID, ETLJobID: item.ETLJobID, CertificateID: item.CertificateID,
		UserID: item.UserID, Trigger: item.Trigger, Status: item.Status, Stage: item.Stage,
		Force: item.Force, Deploy: item.Deploy, Error: item.Error, Result: result,
		StartedAt: item.StartedAt, FinishedAt: item.FinishedAt, CreatedAt: item.CreatedAt,
	}
}
