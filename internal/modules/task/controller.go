package task

import (
	"github.com/gin-gonic/gin"

	"sslplatform/internal/platform/bind"
	"sslplatform/internal/platform/response"
	"sslplatform/internal/platform/routing"
)

type Controller struct {
	svc *Resource
}

func NewController(svc *Resource) *Controller {
	return &Controller{svc: svc}
}

func RegisterRouter(r routing.Router, c *Controller) {
	r.Protected(routing.Meta{
		Resource: "task", Action: "list", Path: "/task/list",
		Summary: "分页查询证书签发与部署任务",
		Fields: []routing.Field{
			{Name: "page", Type: "number", Required: false, Description: "页码"},
			{Name: "size", Type: "number", Required: false, Description: "每页条数"},
			{Name: "filters.status", Type: "string", Required: false, Description: "pending、running、success、failed 或 skipped"},
			{Name: "filters.certificateId", Type: "number", Required: false, Description: "证书 ID"},
			{Name: "filters.etlJobId", Type: "number", Required: false, Description: "流水线 ID"},
		},
	}, c.List)
	r.Protected(routing.Meta{
		Resource: "task", Action: "detail", Path: "/task/detail",
		Summary: "查看任务阶段、结果和错误",
		Fields:  []routing.Field{{Name: "id", Type: "number", Required: true, Description: "任务 ID"}},
	}, c.Detail)
}

func (c *Controller) List(ctx *gin.Context) {
	var req ListRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.List(ctx.Request.Context(), &req)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}
	response.SuccessPage(ctx, result)
}

func (c *Controller) Detail(ctx *gin.Context) {
	var req DetailRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.Detail(ctx.Request.Context(), &req)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}
	response.SuccessResponse(ctx, result)
}
