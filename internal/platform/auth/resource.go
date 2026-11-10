package auth

import (
	"context"

	"sslplatform/internal/platform/domain"
	"sslplatform/internal/platform/errors"
	"sslplatform/internal/platform/response"
)

type ProfileRequest struct{}

type PasswordChanged struct {
	Changed bool `json:"changed"`
}

type Deleted struct {
	ID uint `json:"id"`
}

type PermissionListRequest struct{}

func (s *Resource) Login(ctx context.Context, req *LoginRequest) (response.Response[LoginView], error) {
	if req == nil {
		return response.Response[LoginView]{}, apperror.New(400, "请求不能为空")
	}
	view, err := s.login(ctx, *req)
	if err != nil {
		return response.Response[LoginView]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) Profile(ctx context.Context, _ *ProfileRequest) (response.Response[ProfileView], error) {
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[ProfileView]{}, err
	}
	view, err := s.profile(ctx, scope.UserID, scope.TenantID)
	if err != nil {
		return response.Response[ProfileView]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) ChangePassword(ctx context.Context, req *ChangePasswordRequest) (response.Response[PasswordChanged], error) {
	if req == nil {
		return response.Response[PasswordChanged]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[PasswordChanged]{}, err
	}
	if err := s.changePassword(ctx, scope.UserID, scope.TenantID, *req); err != nil {
		return response.Response[PasswordChanged]{}, err
	}
	return response.Success(PasswordChanged{Changed: true}), nil
}

func (s *Resource) CreateUser(ctx context.Context, req *UserCreateRequest) (response.Response[UserView], error) {
	if req == nil {
		return response.Response[UserView]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[UserView]{}, err
	}
	view, err := s.createUser(ctx, scope.TenantID, *req)
	if err != nil {
		return response.Response[UserView]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) UpdateUser(ctx context.Context, req *UserUpdateRequest) (response.Response[UserView], error) {
	if req == nil {
		return response.Response[UserView]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[UserView]{}, err
	}
	view, err := s.updateUser(ctx, scope.UserID, scope.TenantID, *req)
	if err != nil {
		return response.Response[UserView]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) DeleteUser(ctx context.Context, req *UserDeleteRequest) (response.Response[Deleted], error) {
	if req == nil {
		return response.Response[Deleted]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[Deleted]{}, err
	}
	if err := s.deleteUser(ctx, scope.UserID, scope.TenantID, req.ID); err != nil {
		return response.Response[Deleted]{}, err
	}
	return response.Success(Deleted{ID: req.ID}), nil
}

func (s *Resource) ListUsers(ctx context.Context, req *UserListRequest) (response.PageResponse[UserView], error) {
	if req == nil {
		req = &UserListRequest{}
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.PageResponse[UserView]{}, err
	}
	rows, page, size, total, err := s.listUsers(ctx, scope.TenantID, *req)
	if err != nil {
		return response.PageResponse[UserView]{}, err
	}
	return response.PageOf(rows, page, size, total), nil
}

func (s *Resource) AssignRoles(ctx context.Context, req *UserAssignRoleRequest) (response.Response[UserView], error) {
	if req == nil {
		return response.Response[UserView]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[UserView]{}, err
	}
	view, err := s.assignRoles(ctx, scope.TenantID, *req)
	if err != nil {
		return response.Response[UserView]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) CreateRole(ctx context.Context, req *RoleCreateRequest) (response.Response[RoleView], error) {
	if req == nil {
		return response.Response[RoleView]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[RoleView]{}, err
	}
	view, err := s.createRole(ctx, scope.TenantID, *req)
	if err != nil {
		return response.Response[RoleView]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) UpdateRole(ctx context.Context, req *RoleUpdateRequest) (response.Response[RoleView], error) {
	if req == nil {
		return response.Response[RoleView]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[RoleView]{}, err
	}
	view, err := s.updateRole(ctx, scope.TenantID, *req)
	if err != nil {
		return response.Response[RoleView]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) ListRoles(ctx context.Context, req *RoleListRequest) (response.PageResponse[RoleView], error) {
	if req == nil {
		req = &RoleListRequest{}
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.PageResponse[RoleView]{}, err
	}
	rows, page, size, total, err := s.listRoles(ctx, scope.TenantID, *req)
	if err != nil {
		return response.PageResponse[RoleView]{}, err
	}
	return response.PageOf(rows, page, size, total), nil
}

func (s *Resource) RoleDetail(ctx context.Context, req *RoleDetailRequest) (response.Response[RoleView], error) {
	if req == nil {
		return response.Response[RoleView]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[RoleView]{}, err
	}
	view, err := s.roleDetail(ctx, scope.TenantID, req.ID)
	if err != nil {
		return response.Response[RoleView]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) Grant(ctx context.Context, req *RoleGrantRequest) (response.Response[RoleView], error) {
	if req == nil {
		return response.Response[RoleView]{}, apperror.New(400, "请求不能为空")
	}
	scope, err := domain.Must(ctx)
	if err != nil {
		return response.Response[RoleView]{}, err
	}
	view, err := s.grant(ctx, scope.TenantID, *req)
	if err != nil {
		return response.Response[RoleView]{}, err
	}
	return response.Success(*view), nil
}

func (s *Resource) ListPermissionCatalog(ctx context.Context, _ *PermissionListRequest) (response.Response[[]Permission], error) {
	if _, err := domain.Must(ctx); err != nil {
		return response.Response[[]Permission]{}, err
	}
	rows, err := s.listPermissionCatalog(ctx)
	if err != nil {
		return response.Response[[]Permission]{}, err
	}
	if rows == nil {
		rows = []Permission{}
	}
	return response.Success(rows), nil
}
