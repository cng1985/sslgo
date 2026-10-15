package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"sslplatform/internal/auth"
	"sslplatform/internal/common/response"
	"sslplatform/internal/config"
	"sslplatform/internal/httpapi"
)

// Endpoint 保存实际监听地址，便于测试发现随机端口。
type Endpoint struct {
	mu   sync.RWMutex
	Addr string
}

func NewEndpoint() *Endpoint { return &Endpoint{} }

func (e *Endpoint) Set(addr string) {
	e.mu.Lock()
	e.Addr = addr
	e.mu.Unlock()
}

func (e *Endpoint) Get() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.Addr
}

// Registrar 把模块路由注册为 POST，并记录 AI 可读的接口目录。
type Registrar struct {
	engine *gin.Engine
	v1     *gin.RouterGroup
	authn  gin.HandlerFunc
	authz  func(resource, action string) gin.HandlerFunc
	mu     sync.Mutex
	routes []httpapi.RouteInfo
}

func NewEngine(cfg *config.Config) *gin.Engine {
	if cfg.Server.Mode == "debug" {
		gin.SetMode(gin.DebugMode)
	} else if cfg.Server.Mode == "test" {
		gin.SetMode(gin.TestMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	_ = engine.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	engine.Use(recovery(), guardMethod(), rejectClientTenant())
	return engine
}

func NewRegistrar(engine *gin.Engine, mw *auth.Middleware) *Registrar {
	return &Registrar{
		engine: engine,
		v1:     engine.Group("/api/v1"),
		authn:  mw.Authenticate(),
		authz:  mw.Require,
	}
}

func ProvideRouter(r *Registrar) httpapi.Router { return r }

func (r *Registrar) Public(meta httpapi.Meta, handlers ...gin.HandlerFunc) {
	r.mount("public", meta, handlers...)
}

func (r *Registrar) Login(meta httpapi.Meta, handlers ...gin.HandlerFunc) {
	all := append([]gin.HandlerFunc{r.authn}, handlers...)
	r.mount("login", meta, all...)
}

func (r *Registrar) Protected(meta httpapi.Meta, handlers ...gin.HandlerFunc) {
	all := append([]gin.HandlerFunc{r.authn, r.authz(meta.Resource, meta.Action)}, handlers...)
	r.mount("rbac", meta, all...)
}

func (r *Registrar) Routes() []httpapi.RouteInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]httpapi.RouteInfo, len(r.routes))
	copy(out, r.routes)
	return out
}

func (r *Registrar) mount(authMode string, meta httpapi.Meta, handlers ...gin.HandlerFunc) {
	path := meta.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	info := httpapi.RouteInfo{
		Method:   http.MethodPost,
		Path:     "/api/v1" + path,
		Resource: meta.Resource,
		Action:   meta.Action,
		Summary:  meta.Summary,
		Auth:     authMode,
		Fields:   meta.Fields,
	}
	if info.Fields == nil {
		info.Fields = []httpapi.Field{}
	}
	r.mu.Lock()
	r.routes = append(r.routes, info)
	r.mu.Unlock()
	r.v1.POST(path, handlers...)
}

func recovery() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, rec any) {
		slog.Error("请求处理异常", "panic", rec, "path", c.Request.URL.Path)
		response.Fail(c, http.StatusInternalServerError, "内部错误")
	})
}

func guardMethod() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") && c.Request.Method != http.MethodPost {
			response.Fail(c, http.StatusMethodNotAllowed, "接口只接受 POST")
			c.Abort()
			return
		}
		c.Next()
	}
}

func rejectClientTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/api/") || c.Request.Body == nil {
			c.Next()
			return
		}
		raw, err := io.ReadAll(c.Request.Body)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "读取请求失败")
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(raw))
		if len(bytes.TrimSpace(raw)) == 0 {
			c.Next()
			return
		}
		var payload any
		if json.Unmarshal(raw, &payload) == nil && containsTenant(payload) {
			response.Fail(c, http.StatusBadRequest, "不允许客户端提交 tenantId")
			c.Abort()
			return
		}
		c.Next()
	}
}

func containsTenant(v any) bool {
	switch typed := v.(type) {
	case map[string]any:
		for key, child := range typed {
			if strings.EqualFold(key, "tenantId") || key == "tenant_id" {
				return true
			}
			if containsTenant(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if containsTenant(child) {
				return true
			}
		}
	}
	return false
}
