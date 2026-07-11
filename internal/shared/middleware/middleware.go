// Package middleware 提供 Gin 通用中间件:trace_id 注入、跨域、panic 恢复。
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"hongzewei.sso/internal/shared/contextx"
	"hongzewei.sso/internal/shared/errcode"
	"hongzewei.sso/internal/shared/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Trace 为每个请求生成 trace_id 并注入 context 与响应头,便于问题定位。
func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		tid := genTraceID()
		ctx := contextx.WithTraceID(c.Request.Context(), tid)
		c.Request = c.Request.WithContext(ctx)
		c.Header("X-Trace-Id", tid)
		c.Next()
	}
}

// CORS 宽松跨域策略(便于本地联调/演示;生产应收敛 Allow-Origin)。
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin,Content-Type,Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// Recovery 捕获 panic,记录日志并返回统一错误结构。
func Recovery(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered",
					zap.Any("error", r),
					zap.String("path", c.Request.URL.Path),
					zap.String("trace_id", contextx.TraceID(c.Request.Context())),
				)
				response.Fail(c, errcode.CommonInternal)
				c.Abort()
			}
		}()
		c.Next()
	}
}

func genTraceID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "trace-unknown"
	}
	return hex.EncodeToString(b)
}
