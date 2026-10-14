package common

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
	fields := []httpapi.Field{
		{Name: "model", Type: "string", Required: true, Description: "模型名：certificate、etlJob、task、datasource、user、schedule、role，也接受 pageCertificate、pageETLJob 等别名"},
		{Name: "page", Type: "number", Required: false, Description: "页码，从 1 开始"},
		{Name: "size", Type: "number", Required: false, Description: "每页条数，最大 100"},
		{Name: "filters", Type: "object", Required: false, Description: "字段过滤。等值直接赋值，也可使用 like、in、gt、gte、lt、lte"},
	}
	r.Protected(httpapi.Meta{
		Resource: "query", Action: "search", Path: "/query/search",
		Summary: "在当前租户内按关键字搜索白名单模型",
		Fields:  append(fields, httpapi.Field{Name: "keyword", Type: "string", Required: false, Description: "关键字，匹配模型预定义文本列"}),
	}, c.Search)
	r.Protected(httpapi.Meta{
		Resource: "query", Action: "list", Path: "/query/list",
		Summary: "在当前租户内分页列出白名单模型",
		Fields:  fields,
	}, c.List)
	r.Protected(httpapi.Meta{
		Resource: "query", Action: "one", Path: "/query/one",
		Summary: "在当前租户内按条件返回一条记录",
		Fields: []httpapi.Field{
			{Name: "model", Type: "string", Required: true, Description: "模型名"},
			{Name: "filters", Type: "object", Required: true, Description: "过滤条件，例如 {\"id\": 1}"},
		},
	}, c.One)
}

func (c *Controller) Search(ctx *gin.Context) {
	var req SearchRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	rows, page, size, total, err := c.svc.Search(ctx.Request.Context(), tc.TenantID, req)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.Page(ctx, rows, page, size, total)
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

func (c *Controller) One(ctx *gin.Context) {
	var req OneRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	row, err := c.svc.One(ctx.Request.Context(), tc.TenantID, req)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, row)
}
