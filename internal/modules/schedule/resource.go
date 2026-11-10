package schedule

import (
	"context"

	"sslplatform/internal/platform/domain"
	"sslplatform/internal/platform/errors"
	"sslplatform/internal/platform/response"
)

type Deleted struct {
	ID uint `json:"id"`
}

func (s *Resource) Create(ctx context.Context, req *CreateRequest) (response.Response[View], error) {
	if req == nil {
		return response.Response[View]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[View]{}, err
	}
	view, err := s.create(ctx, scope.TenantID, *req)
	if err != nil {
		return response.Response[View]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) Update(ctx context.Context, req *UpdateRequest) (response.Response[View], error) {
	if req == nil {
		return response.Response[View]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[View]{}, err
	}
	view, err := s.update(ctx, scope.TenantID, *req)
	if err != nil {
		return response.Response[View]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) Delete(ctx context.Context, req *IDRequest) (response.Response[Deleted], error) {
	if req == nil {
		return response.Response[Deleted]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[Deleted]{}, err
	}
	if err := s.delete(ctx, scope.TenantID, req.ID); err != nil {
		return response.Response[Deleted]{}, err
	}
	return response.Success(Deleted{ID: req.ID}), nil
}

func (s *Resource) List(ctx context.Context, req *ListRequest) (response.PageResponse[View], error) {
	if req == nil {
		req = &ListRequest{}
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.PageResponse[View]{}, err
	}
	rows, page, size, total, err := s.list(ctx, scope.TenantID, *req)
	if err != nil {
		return response.PageResponse[View]{}, err
	}
	return response.PageOf(rows, page, size, total), nil
}

func (s *Resource) Enable(ctx context.Context, req *IDRequest) (response.Response[View], error) {
	return s.setEnabled(ctx, req, true)
}

func (s *Resource) Disable(ctx context.Context, req *IDRequest) (response.Response[View], error) {
	return s.setEnabled(ctx, req, false)
}

func (s *Resource) setEnabled(ctx context.Context, req *IDRequest, enabled bool) (response.Response[View], error) {
	if req == nil {
		return response.Response[View]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[View]{}, err
	}
	view, err := s.enable(ctx, scope.TenantID, req.ID, enabled)
	if err != nil {
		return response.Response[View]{}, err
	}
	return response.Success(*view), nil
}
