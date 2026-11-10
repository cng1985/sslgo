package query

import (
	"context"

	"sslplatform/internal/platform/domain"
	"sslplatform/internal/platform/errors"
	"sslplatform/internal/platform/response"
)

func (s *Resource) Search(ctx context.Context, req *SearchRequest) (response.PageResponse[map[string]any], error) {
	if req == nil {
		return response.PageResponse[map[string]any]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.PageResponse[map[string]any]{}, err
	}
	rows, page, size, total, err := s.search(ctx, scope.TenantID, *req)
	if err != nil {
		return response.PageResponse[map[string]any]{}, err
	}
	return response.PageOf(rows, page, size, total), nil
}

func (s *Resource) List(ctx context.Context, req *ListRequest) (response.PageResponse[map[string]any], error) {
	if req == nil {
		return response.PageResponse[map[string]any]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.PageResponse[map[string]any]{}, err
	}
	rows, page, size, total, err := s.list(ctx, scope.TenantID, *req)
	if err != nil {
		return response.PageResponse[map[string]any]{}, err
	}
	return response.PageOf(rows, page, size, total), nil
}

func (s *Resource) One(ctx context.Context, req *OneRequest) (response.Response[map[string]any], error) {
	if req == nil {
		return response.Response[map[string]any]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[map[string]any]{}, err
	}
	row, err := s.one(ctx, scope.TenantID, *req)
	if err != nil {
		return response.Response[map[string]any]{}, err
	}
	if row == nil {
		row = map[string]any{}
	}
	return response.Success(row), nil
}
