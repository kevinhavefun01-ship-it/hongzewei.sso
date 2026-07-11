// Package contextx 提供跨层传递的请求级上下文读写(trace_id、当前登录用户等)。
// 统一 key 定义在此,避免各处裸用字符串 key 造成隐性耦合。
package contextx

import "context"

type ctxKey int

const (
	keyTraceID ctxKey = iota
	keyUserID
	keyUsername
	keyIsAdmin
)

// WithTraceID 写入 trace_id
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, keyTraceID, traceID)
}

// TraceID 读取 trace_id,不存在返回空串
func TraceID(ctx context.Context) string {
	v, _ := ctx.Value(keyTraceID).(string)
	return v
}

// WithUser 写入当前登录用户信息(鉴权中间件解析 JWT 后注入)
func WithUser(ctx context.Context, userID int64, username string, isAdmin bool) context.Context {
	ctx = context.WithValue(ctx, keyUserID, userID)
	ctx = context.WithValue(ctx, keyUsername, username)
	ctx = context.WithValue(ctx, keyIsAdmin, isAdmin)
	return ctx
}

// UserID 读取当前登录用户 ID,未登录返回 0
func UserID(ctx context.Context) int64 {
	v, _ := ctx.Value(keyUserID).(int64)
	return v
}

// Username 读取当前登录用户名
func Username(ctx context.Context) string {
	v, _ := ctx.Value(keyUsername).(string)
	return v
}

// IsAdmin 读取当前用户是否管理员
func IsAdmin(ctx context.Context) bool {
	v, _ := ctx.Value(keyIsAdmin).(bool)
	return v
}
