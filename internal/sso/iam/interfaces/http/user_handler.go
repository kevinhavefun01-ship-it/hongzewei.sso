package http

import (
	"hongzewei.sso/internal/shared/contextx"
	"hongzewei.sso/internal/shared/response"
	"hongzewei.sso/internal/sso/iam/application"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户自助接口(需登录态,非管理员)
type UserHandler struct {
	userSvc *application.UserService
}

// NewUserHandler 构造用户 handler
func NewUserHandler(userSvc *application.UserService) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

// RegisterRoutes 注册需要登录态的用户路由
func (h *UserHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/me", h.Me)
	r.POST("/change-password", h.ChangePassword)
}

// Me 返回当前登录用户信息
func (h *UserHandler) Me(c *gin.Context) {
	userID := contextx.UserID(c.Request.Context())
	u, err := h.userSvc.GetByID(c.Request.Context(), userID)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.Success(c, u)
}

// ChangePasswordRequest 改密请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ChangePassword 用户自助改密
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	userID := contextx.UserID(c.Request.Context())
	if err := h.userSvc.ChangeOwnPassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		response.FailErr(c, err)
		return
	}
	response.Success(c, nil)
}
