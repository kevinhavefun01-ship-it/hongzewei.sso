package application

import (
	"context"
	"strconv"

	"hongzewei.sso/internal/shared/errcode"
	"hongzewei.sso/internal/sso/iam/domain"
	"hongzewei.sso/internal/sso/iam/infrastructure"

	"go.uber.org/zap"
)

// ConsentService Hydra 授权同意用例(MVP:自动放行)。
type ConsentService struct {
	hydra    *infrastructure.HydraClient
	userRepo domain.UserRepository
	log      *zap.Logger
}

// NewConsentService 构造 consent 服务
func NewConsentService(hydra *infrastructure.HydraClient, userRepo domain.UserRepository, log *zap.Logger) *ConsentService {
	return &ConsentService{hydra: hydra, userRepo: userRepo, log: log}
}

// HandleConsent 处理 consent 请求并自动 accept,返回 Hydra 给出的重定向地址。
func (s *ConsentService) HandleConsent(ctx context.Context, challenge string) (string, error) {
	req, err := s.hydra.GetConsentRequest(ctx, challenge)
	if err != nil {
		return "", errcode.New(SSOIAMHydraUnavailable, err)
	}

	userID, err := strconv.ParseInt(req.Subject, 10, 64)
	if err != nil {
		return "", errcode.New(SSOIAMConsentUserInvalid, err)
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return "", errcode.New(SSOIAMUserNotFound, err)
	}
	if ok, _ := user.CanLogin(); !ok {
		code := SSOIAMAccountDisabled
		if user.IsLocked {
			code = SSOIAMAccountLocked
		}
		s.log.Warn("consent blocked: user cannot login", zap.Int64("user_id", userID))
		return "", errcode.New(code, nil)
	}

	// id_token 仅放 ASCII 安全字段(username);中文等信息由子系统调用 /userinfo 获取,
	// 避免非 ASCII 写入 Hydra 存储引发编码问题。
	redirect, err := s.hydra.AcceptConsentRequest(ctx, challenge, req.RequestedScope, map[string]any{
		"username": user.Username,
	})
	if err != nil {
		s.log.Error("accept consent failed", zap.String("challenge", challenge), zap.Error(err))
		return "", errcode.New(SSOIAMHydraUnavailable, err)
	}
	return redirect.RedirectTo, nil
}
