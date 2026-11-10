package server

import (
	"github.com/gin-gonic/gin"

	"sslplatform/internal/appinfo"
	"sslplatform/internal/platform/config"
	"sslplatform/internal/platform/response"
	"sslplatform/internal/platform/routing"
)

func RegisterSystem(r *Registrar, cfg *config.Config) {
	r.Public(routing.Meta{
		Resource: "system", Action: "info", Path: "/system/info",
		Summary: "服务健康信息，不包含密钥",
	}, func(c *gin.Context) {
		response.OK(c, map[string]any{
			"name":     cfg.App.Name,
			"version":  appinfo.Version,
			"api":      "/api/v1",
			"method":   "POST",
			"acme":     cfg.ACME.DirectoryURL,
			"response": map[string]string{"code": "业务码，成功为 200", "msg": "说明", "data": "结果"},
		})
	})
	r.Protected(routing.Meta{
		Resource: "system", Action: "catalog", Path: "/system/catalog",
		Summary: "返回全部资源、动作和参数说明",
	}, func(c *gin.Context) {
		response.OK(c, map[string]any{
			"version": appinfo.Version,
			"rules": []string{
				"所有业务接口使用 POST 和 JSON",
				"成功响应 code 为 200",
				"分页响应包含 page、size、total、responseType=page",
				"租户来自登录态，请求体禁止出现 tenantId",
			},
			"routes": r.Routes(),
		})
	})
}
