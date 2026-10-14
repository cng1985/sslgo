package scheduler

import (
	"github.com/gin-gonic/gin"

	"sslplatform/internal/common/bind"
	"sslplatform/internal/common/response"
	"sslplatform/internal/common/tenant"
	"sslplatform/internal/httpapi"
)

type Controller struct {
	svc *Service
}

func NewController(svc *Service) *Controller {
	return &Controller{svc: svc}
}

func RegisterRouter(r httpapi.Router, c *Controller) {
	r.Protected(httpapi.Meta{
		Resource: "schedule", Action: "create", Path: "/schedule/create",
		Summary: "为签发流水线创建定时续期。到期窗口外的执行会自动跳过",
		Fields: []httpapi.Field{
			{Name: "name", Type: "string", Required: true, Description: "调度名称"},
			{Name: "etlJobId", Type: "number", Required: true, Description: "流水线 ID"},
			{Name: "cron", Type: "string", Required: true, Description: "5 段 Cron，例如 15 3 * * *"},
			{Name: "enabled", Type: "boolean", Required: false, Description: "是否立即启用"},
		},
	}, c.Create)
	r.Protected(httpapi.Meta{
		Resource: "schedule", Action: "update", Path: "/schedule/update",
		Summary: "更新调度名称、Cron 或启用状态",
		Fields: []httpapi.Field{
			{Name: "id", Type: "number", Required: true, Description: "调度 ID"},
			{Name: "name", Type: "string", Required: false, Description: "名称"},
			{Name: "cron", Type: "string", Required: false, Description: "5 段 Cron"},
			{Name: "enabled", Type: "boolean", Required: false, Description: "是否启用"},
		},
	}, c.Update)
	r.Protected(httpapi.Meta{
		Resource: "schedule", Action: "delete", Path: "/schedule/delete",
		Summary: "删除调度",
		Fields:  []httpapi.Field{{Name: "id", Type: "number", Required: true, Description: "调度 ID"}},
	}, c.Delete)
	r.Protected(httpapi.Meta{
		Resource: "schedule", Action: "list", Path: "/schedule/list",
		Summary: "分页查询调度",
		Fields: []httpapi.Field{
			{Name: "page", Type: "number", Required: false, Description: "页码"},
			{Name: "size", Type: "number", Required: false, Description: "每页条数"},
			{Name: "filters.etlJobId", Type: "number", Required: false, Description: "流水线 ID"},
		},
	}, c.List)
	r.Protected(httpapi.Meta{
		Resource: "schedule", Action: "enable", Path: "/schedule/enable",
		Summary: "启用调度",
		Fields:  []httpapi.Field{{Name: "id", Type: "number", Required: true, Description: "调度 ID"}},
	}, c.Enable)
	r.Protected(httpapi.Meta{
		Resource: "schedule", Action: "disable", Path: "/schedule/disable",
		Summary: "停用调度",
		Fields:  []httpapi.Field{{Name: "id", Type: "number", Required: true, Description: "调度 ID"}},
	}, c.Disable)
}

func (c *Controller) Create(ctx *gin.Context) {
	var req CreateRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	view, err := c.svc.Create(ctx.Request.Context(), tc.TenantID, req)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, view)
}

func (c *Controller) Update(ctx *gin.Context) {
	var req UpdateRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	view, err := c.svc.Update(ctx.Request.Context(), tc.TenantID, req)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, view)
}

func (c *Controller) Delete(ctx *gin.Context) {
	var req IDRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	if err := c.svc.Delete(ctx.Request.Context(), tc.TenantID, req.ID); err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, map[string]any{"id": req.ID})
}

func (c *Controller) List(ctx *gin.Context) {
	var req ListRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	rows, page, size, total, err := c.svc.List(ctx.Request.Context(), tc.TenantID, req)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.Page(ctx, rows, page, size, total)
}

func (c *Controller) Enable(ctx *gin.Context) {
	c.setEnabled(ctx, true)
}

func (c *Controller) Disable(ctx *gin.Context) {
	c.setEnabled(ctx, false)
}

func (c *Controller) setEnabled(ctx *gin.Context, enabled bool) {
	var req IDRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	view, err := c.svc.Enable(ctx.Request.Context(), tc.TenantID, req.ID, enabled)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, view)
}
