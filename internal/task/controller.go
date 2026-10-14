package task

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
		Resource: "task", Action: "list", Path: "/task/list",
		Summary: "分页查询证书签发与部署任务",
		Fields: []httpapi.Field{
			{Name: "page", Type: "number", Required: false, Description: "页码"},
			{Name: "size", Type: "number", Required: false, Description: "每页条数"},
			{Name: "filters.status", Type: "string", Required: false, Description: "pending、running、success、failed 或 skipped"},
			{Name: "filters.certificateId", Type: "number", Required: false, Description: "证书 ID"},
			{Name: "filters.etlJobId", Type: "number", Required: false, Description: "流水线 ID"},
		},
	}, c.List)
	r.Protected(httpapi.Meta{
		Resource: "task", Action: "detail", Path: "/task/detail",
		Summary: "查看任务阶段、结果和错误",
		Fields:  []httpapi.Field{{Name: "id", Type: "number", Required: true, Description: "任务 ID"}},
	}, c.Detail)
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

func (c *Controller) Detail(ctx *gin.Context) {
	var req DetailRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	view, err := c.svc.Detail(ctx.Request.Context(), tc.TenantID, req.ID)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, view)
}
