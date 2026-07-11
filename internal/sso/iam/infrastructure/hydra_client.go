// Package infrastructure 是 iam 上下文的基础设施层(Hydra 调用、GORM/Redis 仓储实现)。
package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	resty "github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

// HydraClient 封装 Ory Hydra v2 Admin API。
//
// 注意:v2 相比 v1 的关键差异是 Admin 端点统一挂在 /admin 前缀下
// (v1 为 /oauth2/auth/requests/login,v2 为 /admin/oauth2/auth/requests/login)。
type HydraClient struct {
	adminURL string
	client   *resty.Client
	log      *zap.Logger
}

// NewHydraClient 创建 Hydra Admin 客户端
func NewHydraClient(adminURL string, log *zap.Logger) *HydraClient {
	return &HydraClient{
		adminURL: strings.TrimRight(adminURL, "/"),
		client:   resty.New().SetTimeout(10 * time.Second),
		log:      log,
	}
}

// LoginRequest Hydra 登录请求信息
type LoginRequest struct {
	Challenge  string `json:"challenge"`
	Subject    string `json:"subject"`
	Skip       bool   `json:"skip"` // true 表示 Hydra 已有 login session,可静默 accept(单点登录关键)
	RequestURL string `json:"request_url"`
	Client     *struct {
		ClientID string `json:"client_id"`
	} `json:"client,omitempty"`
}

// ConsentRequest Hydra 授权同意请求信息
type ConsentRequest struct {
	Challenge      string   `json:"challenge"`
	Subject        string   `json:"subject"`
	RequestedScope []string `json:"requested_scope"`
	Skip           bool     `json:"skip"`
	Client         *struct {
		ClientID string `json:"client_id"`
	} `json:"client,omitempty"`
}

// LogoutRequest Hydra 登出请求信息
type LogoutRequest struct {
	Challenge string `json:"challenge"`
	Subject   string `json:"subject"`
}

// RedirectResponse Hydra accept/reject 后返回的跳转地址
type RedirectResponse struct {
	RedirectTo string `json:"redirect_to"`
}

// ErrLoginRequestGone challenge 已被消耗(浏览器后退场景),Hydra 返回 410 携带重定向地址。
type ErrLoginRequestGone struct{ RedirectTo string }

func (e *ErrLoginRequestGone) Error() string {
	return fmt.Sprintf("login_request already used, redirect_to: %s", e.RedirectTo)
}

// GetLoginRequest 获取登录请求。410 时返回 *ErrLoginRequestGone。
func (c *HydraClient) GetLoginRequest(ctx context.Context, challenge string) (*LoginRequest, error) {
	endpoint := fmt.Sprintf("%s/admin/oauth2/auth/requests/login?login_challenge=%s", c.adminURL, challenge)
	resp, err := c.client.R().SetContext(ctx).Get(endpoint)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusGone {
		var body struct {
			RedirectTo string `json:"redirect_to"`
		}
		_ = json.Unmarshal(resp.Body(), &body)
		return nil, &ErrLoginRequestGone{RedirectTo: body.RedirectTo}
	}
	if resp.IsError() {
		return nil, fmt.Errorf("hydra GET login_request: %d %s", resp.StatusCode(), resp.String())
	}
	var req LoginRequest
	if err := json.Unmarshal(resp.Body(), &req); err != nil {
		return nil, err
	}
	req.Challenge = challenge
	return &req, nil
}

// AcceptLoginRequest 接受登录请求,subject 为用户唯一标识(本项目用用户 ID)。
// remember=true 让 Hydra 记住登录会话,后续其他 client 可静默通过(单点登录)。
func (c *HydraClient) AcceptLoginRequest(ctx context.Context, challenge, subject string) (*RedirectResponse, error) {
	url := fmt.Sprintf("%s/admin/oauth2/auth/requests/login/accept?login_challenge=%s", c.adminURL, challenge)
	body := map[string]any{
		"subject":      subject,
		"remember":     true,
		"remember_for": 2592000, // 30 天
	}
	return c.putJSON(ctx, url, body)
}

// RejectLoginRequest 拒绝登录请求(如账号被禁用)。
func (c *HydraClient) RejectLoginRequest(ctx context.Context, challenge, reason string) (*RedirectResponse, error) {
	url := fmt.Sprintf("%s/admin/oauth2/auth/requests/login/reject?login_challenge=%s", c.adminURL, challenge)
	body := map[string]any{
		"error":             "access_denied",
		"error_description": reason,
	}
	return c.putJSON(ctx, url, body)
}

// GetConsentRequest 获取授权同意请求
func (c *HydraClient) GetConsentRequest(ctx context.Context, challenge string) (*ConsentRequest, error) {
	endpoint := fmt.Sprintf("%s/admin/oauth2/auth/requests/consent?consent_challenge=%s", c.adminURL, challenge)
	resp, err := c.client.R().SetContext(ctx).Get(endpoint)
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("hydra GET consent_request: %d %s", resp.StatusCode(), resp.String())
	}
	var req ConsentRequest
	if err := json.Unmarshal(resp.Body(), &req); err != nil {
		return nil, err
	}
	req.Challenge = challenge
	return &req, nil
}

// AcceptConsentRequest 接受授权同意。
// grant_access_token_audience 必须显式存在(哪怕空数组),否则部分 Hydra 版本会报错中断流程。
// idTokenExtra 会写入 id_token 的 claims:仅放 ASCII 安全字段(如 username),
// 避免中文写入 Hydra 存储触发编码错误;需要更多用户信息应走 /userinfo。
func (c *HydraClient) AcceptConsentRequest(ctx context.Context, challenge string, scopes []string, idTokenExtra map[string]any) (*RedirectResponse, error) {
	url := fmt.Sprintf("%s/admin/oauth2/auth/requests/consent/accept?consent_challenge=%s", c.adminURL, challenge)
	body := map[string]any{
		"grant_scope":                 scopes,
		"grant_access_token_audience": []string{},
		"remember":                    true,
		"remember_for":                2592000,
		"session": map[string]any{
			"id_token": idTokenExtra,
		},
	}
	return c.putJSON(ctx, url, body)
}

// GetLogoutRequest 获取登出请求
func (c *HydraClient) GetLogoutRequest(ctx context.Context, challenge string) (*LogoutRequest, error) {
	endpoint := fmt.Sprintf("%s/admin/oauth2/auth/requests/logout?logout_challenge=%s", c.adminURL, challenge)
	resp, err := c.client.R().SetContext(ctx).Get(endpoint)
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("hydra GET logout_request: %d %s", resp.StatusCode(), resp.String())
	}
	var req LogoutRequest
	if err := json.Unmarshal(resp.Body(), &req); err != nil {
		return nil, err
	}
	req.Challenge = challenge
	return &req, nil
}

// AcceptLogoutRequest 接受登出请求(自动 accept,用户无感)
func (c *HydraClient) AcceptLogoutRequest(ctx context.Context, challenge string) (*RedirectResponse, error) {
	url := fmt.Sprintf("%s/admin/oauth2/auth/requests/logout/accept?logout_challenge=%s", c.adminURL, challenge)
	return c.putJSON(ctx, url, struct{}{})
}

// DeleteLoginSession 删除某 subject 的 Hydra 登录会话(登出时调用)。
// v2 使用 query 参数 subject;404 视为幂等成功。
func (c *HydraClient) DeleteLoginSession(ctx context.Context, subject string) error {
	url := fmt.Sprintf("%s/admin/oauth2/auth/sessions/login?subject=%s", c.adminURL, subject)
	err := c.doJSON(ctx, http.MethodDelete, url, nil, nil)
	if errors.Is(err, ErrOAuth2ClientNotFound) {
		return nil
	}
	return err
}

// Ping 检查 Hydra Admin API 是否可达
func (c *HydraClient) Ping(ctx context.Context) error {
	resp, err := c.client.R().SetContext(ctx).Get(fmt.Sprintf("%s/health/alive", c.adminURL))
	if err != nil {
		return fmt.Errorf("hydra unreachable: %w", err)
	}
	if resp.IsError() {
		return fmt.Errorf("hydra health check failed: %d", resp.StatusCode())
	}
	return nil
}

func (c *HydraClient) putJSON(ctx context.Context, url string, body any) (*RedirectResponse, error) {
	resp, err := c.client.R().SetContext(ctx).SetBody(body).Put(url)
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("hydra PUT %s: %d %s", url, resp.StatusCode(), resp.String())
	}
	var result RedirectResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}
	if result.RedirectTo == "" {
		return nil, fmt.Errorf("hydra PUT %s: empty redirect_to", url)
	}
	return &result, nil
}
