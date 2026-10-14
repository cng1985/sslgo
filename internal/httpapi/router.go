package httpapi

import "github.com/gin-gonic/gin"

// Field 描述单个请求字段，供 AI Agent 理解参数语义。
type Field struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// Meta 是一条 POST 接口的资源、动作和参数说明。
type Meta struct {
	Resource string
	Action   string
	Path     string
	Summary  string
	Fields   []Field
}

// RouteInfo 是对外发布的接口目录项。
type RouteInfo struct {
	Method   string  `json:"method"`
	Path     string  `json:"path"`
	Resource string  `json:"resource"`
	Action   string  `json:"action"`
	Summary  string  `json:"summary"`
	Auth     string  `json:"auth"`
	Fields   []Field `json:"fields"`
}

// Router 约束所有业务模块只注册 POST JSON 接口。
type Router interface {
	Public(meta Meta, h ...gin.HandlerFunc)
	Login(meta Meta, h ...gin.HandlerFunc)
	Protected(meta Meta, h ...gin.HandlerFunc)
	Routes() []RouteInfo
}
