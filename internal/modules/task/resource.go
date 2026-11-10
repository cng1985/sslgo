package task

import (
	"context"

	"sslplatform/internal/platform/domain"
	"sslplatform/internal/platform/errors"
	"sslplatform/internal/platform/response"
)

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

func (s *Resource) Detail(ctx context.Context, req *DetailRequest) (response.Response[View], error) {
	if req == nil {
		return response.Response[View]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[View]{}, err
	}
	view, err := s.detail(ctx, scope.TenantID, req.ID)
	if err != nil {
		return response.Response[View]{}, err
	}
	return response.Success(*view), nil
}
