package datasource

import (
	"context"

	"sslplatform/internal/modules/acme"
	"sslplatform/internal/platform/domain"
	"sslplatform/internal/platform/errors"
	"sslplatform/internal/platform/response"
)

// Deleted 是删除成功回执。
type Deleted struct {
	ID uint `json:"id"`
}

// CatalogRequest 没有业务字段。目录由服务端维护。
type CatalogRequest struct{}

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

func (s *Resource) Detail(ctx context.Context, req *IDRequest) (response.Response[View], error) {
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

func (s *Resource) Register(ctx context.Context, req *RegisterAccountRequest) (response.Response[View], error) {
	if req == nil {
		return response.Response[View]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[View]{}, err
	}
	view, err := s.registerAccount(ctx, scope.TenantID, *req)
	if err != nil {
		return response.Response[View]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) Catalog(ctx context.Context, _ *CatalogRequest) (response.Response[acme.ProviderCatalog], error) {
	if _, err := domain.Must(ctx); err != nil {
		return response.Response[acme.ProviderCatalog]{}, err
	}
	return response.Success(acme.Catalog()), nil
}

func (s *Resource) Test(ctx context.Context, req *IDRequest) (response.Response[TestView], error) {
	if req == nil {
		return response.Response[TestView]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[TestView]{}, err
	}
	view, err := s.test(ctx, scope.TenantID, req.ID)
	if err != nil {
		return response.Response[TestView]{}, err
	}
	return response.Success(*view), nil
}
