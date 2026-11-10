package query

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
	fields := []routing.Field{
		{Name: "model", Type: "string", Required: true, Description: "模型名：certificate、etlJob、task、datasource、user、schedule、role，也接受 pageCertificate、pageETLJob 等别名"},
		{Name: "page", Type: "number", Required: false, Description: "页码，从 1 开始"},
		{Name: "size", Type: "number", Required: false, Description: "每页条数，最大 100"},
		{Name: "filters", Type: "object", Required: false, Description: "字段过滤。等值直接赋值，也可使用 like、in、gt、gte、lt、lte"},
	}
	r.Protected(routing.Meta{
		Resource: "query", Action: "search", Path: "/query/search",
		Summary: "在当前租户内按关键字搜索白名单模型",
		Fields:  append(fields, routing.Field{Name: "keyword", Type: "string", Required: false, Description: "关键字，匹配模型预定义文本列"}),
	}, c.Search)
	r.Protected(routing.Meta{
		Resource: "query", Action: "list", Path: "/query/list",
		Summary: "在当前租户内分页列出白名单模型",
		Fields:  fields,
	}, c.List)
	r.Protected(routing.Meta{
		Resource: "query", Action: "one", Path: "/query/one",
		Summary: "在当前租户内按条件返回一条记录",
		Fields: []routing.Field{
			{Name: "model", Type: "string", Required: true, Description: "模型名"},
			{Name: "filters", Type: "object", Required: true, Description: "过滤条件，例如 {\"id\": 1}"},
		},
	}, c.One)
}

func (c *Controller) Search(ctx *gin.Context) {
	var req SearchRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.Search(ctx.Request.Context(), &req)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}
	response.SuccessPage(ctx, result)
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

func (c *Controller) One(ctx *gin.Context) {
	var req OneRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.HandleError(ctx, err)
		return
	}
	result, err := c.svc.One(ctx.Request.Context(), &req)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}
	response.SuccessResponse(ctx, result)
}
