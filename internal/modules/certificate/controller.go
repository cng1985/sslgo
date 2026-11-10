package certificate

import (
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
		Resource: "certificate", Action: "create", Path: "/certificate/create",
		Summary: "创建证书申请，并自动生成签发流水线",
		Fields: []routing.Field{
			{Name: "name", Type: "string", Required: true, Description: "证书名称"},
			{Name: "domains", Type: "array<string>", Required: true, Description: "域名列表。通配符只能在最左侧，并且必须使用 dns-01"},
			{Name: "email", Type: "string", Required: false, Description: "可选联系邮箱，默认使用 ACME 数据源邮箱"},
			{Name: "challengeType", Type: "string", Required: false, Description: "http-01 或 dns-01，默认 http-01。dns-01 使用 ACME 账户绑定的云厂商 DNS"},
			{Name: "keyType", Type: "string", Required: false, Description: "ec256 或 rsa2048，默认 ec256"},
			{Name: "acmeDataSourceId", Type: "number", Required: true, Description: "ACME 数据源 ID"},
			{Name: "nginxDataSourceId", Type: "number", Required: false, Description: "Nginx 数据源 ID。DNS-01 可以不填，此时只申请证书，不部署"},
			{Name: "renewBeforeDays", Type: "number", Required: false, Description: "到期前多少天续期，默认 30"},
			{Name: "autoRenew", Type: "boolean", Required: false, Description: "是否允许调度自动续期"},
			{Name: "agreeTos", Type: "boolean", Required: true, Description: "是否同意 ACME 服务条款"},
		},
	}, c.CreateCertificate)
	r.Protected(routing.Meta{
		Resource: "certificate", Action: "update", Path: "/certificate/update",
		Summary: "更新证书申请参数",
		Fields: []routing.Field{
			{Name: "id", Type: "number", Required: true, Description: "证书 ID"},
			{Name: "name", Type: "string", Required: true, Description: "证书名称"},
			{Name: "domains", Type: "array<string>", Required: true, Description: "域名列表。通配符必须使用 dns-01"},
			{Name: "email", Type: "string", Required: false, Description: "联系邮箱"},
			{Name: "challengeType", Type: "string", Required: false, Description: "http-01 或 dns-01"},
			{Name: "keyType", Type: "string", Required: false, Description: "ec256 或 rsa2048"},
			{Name: "acmeDataSourceId", Type: "number", Required: true, Description: "ACME 数据源 ID"},
			{Name: "nginxDataSourceId", Type: "number", Required: false, Description: "Nginx 数据源 ID。传 0 表示解除绑定，之后只能单独申请"},
			{Name: "renewBeforeDays", Type: "number", Required: false, Description: "续期窗口天数"},
			{Name: "autoRenew", Type: "boolean", Required: false, Description: "是否自动续期"},
			{Name: "agreeTos", Type: "boolean", Required: true, Description: "是否同意服务条款"},
		},
	}, c.UpdateCertificate)
	r.Protected(routing.Meta{
		Resource: "certificate", Action: "delete", Path: "/certificate/delete",
		Summary: "删除证书及其流水线和调度，不删除磁盘上的历史备份",
		Fields:  []routing.Field{{Name: "id", Type: "number", Required: true, Description: "证书 ID"}},
	}, c.DeleteCertificate)
	r.Protected(routing.Meta{
		Resource: "certificate", Action: "list", Path: "/certificate/list",
		Summary: "分页查询当前租户证书，返回到期时间 notAfter 和剩余天数 daysRemaining",
		Fields: []routing.Field{
			{Name: "page", Type: "number", Required: false, Description: "页码"},
			{Name: "size", Type: "number", Required: false, Description: "每页条数"},
			{Name: "filters.status", Type: "string", Required: false, Description: "draft、issuing、issued、deployed 或 failed"},
			{Name: "filters.keyword", Type: "string", Required: false, Description: "名称或域名关键字"},
		},
	}, c.ListCertificates)
	r.Protected(routing.Meta{
		Resource: "certificate", Action: "detail", Path: "/certificate/detail",
		Summary: "查看证书详情、指纹和公钥 PEM，不返回私钥",
		Fields:  []routing.Field{{Name: "id", Type: "number", Required: true, Description: "证书 ID"}},
	}, c.DetailCertificate)
	r.Protected(routing.Meta{
		Resource: "certificate", Action: "download", Path: "/certificate/download",
		Summary: "下载已签发证书。nginx 包内是域名.pem 与域名.key，通配符去掉 *.；httpd 为证书、私钥和证书链，iis 为带密码的 PFX",
		Fields: []routing.Field{
			{Name: "id", Type: "number", Required: true, Description: "证书 ID"},
			{Name: "format", Type: "string", Required: true, Description: "nginx、httpd 或 iis。apache 等同于 httpd"},
			{Name: "password", Type: "string", Required: false, Description: "iis 必填，8 到 64 位，作为 PFX 导入密码"},
		},
	}, c.Download)
	r.Protected(routing.Meta{
		Resource: "certificate", Action: "issue", Path: "/certificate/issue",
		Summary: "申请证书并部署：替换旧文件后重启 Nginx。证书必须已绑定 Nginx",
		Fields: []routing.Field{
			{Name: "id", Type: "number", Required: true, Description: "证书 ID"},
			{Name: "force", Type: "boolean", Required: false, Description: "为 true 时忽略续期窗口，立即重新签发"},
		},
	}, c.Issue)
	r.Protected(routing.Meta{
		Resource: "certificate", Action: "apply", Path: "/certificate/apply",
		Summary: "单独申请证书：向 ACME 签发并保存材料，不写 Nginx、不重启",
		Fields: []routing.Field{
			{Name: "id", Type: "number", Required: true, Description: "证书 ID"},
			{Name: "force", Type: "boolean", Required: false, Description: "为 true 时忽略续期窗口，立即重新申请"},
		},
	}, c.Apply)
	r.Protected(routing.Meta{
		Resource: "certificate", Action: "deploy", Path: "/certificate/deploy",
		Summary: "不重新申请。已保存的证书会写入 Nginx 后重启；只有磁盘文件时只重启",
		Fields:  []routing.Field{{Name: "id", Type: "number", Required: true, Description: "证书 ID"}},
	}, c.Deploy)

	r.Protected(routing.Meta{
		Resource: "etlJob", Action: "create", Path: "/etlJob/create",
		Summary: "为证书创建签发流水线",
		Fields: []routing.Field{
			{Name: "name", Type: "string", Required: false, Description: "流水线名称"},
			{Name: "certificateId", Type: "number", Required: true, Description: "证书 ID"},
		},
	}, c.CreateJob)
	r.Protected(routing.Meta{
		Resource: "etlJob", Action: "update", Path: "/etlJob/update",
		Summary: "更新流水线名称",
		Fields: []routing.Field{
			{Name: "id", Type: "number", Required: true, Description: "流水线 ID"},
			{Name: "name", Type: "string", Required: true, Description: "流水线名称"},
		},
	}, c.UpdateJob)
	r.Protected(routing.Meta{
		Resource: "etlJob", Action: "delete", Path: "/etlJob/delete",
		Summary: "删除流水线及其调度",
		Fields:  []routing.Field{{Name: "id", Type: "number", Required: true, Description: "流水线 ID"}},
	}, c.DeleteJob)
	r.Protected(routing.Meta{
		Resource: "etlJob", Action: "list", Path: "/etlJob/list",
		Summary: "分页查询签发流水线",
		Fields: []routing.Field{
			{Name: "page", Type: "number", Required: false, Description: "页码"},
			{Name: "size", Type: "number", Required: false, Description: "每页条数"},
			{Name: "filters.certificateId", Type: "number", Required: false, Description: "证书 ID"},
			{Name: "filters.status", Type: "string", Required: false, Description: "idle 或 running"},
		},
	}, c.ListJobs)
	r.Protected(routing.Meta{
		Resource: "etlJob", Action: "detail", Path: "/etlJob/detail",
		Summary: "查看流水线详情",
		Fields:  []routing.Field{{Name: "id", Type: "number", Required: true, Description: "流水线 ID"}},
	}, c.DetailJob)
	r.Protected(routing.Meta{
		Resource: "etlJob", Action: "execute", Path: "/etlJob/execute",
		Summary: "异步执行 Extract、Transform、Load 签发流水线",
		Fields: []routing.Field{
			{Name: "id", Type: "number", Required: true, Description: "流水线 ID"},
			{Name: "force", Type: "boolean", Required: false, Description: "是否强制重新签发"},
			{Name: "deploy", Type: "boolean", Required: false, Description: "为 false 时只申请证书。省略时：已绑定 Nginx 则部署，否则只申请"},
		},
	}, c.Execute)
	r.Protected(routing.Meta{
		Resource: "dashboard", Action: "summary", Path: "/dashboard/summary",
		Summary: "汇总证书、即将过期数量和运行中的任务",
	}, c.Summary)
}
