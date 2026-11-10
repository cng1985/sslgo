package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"sslplatform/internal/platform/domain"
	"sslplatform/internal/platform/response"
)

// Identity 是登录态和权限检查，由 auth.Resource 实现。
type Identity interface {
	AuthenticateToken(ctx context.Context, token string) (domain.Scope, error)
	Allowed(ctx context.Context, userID uint, resource, action string) (bool, error)
}

// Auth 把令牌和权限检查接到 Gin。
type Auth struct {
	resource Identity
	mu       sync.Mutex
	hits     map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewAuth(resource Identity) *Auth {
	return &Auth{
		resource: resource,
		hits:     map[string][]time.Time{},
		limit:    30,
		window:   time.Minute,
	}
}

func (m *Auth) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			response.Fail(c, http.StatusUnauthorized, "请先登录")
			c.Abort()
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		scope, err := m.resource.AuthenticateToken(c.Request.Context(), token)
		if err != nil {
			response.HandleError(c, err)
			c.Abort()
			return
		}
		scope.RequestID = domain.RequestID(c.Request.Context())
		c.Request = c.Request.WithContext(domain.With(c.Request.Context(), scope))
		c.Next()
	}
}

func (m *Auth) Require(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, ok := domain.From(c.Request.Context())
		if !ok {
			response.Fail(c, http.StatusUnauthorized, "请先登录")
			c.Abort()
			return
		}
		allowed, err := m.resource.Allowed(c.Request.Context(), scope.UserID, resource, action)
		if err != nil {
			response.HandleError(c, err)
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

func (m *Auth) LimitLogin() gin.HandlerFunc {
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
