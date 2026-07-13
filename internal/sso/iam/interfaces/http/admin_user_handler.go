package http

import (
	"strconv"

	"hongzewei.sso/internal/shared/contextx"
	"hongzewei.sso/internal/shared/errcode"
	"hongzewei.sso/internal/shared/response"
	"hongzewei.sso/internal/sso/iam/application"

	"github.com/gin-gonic/gin"
)

// AdminUserHandler 管理员用户 CRUD
type AdminUserHandler struct {
	userSvc *application.UserService
}

// NewAdminUserHandler 构造管理员用户 handler
func NewAdminUserHandler(userSvc *application.UserService) *AdminUserHandler {
	return &AdminUserHandler{userSvc: userSvc}
}

// RegisterRoutes 注册管理员路由
func (h *AdminUserHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/users", h.List)
	r.POST("/users", h.Create)
	r.PUT("/users/:id", h.Update)
	r.DELETE("/users/:id", h.Delete)
	r.POST("/users/:id/reset-password", h.ResetPassword)
	r.GET("/login-logs", h.LoginLogs)
}

// List 分页列出用户
func (h *AdminUserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	users, total, err := h.userSvc.List(c.Request.Context(), page, pageSize)
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.SuccessPage(c, users, total, page, pageSize)
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	RealName string `json:"real_name"`
	IsAdmin  bool   `json:"is_admin"`
}

// Create 创建用户
func (h *AdminUserHandler) Create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	u, err := h.userSvc.Create(c.Request.Context(), &application.CreateUserInput{
		Username: req.Username,
		Password: req.Password,
		RealName: req.RealName,
		IsAdmin:  req.IsAdmin,
	})
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.Success(c, u)
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	RealName *string `json:"real_name"`
	IsActive *bool   `json:"is_active"`
	IsAdmin  *bool   `json:"is_admin"`
}

// Update 更新用户
func (h *AdminUserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.CommonInvalidParam)
		return
	}
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	if err := h.userSvc.Update(c.Request.Context(), id, &application.UpdateUserInput{
		RealName: req.RealName,
		IsActive: req.IsActive,
		IsAdmin:  req.IsAdmin,
	}); err != nil {
		response.FailErr(c, err)
		return
	}
	response.Success(c, nil)
}

// Delete 删除用户
func (h *AdminUserHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.CommonInvalidParam)
		return
	}
	operatorID := contextx.UserID(c.Request.Context())
	if err := h.userSvc.Delete(c.Request.Context(), id, operatorID); err != nil {
		response.FailErr(c, err)
		return
	}
	response.Success(c, nil)
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required"`
}

// ResetPassword 管理员重置他人密码
func (h *AdminUserHandler) ResetPassword(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, errcode.CommonInvalidParam)
		return
	}
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	if err := h.userSvc.ResetPassword(c.Request.Context(), id, req.NewPassword); err != nil {
		response.FailErr(c, err)
		return
	}
	response.Success(c, nil)
}

// LoginLogs 分页查询登录日志(MVP:直接透传仓储)
func (h *AdminUserHandler) LoginLogs(c *gin.Context) {
	// MVP 阶段暂未暴露 logRepo,此处返回空列表占位;
	// 后续可独立出 AdminLoginLogHandler 承接。
	response.SuccessPage(c, []any{}, 0, 1, 20)
}
