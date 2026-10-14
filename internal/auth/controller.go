package auth

import (
	"github.com/gin-gonic/gin"

	"sslplatform/internal/common/bind"
	"sslplatform/internal/common/response"
	"sslplatform/internal/common/tenant"
	"sslplatform/internal/httpapi"
)

type Controller struct {
	svc *Service
	mw  *Middleware
}

func NewController(svc *Service, mw *Middleware) *Controller {
	return &Controller{svc: svc, mw: mw}
}

func RegisterRouter(r httpapi.Router, c *Controller, mw *Middleware) {
	r.Public(httpapi.Meta{
		Resource: "auth", Action: "login", Path: "/auth/login",
		Summary: "使用用户名和密码登录，返回 JWT。租户由账号归属决定",
		Fields: []httpapi.Field{
			{Name: "username", Type: "string", Required: true, Description: "登录用户名"},
			{Name: "password", Type: "string", Required: true, Description: "登录密码"},
		},
	}, mw.LimitLogin(), c.Login)

	r.Login(httpapi.Meta{
		Resource: "auth", Action: "profile", Path: "/auth/profile",
		Summary: "读取当前登录用户、租户、角色和权限",
	}, c.Profile)

	r.Login(httpapi.Meta{
		Resource: "auth", Action: "changePassword", Path: "/auth/changePassword",
		Summary: "修改当前用户密码",
		Fields: []httpapi.Field{
			{Name: "oldPassword", Type: "string", Required: true, Description: "当前密码"},
			{Name: "newPassword", Type: "string", Required: true, Description: "新密码，至少 8 位"},
		},
	}, c.ChangePassword)

	r.Protected(httpapi.Meta{
		Resource: "user", Action: "create", Path: "/user/create",
		Summary: "在当前租户创建用户并分配角色",
		Fields: []httpapi.Field{
			{Name: "username", Type: "string", Required: true, Description: "用户名"},
			{Name: "password", Type: "string", Required: true, Description: "初始密码"},
			{Name: "displayName", Type: "string", Required: false, Description: "显示名"},
			{Name: "roleIds", Type: "array<number>", Required: false, Description: "角色 ID 列表"},
		},
	}, c.CreateUser)

	r.Protected(httpapi.Meta{
		Resource: "user", Action: "update", Path: "/user/update",
		Summary: "更新当前租户内用户的显示名、状态或密码",
		Fields: []httpapi.Field{
			{Name: "id", Type: "number", Required: true, Description: "用户 ID"},
			{Name: "displayName", Type: "string", Required: false, Description: "显示名"},
			{Name: "status", Type: "string", Required: false, Description: "active 或 disabled"},
			{Name: "password", Type: "string", Required: false, Description: "重置后的密码"},
		},
	}, c.UpdateUser)

	r.Protected(httpapi.Meta{
		Resource: "user", Action: "delete", Path: "/user/delete",
		Summary: "删除当前租户内的用户",
		Fields:  []httpapi.Field{{Name: "id", Type: "number", Required: true, Description: "用户 ID"}},
	}, c.DeleteUser)

	r.Protected(httpapi.Meta{
		Resource: "user", Action: "list", Path: "/user/list",
		Summary: "分页查询当前租户用户",
		Fields: []httpapi.Field{
			{Name: "page", Type: "number", Required: false, Description: "页码，从 1 开始"},
			{Name: "size", Type: "number", Required: false, Description: "每页条数，最大 100"},
			{Name: "filters.keyword", Type: "string", Required: false, Description: "用户名或显示名关键字"},
			{Name: "filters.status", Type: "string", Required: false, Description: "账号状态"},
		},
	}, c.ListUsers)

	r.Protected(httpapi.Meta{
		Resource: "user", Action: "assignRole", Path: "/user/assignRole",
		Summary: "替换用户在当前租户内的角色",
		Fields: []httpapi.Field{
			{Name: "userId", Type: "number", Required: true, Description: "用户 ID"},
			{Name: "roleIds", Type: "array<number>", Required: true, Description: "角色 ID 列表"},
		},
	}, c.AssignRole)

	r.Protected(httpapi.Meta{
		Resource: "role", Action: "create", Path: "/role/create",
		Summary: "创建自定义角色",
		Fields: []httpapi.Field{
			{Name: "code", Type: "string", Required: true, Description: "角色编码"},
			{Name: "name", Type: "string", Required: true, Description: "角色名称"},
			{Name: "description", Type: "string", Required: false, Description: "说明"},
		},
	}, c.CreateRole)

	r.Protected(httpapi.Meta{
		Resource: "role", Action: "update", Path: "/role/update",
		Summary: "更新角色名称和说明",
		Fields: []httpapi.Field{
			{Name: "id", Type: "number", Required: true, Description: "角色 ID"},
			{Name: "name", Type: "string", Required: false, Description: "角色名称"},
			{Name: "description", Type: "string", Required: false, Description: "说明"},
		},
	}, c.UpdateRole)

	r.Protected(httpapi.Meta{
		Resource: "role", Action: "list", Path: "/role/list",
		Summary: "分页查询当前租户角色",
		Fields: []httpapi.Field{
			{Name: "page", Type: "number", Required: false, Description: "页码"},
			{Name: "size", Type: "number", Required: false, Description: "每页条数"},
		},
	}, c.ListRoles)

	r.Protected(httpapi.Meta{
		Resource: "role", Action: "detail", Path: "/role/detail",
		Summary: "查看角色及其权限",
		Fields:  []httpapi.Field{{Name: "id", Type: "number", Required: true, Description: "角色 ID"}},
	}, c.RoleDetail)

	r.Protected(httpapi.Meta{
		Resource: "role", Action: "grant", Path: "/role/grant",
		Summary: "用权限 ID 列表替换角色授权",
		Fields: []httpapi.Field{
			{Name: "roleId", Type: "number", Required: true, Description: "角色 ID"},
			{Name: "permissionIds", Type: "array<number>", Required: true, Description: "权限 ID 列表"},
		},
	}, c.Grant)

	r.Protected(httpapi.Meta{
		Resource: "role", Action: "permissionList", Path: "/role/permissionList",
		Summary: "列出系统全部权限点",
	}, c.PermissionList)
}

func (c *Controller) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	view, err := c.svc.Login(ctx.Request.Context(), req)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, view)
}

func (c *Controller) Profile(ctx *gin.Context) {
	tc, _ := tenant.From(ctx.Request.Context())
	view, err := c.svc.Profile(ctx.Request.Context(), tc.UserID, tc.TenantID)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, view)
}

func (c *Controller) ChangePassword(ctx *gin.Context) {
	var req ChangePasswordRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	if err := c.svc.ChangePassword(ctx.Request.Context(), tc.UserID, tc.TenantID, req); err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, map[string]any{"changed": true})
}

func (c *Controller) CreateUser(ctx *gin.Context) {
	var req UserCreateRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	view, err := c.svc.CreateUser(ctx.Request.Context(), tc.TenantID, req)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, view)
}

func (c *Controller) UpdateUser(ctx *gin.Context) {
	var req UserUpdateRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	view, err := c.svc.UpdateUser(ctx.Request.Context(), tc.UserID, tc.TenantID, req)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, view)
}

func (c *Controller) DeleteUser(ctx *gin.Context) {
	var req UserDeleteRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	if err := c.svc.DeleteUser(ctx.Request.Context(), tc.UserID, tc.TenantID, req.ID); err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, map[string]any{"id": req.ID})
}

func (c *Controller) ListUsers(ctx *gin.Context) {
	var req UserListRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	rows, page, size, total, err := c.svc.ListUsers(ctx.Request.Context(), tc.TenantID, req)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.Page(ctx, rows, page, size, total)
}

func (c *Controller) AssignRole(ctx *gin.Context) {
	var req UserAssignRoleRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	view, err := c.svc.AssignRoles(ctx.Request.Context(), tc.TenantID, req)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, view)
}

func (c *Controller) CreateRole(ctx *gin.Context) {
	var req RoleCreateRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	view, err := c.svc.CreateRole(ctx.Request.Context(), tc.TenantID, req)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, view)
}

func (c *Controller) UpdateRole(ctx *gin.Context) {
	var req RoleUpdateRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	view, err := c.svc.UpdateRole(ctx.Request.Context(), tc.TenantID, req)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, view)
}

func (c *Controller) ListRoles(ctx *gin.Context) {
	var req RoleListRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	rows, page, size, total, err := c.svc.ListRoles(ctx.Request.Context(), tc.TenantID, req)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.Page(ctx, rows, page, size, total)
}

func (c *Controller) RoleDetail(ctx *gin.Context) {
	var req RoleDetailRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	view, err := c.svc.RoleDetail(ctx.Request.Context(), tc.TenantID, req.ID)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, view)
}

func (c *Controller) Grant(ctx *gin.Context) {
	var req RoleGrantRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	view, err := c.svc.Grant(ctx.Request.Context(), tc.TenantID, req)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, view)
}

func (c *Controller) PermissionList(ctx *gin.Context) {
	rows, err := c.svc.ListPermissionCatalog(ctx.Request.Context())
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, rows)
}
