package auth

// PermissionCatalog 是系统内全部资源动作，接口和授权都以此为准。
func PermissionCatalog() []Permission {
	items := []struct {
		Resource string
		Action   string
		Name     string
	}{
		{"user", "list", "查看用户"},
		{"user", "create", "创建用户"},
		{"user", "update", "更新用户"},
		{"user", "delete", "删除用户"},
		{"user", "assignRole", "分配用户角色"},
		{"role", "list", "查看角色"},
		{"role", "create", "创建角色"},
		{"role", "update", "更新角色"},
		{"role", "detail", "查看角色详情"},
		{"role", "grant", "授予角色权限"},
		{"role", "permissionList", "查看权限目录"},
		{"datasource", "list", "查看数据源"},
		{"datasource", "detail", "查看数据源详情"},
		{"datasource", "create", "创建数据源"},
		{"datasource", "update", "更新数据源"},
		{"datasource", "delete", "删除数据源"},
		{"datasource", "test", "测试数据源"},
		{"certificate", "list", "查看证书"},
		{"certificate", "detail", "查看证书详情"},
		{"certificate", "create", "创建证书"},
		{"certificate", "update", "更新证书"},
		{"certificate", "delete", "删除证书"},
		{"certificate", "issue", "签发证书"},
		{"certificate", "deploy", "部署证书并重启 Nginx"},
		{"etlJob", "list", "查看签发流水线"},
		{"etlJob", "detail", "查看流水线详情"},
		{"etlJob", "create", "创建流水线"},
		{"etlJob", "update", "更新流水线"},
		{"etlJob", "delete", "删除流水线"},
		{"etlJob", "execute", "执行流水线"},
		{"task", "list", "查看任务"},
		{"task", "detail", "查看任务详情"},
		{"schedule", "list", "查看调度"},
		{"schedule", "create", "创建调度"},
		{"schedule", "update", "更新调度"},
		{"schedule", "delete", "删除调度"},
		{"schedule", "enable", "启用调度"},
		{"schedule", "disable", "停用调度"},
		{"query", "search", "通用搜索"},
		{"query", "list", "通用列表"},
		{"query", "one", "通用单条查询"},
		{"dashboard", "summary", "查看总览"},
		{"system", "catalog", "查看接口目录"},
	}
	out := make([]Permission, 0, len(items))
	for _, item := range items {
		out = append(out, Permission{Resource: item.Resource, Action: item.Action, Name: item.Name})
	}
	return out
}

// BuiltinGrants 返回内置角色的权限键。管理员为全部权限。
func BuiltinGrants(code string) map[string]struct{} {
	all := map[string]struct{}{}
	for _, p := range PermissionCatalog() {
		all[p.Key()] = struct{}{}
	}
	switch code {
	case RoleAdmin:
		return all
	case RoleOperator:
		return pick(all,
			"datasource:list", "datasource:detail", "datasource:create", "datasource:update", "datasource:delete", "datasource:test",
			"certificate:list", "certificate:detail", "certificate:create", "certificate:update", "certificate:delete", "certificate:issue", "certificate:deploy",
			"etlJob:list", "etlJob:detail", "etlJob:create", "etlJob:update", "etlJob:delete", "etlJob:execute",
			"task:list", "task:detail",
			"schedule:list", "schedule:create", "schedule:update", "schedule:delete", "schedule:enable", "schedule:disable",
			"query:search", "query:list", "query:one",
			"dashboard:summary", "system:catalog",
		)
	case RoleViewer:
		return pick(all,
			"user:list", "role:list", "role:detail",
			"datasource:list", "datasource:detail",
			"certificate:list", "certificate:detail",
			"etlJob:list", "etlJob:detail",
			"task:list", "task:detail",
			"schedule:list",
			"query:search", "query:list", "query:one",
			"dashboard:summary", "system:catalog",
		)
	default:
		return map[string]struct{}{}
	}
}

func pick(all map[string]struct{}, keys ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if _, ok := all[key]; ok {
			out[key] = struct{}{}
		}
	}
	return out
}

func isBuiltIn(code string) bool {
	return code == RoleAdmin || code == RoleOperator || code == RoleViewer
}
