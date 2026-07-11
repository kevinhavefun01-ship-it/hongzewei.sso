package http

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"hzw.sso/internal/shared/contextx"
	"hzw.sso/internal/shared/errcode"
	"hzw.sso/internal/sso/iam/application"

	"github.com/gin-gonic/gin"
)

// ConsentHandler Hydra 授权同意 handler
type ConsentHandler struct {
	consentSvc *application.ConsentService
	ssoBaseURL string // 出错时重定向回登录页的基地址
}

// NewConsentHandler 构造 consent handler
func NewConsentHandler(consentSvc *application.ConsentService, ssoBaseURL string) *ConsentHandler {
	return &ConsentHandler{consentSvc: consentSvc, ssoBaseURL: strings.TrimRight(ssoBaseURL, "/")}
}

// HandleConsent 处理 Hydra consent 请求(浏览器 GET 跳转过来)。
// 成功 302 到 Hydra 给出的地址;失败 302 回登录页并带错误码。
func (h *ConsentHandler) HandleConsent(c *gin.Context) {
	challenge := c.Query("consent_challenge")
	if challenge == "" {
		h.redirectErr(c, int(application.SSOIAMConsentChallengeMissing))
		return
	}
	redirectTo, err := h.consentSvc.HandleConsent(c.Request.Context(), challenge)
	if err != nil {
		var ae *errcode.Error
		if errors.As(err, &ae) {
			h.redirectErr(c, int(ae.Code))
		} else {
			h.redirectErr(c, int(errcode.CommonInternal))
		}
		return
	}
	c.Redirect(http.StatusFound, redirectTo)
}

func (h *ConsentHandler) redirectErr(c *gin.Context, code int) {
	trace := contextx.TraceID(c.Request.Context())
	c.Redirect(http.StatusFound, fmt.Sprintf("%s/login?err=%s&trace=%s", h.ssoBaseURL, strconv.Itoa(code), trace))
}
