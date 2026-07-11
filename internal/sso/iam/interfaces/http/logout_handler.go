package http

import (
	"net/http"
	"strings"

	"hongzewei.sso/internal/sso/iam/infrastructure"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LogoutHandler 处理 Hydra logout_challenge(自动 accept,用户无感)。
type LogoutHandler struct {
	hydra    *infrastructure.HydraClient
	loginURL string
	log      *zap.Logger
}

// NewLogoutHandler 构造 logout handler
func NewLogoutHandler(hydra *infrastructure.HydraClient, loginURL string, log *zap.Logger) *LogoutHandler {
	return &LogoutHandler{hydra: hydra, loginURL: strings.TrimRight(loginURL, "/"), log: log}
}

// HandleLogout 处理 Hydra 登出回调。任何异常都回退到登录页,不向用户暴露细节。
func (h *LogoutHandler) HandleLogout(c *gin.Context) {
	challenge := c.Query("logout_challenge")
	if challenge == "" {
		c.Redirect(http.StatusFound, h.loginURL)
		return
	}
	redirect, err := h.hydra.AcceptLogoutRequest(c.Request.Context(), challenge)
	if err != nil {
		h.log.Warn("accept logout failed", zap.Error(err))
		c.Redirect(http.StatusFound, h.loginURL)
		return
	}
	c.Redirect(http.StatusFound, redirect.RedirectTo)
}
