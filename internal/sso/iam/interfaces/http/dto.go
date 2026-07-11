// Package http 是 iam 上下文的接口层(Gin handler)。
package http

// LoginRequest 账号密码登录请求
type LoginRequest struct {
	Username       string `json:"username" binding:"required"`
	Password       string `json:"password" binding:"required"`
	LoginChallenge string `json:"login_challenge"` // 来自 OAuth2 流程时携带,可选
}

// SSOContinueRequest 已登录用户继续授权码流程
type SSOContinueRequest struct {
	LoginChallenge string `json:"login_challenge" binding:"required"`
}
