package middleware

import (
	"strings"

	"hongzewei.sso/internal/shared/contextx"
	"hongzewei.sso/internal/shared/errcode"
	"hongzewei.sso/internal/shared/jwtx"
	"hongzewei.sso/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// Auth 校验管理态 JWT,解析出用户信息注入 context。
// 安全红线:凡是需要登录态的接口都必须挂此中间件,禁止「只前端检查、后端裸奔」。
func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c)
		if token == "" {
			response.Fail(c, errcode.CommonTokenMissing)
			c.Abort()
			return
		}
		claims, err := jwtx.Parse(secret, token)
		if err != nil {
			response.Fail(c, errcode.CommonUnauthorized, err)
			c.Abort()
			return
		}
		ctx := contextx.WithUser(c.Request.Context(), claims.UserID, claims.Username, claims.IsAdmin)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// AdminAuth 在 Auth 基础上追加管理员校验。
// is_admin 来自服务端重新签名校验过的 JWT,前端无法伪造(篡改会导致签名失效)。
func AdminAuth(secret string) gin.HandlerFunc {
	authFn := Auth(secret)
	return func(c *gin.Context) {
		authFn(c)
		if c.IsAborted() {
			return
		}
		if !contextx.IsAdmin(c.Request.Context()) {
			response.Fail(c, errcode.CommonForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}

func extractBearer(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(h)
}
