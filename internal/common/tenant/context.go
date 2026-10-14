package tenant

import "context"

// Context 是服务端从登录态解析出的租户上下文。客户端不允许提交租户标识。
type Context struct {
	TenantID uint
	UserID   uint
	Username string
}

type ctxKey struct{}

func With(ctx context.Context, c Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, c)
}

func From(ctx context.Context) (Context, bool) {
	v, ok := ctx.Value(ctxKey{}).(Context)
	if !ok || v.TenantID == 0 || v.UserID == 0 {
		return Context{}, false
	}
	return v, true
}
