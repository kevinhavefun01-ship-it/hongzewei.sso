package infrastructure

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// 哨兵错误:便于上层做幂等处理
var (
	ErrOAuth2ClientAlreadyExists = errors.New("oauth2 client already exists")
	ErrOAuth2ClientNotFound      = errors.New("oauth2 client not found")
)

// OAuth2Client Hydra /admin/clients 资源
type OAuth2Client struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret,omitempty"`
	ClientName              string   `json:"client_name,omitempty"`
	RedirectURIs            []string `json:"redirect_uris"`
	PostLogoutRedirectURIs  []string `json:"post_logout_redirect_uris,omitempty"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	Scope                   string   `json:"scope,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
}

// CreateOAuth2Client POST /admin/clients。响应中一次性返回 client_secret 明文,之后不再回显。
func (c *HydraClient) CreateOAuth2Client(ctx context.Context, in *OAuth2Client) (*OAuth2Client, error) {
	var out OAuth2Client
	if err := c.doJSON(ctx, http.MethodPost, c.adminURL+"/admin/clients", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetOAuth2Client GET /admin/clients/{id}(secret 不回显)
func (c *HydraClient) GetOAuth2Client(ctx context.Context, clientID string) (*OAuth2Client, error) {
	var out OAuth2Client
	url := fmt.Sprintf("%s/admin/clients/%s", c.adminURL, clientID)
	if err := c.doJSON(ctx, http.MethodGet, url, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateOAuth2Client PUT /admin/clients/{id}(全量替换语义)
func (c *HydraClient) UpdateOAuth2Client(ctx context.Context, in *OAuth2Client) (*OAuth2Client, error) {
	var out OAuth2Client
	url := fmt.Sprintf("%s/admin/clients/%s", c.adminURL, in.ClientID)
	if err := c.doJSON(ctx, http.MethodPut, url, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteOAuth2Client DELETE /admin/clients/{id},404 视为已删除(幂等)
func (c *HydraClient) DeleteOAuth2Client(ctx context.Context, clientID string) error {
	url := fmt.Sprintf("%s/admin/clients/%s", c.adminURL, clientID)
	err := c.doJSON(ctx, http.MethodDelete, url, nil, nil)
	if errors.Is(err, ErrOAuth2ClientNotFound) {
		return nil
	}
	return err
}

// TokenIntrospection Hydra introspect 响应
type TokenIntrospection struct {
	Active   bool   `json:"active"`
	Sub      string `json:"sub"`
	ClientID string `json:"client_id"`
	Scope    string `json:"scope"`
}

// IntrospectToken POST /admin/oauth2/introspect,校验 access_token 并提取 sub。
func (c *HydraClient) IntrospectToken(ctx context.Context, token string) (*TokenIntrospection, error) {
	endpoint := c.adminURL + "/admin/oauth2/introspect"
	var result TokenIntrospection
	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetBody("token=" + url.QueryEscape(token)).
		SetResult(&result).
		Post(endpoint)
	if err != nil {
		return nil, fmt.Errorf("hydra introspect: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("hydra introspect: %d %s", resp.StatusCode(), resp.String())
	}
	return &result, nil
}

// BuildAuthorizeURL 构造 OAuth2 授权码流入口 URL(纯字符串拼接,不发请求)。
// 子应用/门户跳转此 URL,由 Hydra 发起 login_challenge 流程。
func BuildAuthorizeURL(publicURL, clientID, redirectURI, scope, state string) string {
	if scope == "" {
		scope = "openid offline profile email"
	}
	q := url.Values{}
	q.Set("client_id", clientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", scope)
	q.Set("state", state)
	return strings.TrimRight(publicURL, "/") + "/oauth2/auth?" + q.Encode()
}

// RandomState 生成 16 字节随机 state(OAuth2 CSRF 防御)
func RandomState() string {
	return randomHex(16)
}

// RandomClientSecret 生成 32 字节随机 client_secret
func RandomClientSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate client secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// doJSON 发送 JSON 请求;409/404 转换为哨兵错误便于幂等处理。
func (c *HydraClient) doJSON(ctx context.Context, method, url string, in, out any) error {
	req := c.client.R().SetContext(ctx).SetHeader("Accept", "application/json")
	if in != nil {
		req = req.SetBody(in)
	}
	resp, err := req.Execute(method, url)
	if err != nil {
		return fmt.Errorf("hydra %s %s: %w", method, url, err)
	}
	if resp.IsError() {
		switch resp.StatusCode() {
		case http.StatusConflict:
			return ErrOAuth2ClientAlreadyExists
		case http.StatusNotFound:
			return ErrOAuth2ClientNotFound
		}
		return fmt.Errorf("hydra %s %s: %d %s", method, url, resp.StatusCode(), resp.String())
	}
	if out == nil || resp.StatusCode() == http.StatusNoContent {
		return nil
	}
	if err := json.Unmarshal(resp.Body(), out); err != nil {
		return fmt.Errorf("hydra decode %s %s: %w", method, url, err)
	}
	return nil
}
