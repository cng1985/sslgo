package domain

import (
	"context"

	"sslplatform/internal/platform/errors"
)

// Scope 是认证后写入请求上下文的可信身份。客户端提交的租户或用户 ID 不能进入这里。
type Scope struct {
	ActorID   uint
	UserID    uint
	TenantID  uint
	Username  string
	RequestID string
	System    bool
}

type scopeKey struct{}

// With 写入可信 Scope。ActorID 与 UserID 保持一致。
func With(ctx context.Context, scope Scope) context.Context {
	if scope.ActorID == 0 {
		scope.ActorID = scope.UserID
	}
	if scope.UserID == 0 {
		scope.UserID = scope.ActorID
	}
	return context.WithValue(ctx, scopeKey{}, scope)
}

// From 读取可信 Scope。普通请求必须有租户和用户；系统任务用 System 标记。
func From(ctx context.Context) (Scope, bool) {
	scope, ok := ctx.Value(scopeKey{}).(Scope)
	if !ok || scope.TenantID == 0 {
		return Scope{}, false
	}
	if scope.UserID == 0 && scope.ActorID == 0 && !scope.System {
		return Scope{}, false
	}
	return scope, true
}

// Must 读取业务调用必需的 Scope。缺失时返回 401。
func Must(ctx context.Context) (Scope, error) {
	scope, ok := From(ctx)
	if !ok {
		return Scope{}, apperror.New(401, "请先登录")
	}
	return scope, nil
}

type requestIDKey struct{}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}
