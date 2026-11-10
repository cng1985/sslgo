package middleware

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"sslplatform/internal/appinfo"
	"sslplatform/internal/platform/config"
	"sslplatform/internal/platform/domain"
	"sslplatform/internal/platform/response"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, rec any) {
		slog.Error("请求处理异常", "panic", rec, "path", c.Request.URL.Path, "request_id", domain.RequestID(c.Request.Context()))
		response.Fail(c, http.StatusInternalServerError, "内部错误")
	})
}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" || len(id) > 64 {
			id = newID()
		}
		c.Request = c.Request.WithContext(domain.WithRequestID(c.Request.Context(), id))
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Next()
	}
}

func BodyLimit(cfg *config.Config) gin.HandlerFunc {
	limit := cfg.Server.MaxBodyBytes
	if limit <= 0 {
		limit = 2 << 20
	}
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}

func RequestTimeout(cfg *config.Config) gin.HandlerFunc {
	timeout := cfg.Server.RequestTimeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health/live" || c.Request.URL.Path == "/health/ready" {
			c.Next()
			return
		}
		ctx, cancel := contextWithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func GuardMethod() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") && c.Request.Method != http.MethodPost {
			response.Fail(c, http.StatusMethodNotAllowed, "接口只接受 POST")
			c.Abort()
			return
		}
		c.Next()
	}
}

func RejectClientTenant() gin.HandlerFunc {
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

func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		fields := []any{
			"version", appinfo.Version,
			"request_id", domain.RequestID(c.Request.Context()),
			"method", c.Request.Method,
			"route", route,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
		}
		if scope, ok := domain.From(c.Request.Context()); ok {
			fields = append(fields, "actor_id", scope.UserID, "tenant_id", scope.TenantID)
		}
		slog.Info("请求完成", fields...)
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

func newID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "req"
	}
	return hex.EncodeToString(buf[:])
}
