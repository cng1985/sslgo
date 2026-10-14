package auth

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"sslplatform/internal/common/response"
	"sslplatform/internal/common/tenant"
)

type Middleware struct {
	svc    *Service
	mu     sync.Mutex
	hits   map[string][]time.Time
	limit  int
	window time.Duration
}

func NewMiddleware(svc *Service) *Middleware {
	return &Middleware{
		svc:    svc,
		hits:   map[string][]time.Time{},
		limit:  30,
		window: time.Minute,
	}
}

func (m *Middleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			response.Fail(c, http.StatusUnauthorized, "请先登录")
			c.Abort()
			return
		}
		parsed, err := m.svc.parse(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
		if err != nil {
			response.FromError(c, err)
			c.Abort()
			return
		}
		user, err := m.svc.repo.GetUser(c.Request.Context(), parsed.TenantID, parsed.UserID)
		if err != nil || user.Status != StatusActive {
			response.Fail(c, http.StatusUnauthorized, "登录态无效")
			c.Abort()
			return
		}
		tc := tenant.Context{TenantID: parsed.TenantID, UserID: parsed.UserID, Username: parsed.Username}
		c.Request = c.Request.WithContext(tenant.With(c.Request.Context(), tc))
		c.Next()
	}
}

func (m *Middleware) Require(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tc, ok := tenant.From(c.Request.Context())
		if !ok {
			response.Fail(c, http.StatusUnauthorized, "请先登录")
			c.Abort()
			return
		}
		allowed, err := m.svc.Allowed(c.Request.Context(), tc.UserID, resource, action)
		if err != nil {
			response.FromError(c, err)
			c.Abort()
			return
		}
		if !allowed {
			response.Fail(c, http.StatusForbidden, "没有权限执行该操作")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (m *Middleware) LimitLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()
		m.mu.Lock()
		defer m.mu.Unlock()
		hits := m.hits[ip]
		kept := hits[:0]
		for _, ts := range hits {
			if now.Sub(ts) < m.window {
				kept = append(kept, ts)
			}
		}
		if len(kept) >= m.limit {
			m.hits[ip] = kept
			response.Fail(c, http.StatusTooManyRequests, "登录尝试过于频繁")
			c.Abort()
			return
		}
		m.hits[ip] = append(kept, now)
		c.Next()
	}
}
