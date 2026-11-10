package server

import (
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"sslplatform/internal/platform/config"
	"sslplatform/internal/platform/middleware"
	"sslplatform/internal/platform/routing"
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

// Registrar 把模块路由注册为 POST，并记录接口目录。
type Registrar struct {
	engine *gin.Engine
	v1     *gin.RouterGroup
	authn  gin.HandlerFunc
	authz  func(resource, action string) gin.HandlerFunc
	mu     sync.Mutex
	routes []routing.RouteInfo
}

func NewEngine(cfg *config.Config) *gin.Engine {
	switch cfg.Server.Mode {
	case "debug":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	_ = engine.SetTrustedProxies(cfg.Server.TrustedProxies)
	engine.Use(
		middleware.Recovery(),
		middleware.RequestID(),
		middleware.SecurityHeaders(),
		middleware.BodyLimit(cfg),
		middleware.RequestTimeout(cfg),
		middleware.GuardMethod(),
		middleware.RejectClientTenant(),
		middleware.AccessLog(),
	)
	return engine
}

func NewRegistrar(engine *gin.Engine, mw *middleware.Auth) *Registrar {
	return &Registrar{
		engine: engine,
		v1:     engine.Group("/api/v1"),
		authn:  mw.Authenticate(),
		authz:  mw.Require,
	}
}

func ProvideRouter(r *Registrar) routing.Router { return r }

func (r *Registrar) Public(meta routing.Meta, handlers ...gin.HandlerFunc) {
	r.mount("public", meta, handlers...)
}

func (r *Registrar) Login(meta routing.Meta, handlers ...gin.HandlerFunc) {
	all := append([]gin.HandlerFunc{r.authn}, handlers...)
	r.mount("login", meta, all...)
}

func (r *Registrar) Protected(meta routing.Meta, handlers ...gin.HandlerFunc) {
	all := append([]gin.HandlerFunc{r.authn, r.authz(meta.Resource, meta.Action)}, handlers...)
	r.mount("rbac", meta, all...)
}

func (r *Registrar) Routes() []routing.RouteInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]routing.RouteInfo, len(r.routes))
	copy(out, r.routes)
	return out
}

func (r *Registrar) mount(authMode string, meta routing.Meta, handlers ...gin.HandlerFunc) {
	path := meta.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	info := routing.RouteInfo{
		Method:   http.MethodPost,
		Path:     "/api/v1" + path,
		Resource: meta.Resource,
		Action:   meta.Action,
		Summary:  meta.Summary,
		Auth:     authMode,
		Fields:   meta.Fields,
	}
	if info.Fields == nil {
		info.Fields = []routing.Field{}
	}
	r.mu.Lock()
	r.routes = append(r.routes, info)
	r.mu.Unlock()
	r.v1.POST(path, handlers...)
}
