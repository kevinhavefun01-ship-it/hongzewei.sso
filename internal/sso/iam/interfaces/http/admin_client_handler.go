package http

import (
	"hongzewei.sso/internal/shared/errcode"
	"hongzewei.sso/internal/shared/response"
	"hongzewei.sso/internal/sso/iam/infrastructure"

	"github.com/gin-gonic/gin"
)

// AdminClientHandler OAuth2 Client 管理(管理员)
type AdminClientHandler struct {
	hydra *infrastructure.HydraClient
}

// NewAdminClientHandler 构造客户端管理 handler
func NewAdminClientHandler(hydra *infrastructure.HydraClient) *AdminClientHandler {
	return &AdminClientHandler{hydra: hydra}
}

// RegisterRoutes 注册管理员客户端路由
func (h *AdminClientHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/clients", h.List)
	r.POST("/clients", h.Create)
	r.DELETE("/clients/:id", h.Delete)
}

// CreateClientRequest 创建客户端请求
type CreateClientRequest struct {
	ClientName              string   `json:"client_name" binding:"required"`
	RedirectURIs            []string `json:"redirect_uris" binding:"required"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	Scope                   string   `json:"scope"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// Create 注册新的 OAuth2 Client 到 Hydra(一次性返回 client_secret)
func (h *AdminClientHandler) Create(c *gin.Context) {
	var req CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	secret, err := infrastructure.RandomClientSecret()
	if err != nil {
		response.FailErr(c, err)
		return
	}
	client, err := h.hydra.CreateOAuth2Client(c.Request.Context(), &infrastructure.OAuth2Client{
		ClientSecret:            secret,
		ClientName:              req.ClientName,
		RedirectURIs:            req.RedirectURIs,
		GrantTypes:              nonEmptySlice(req.GrantTypes, []string{"authorization_code"}),
		ResponseTypes:           nonEmptySlice(req.ResponseTypes, []string{"code"}),
		Scope:                   nonEmptyStr(req.Scope, "openid offline profile email"),
		TokenEndpointAuthMethod: nonEmptyStr(req.TokenEndpointAuthMethod, "client_secret_post"),
	})
	if err != nil {
		response.FailErr(c, err)
		return
	}
	response.Success(c, client)
}

// List Hydra 不直接支持分页列举,此处留空占位(MVP 阶段通过 seed 脚本创建即可)
func (h *AdminClientHandler) List(c *gin.Context) {
	response.Success(c, []any{})
}

// Delete 删除 OAuth2 Client
func (h *AdminClientHandler) Delete(c *gin.Context) {
	clientID := c.Param("id")
	if clientID == "" {
		response.Fail(c, errcode.CommonInvalidParam)
		return
	}
	if err := h.hydra.DeleteOAuth2Client(c.Request.Context(), clientID); err != nil {
		response.FailErr(c, err)
		return
	}
	response.Success(c, nil)
}

func nonEmptyStr(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}

func nonEmptySlice(val, fallback []string) []string {
	if len(val) == 0 {
		return fallback
	}
	return val
}
