// Package jwtx 封装管理态 JWT 的签发与校验(HS256)。
//
// 说明:本项目的 OAuth2/OIDC 令牌由 Ory Hydra 负责签发与校验;
// 这里的 JWT 仅用于「SSO 自身管理后台」的登录态(用户在 SSO 登录成功后签发),
// 替代参照项目中较复杂的 sso-admin OAuth 回环,保持开源版自洽、无外部依赖。
package jwtx

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken token 无效或过期
var ErrInvalidToken = errors.New("invalid token")

// Claims 管理态 JWT 载荷
type Claims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

// Sign 签发 token。expireSeconds<=0 时默认 1 天。
func Sign(secret string, userID int64, username string, isAdmin bool, expireSeconds int) (string, error) {
	if expireSeconds <= 0 {
		expireSeconds = 86400
	}
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		IsAdmin:  isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expireSeconds) * time.Second)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// Parse 校验并解析 token
func Parse(secret, tokenStr string) (*Claims, error) {
	var claims Claims
	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return &claims, nil
}
