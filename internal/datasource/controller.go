package datasource

import (
	"github.com/gin-gonic/gin"

	"sslplatform/internal/acme"
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
		Resource: "datasource", Action: "create", Path: "/datasource/create",
		Summary: "创建 ACME 账户或 Nginx 部署目标",
		Fields: []httpapi.Field{
			{Name: "name", Type: "string", Required: true, Description: "数据源名称"},
			{Name: "type", Type: "string", Required: true, Description: "acme 或 nginx"},
			{Name: "description", Type: "string", Required: false, Description: "说明"},
			{Name: "config", Type: "object", Required: true, Description: "acme: email、ca、directoryURL、eabKid、eabHmacKey、termsAgreed、dnsProvider、dns、register；nginx: nginxBin、serviceManager、serviceName、action、webroot、certDir、snippetDir、testBefore、dryRun"},
		},
	}, c.Create)
	r.Protected(httpapi.Meta{
		Resource: "datasource", Action: "update", Path: "/datasource/update",
		Summary: "更新数据源。未提交的 ACME 私钥会保留",
		Fields: []httpapi.Field{
			{Name: "id", Type: "number", Required: true, Description: "数据源 ID"},
			{Name: "name", Type: "string", Required: false, Description: "名称"},
			{Name: "description", Type: "string", Required: false, Description: "说明"},
			{Name: "status", Type: "string", Required: false, Description: "active 或 disabled"},
			{Name: "config", Type: "object", Required: false, Description: "新配置，省略则不修改"},
		},
	}, c.Update)
	r.Protected(httpapi.Meta{
		Resource: "datasource", Action: "delete", Path: "/datasource/delete",
		Summary: "删除未被证书引用的数据源",
		Fields:  []httpapi.Field{{Name: "id", Type: "number", Required: true, Description: "数据源 ID"}},
	}, c.Delete)
	r.Protected(httpapi.Meta{
		Resource: "datasource", Action: "detail", Path: "/datasource/detail",
		Summary: "查看数据源，响应不包含账户私钥",
		Fields:  []httpapi.Field{{Name: "id", Type: "number", Required: true, Description: "数据源 ID"}},
	}, c.Detail)
	r.Protected(httpapi.Meta{
		Resource: "datasource", Action: "list", Path: "/datasource/list",
		Summary: "分页查询当前租户数据源",
		Fields: []httpapi.Field{
			{Name: "page", Type: "number", Required: false, Description: "页码"},
			{Name: "size", Type: "number", Required: false, Description: "每页条数"},
			{Name: "filters.type", Type: "string", Required: false, Description: "acme 或 nginx"},
			{Name: "filters.keyword", Type: "string", Required: false, Description: "名称关键字"},
			{Name: "filters.status", Type: "string", Required: false, Description: "状态"},
		},
	}, c.List)
	r.Protected(httpapi.Meta{
		Resource: "datasource", Action: "test", Path: "/datasource/test",
		Summary: "测试 ACME 目录连通性，或执行 nginx -t",
		Fields:  []httpapi.Field{{Name: "id", Type: "number", Required: true, Description: "数据源 ID"}},
	}, c.Test)
	r.Protected(httpapi.Meta{
		Resource: "datasource", Action: "register", Path: "/datasource/register",
		Summary: "向证书机构注册 ACME 账户，并保存账户地址",
		Fields: []httpapi.Field{
			{Name: "id", Type: "number", Required: true, Description: "ACME 数据源 ID"},
			{Name: "agreeTos", Type: "boolean", Required: true, Description: "是否同意 ACME 服务条款"},
		},
	}, c.Register)
	r.Protected(httpapi.Meta{
		Resource: "datasource", Action: "catalog", Path: "/datasource/catalog",
		Summary: "列出可注册的 ACME 证书机构和可对接的云厂商 DNS",
	}, c.Catalog)
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

func (c *Controller) Detail(ctx *gin.Context) {
	var req IDRequest
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

func (c *Controller) Register(ctx *gin.Context) {
	var req RegisterAccountRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	view, err := c.svc.Register(ctx.Request.Context(), tc.TenantID, req)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, view)
}

func (c *Controller) Catalog(ctx *gin.Context) {
	response.OK(ctx, acme.Catalog())
}

func (c *Controller) Test(ctx *gin.Context) {
	var req IDRequest
	if err := bind.JSON(ctx, &req); err != nil {
		response.FromError(ctx, err)
		return
	}
	tc, _ := tenant.From(ctx.Request.Context())
	view, err := c.svc.Test(ctx.Request.Context(), tc.TenantID, req.ID)
	if err != nil {
		response.FromError(ctx, err)
		return
	}
	response.OK(ctx, view)
}
