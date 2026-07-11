package http

import (
	"hzw.sso/internal/shared/contextx"
	"hzw.sso/internal/shared/errcode"
	"hzw.sso/internal/shared/response"
	"hzw.sso/internal/sso/iam/application"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证相关 HTTP handler
type AuthHandler struct {
	authSvc *application.AuthService
}

// NewAuthHandler 构造认证 handler
func NewAuthHandler(authSvc *application.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// RegisterAuthenticatedRoutes 注册需要登录态的认证路由
func (h *AuthHandler) RegisterAuthenticatedRoutes(r *gin.RouterGroup) {
	r.POST("/logout", h.Logout)
	r.POST("/sso-continue", h.SSOContinue)
}

// Login 账号密码登录(公开)。返回管理态 token,若处于 OAuth 流程则返回 redirect_to。
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	result, err := h.authSvc.PasswordLogin(
		c.Request.Context(),
		req.Username, req.Password, req.LoginChallenge,
		c.ClientIP(), c.GetHeader("User-Agent"),
	)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.Success(c, result)
}

// SSOContinue 已登录用户继续 OAuth2 授权码流程(用于 Hydra skip 场景)。
func (h *AuthHandler) SSOContinue(c *gin.Context) {
	var req SSOContinueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	userID := contextx.UserID(c.Request.Context())
	if userID == 0 {
		response.Fail(c, errcode.CommonTokenMissing)
		return
	}
	result, err := h.authSvc.ContinueSSO(c.Request.Context(), userID, req.LoginChallenge, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.Success(c, result)
}

// Logout 清除用户的 Hydra 登录会话(单点登出)。
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := contextx.UserID(c.Request.Context())
	if err := h.authSvc.Logout(c.Request.Context(), userID); err != nil {
		response.FailErr(c, err)
		return
	}
	response.Success(c, nil)
}
