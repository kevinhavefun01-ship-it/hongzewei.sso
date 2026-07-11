package application

import (
	"context"
	"errors"
	"strconv"

	"hongzewei.sso/internal/shared/errcode"
	"hongzewei.sso/internal/shared/jwtx"
	"hongzewei.sso/internal/sso/iam/domain"
	"hongzewei.sso/internal/sso/iam/infrastructure"

	"go.uber.org/zap"
)

// AuthService 认证用例:密码登录、继续 OAuth 流程、登出。
type AuthService struct {
	userRepo  domain.UserRepository
	logRepo   domain.LoginLogRepository
	hydra     *infrastructure.HydraClient
	jwtSecret string
	jwtExpire int
	log       *zap.Logger
}

// NewAuthService 构造认证服务
func NewAuthService(
	userRepo domain.UserRepository,
	logRepo domain.LoginLogRepository,
	hydra *infrastructure.HydraClient,
	jwtSecret string,
	jwtExpire int,
	log *zap.Logger,
) *AuthService {
	return &AuthService{userRepo: userRepo, logRepo: logRepo, hydra: hydra, jwtSecret: jwtSecret, jwtExpire: jwtExpire, log: log}
}

// UserBrief 返回给前端的用户摘要(不含敏感字段)
type UserBrief struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	RealName string `json:"real_name"`
	Avatar   string `json:"avatar"`
	IsAdmin  bool   `json:"is_admin"`
}

// LoginResult 登录/继续授权的结果。
// RedirectTo 非空表示处于 OAuth2 流程中,前端应跳转该地址交还给 Hydra。
type LoginResult struct {
	Token              string     `json:"token,omitempty"`
	RedirectTo         string     `json:"redirect_to,omitempty"`
	MustChangePassword bool       `json:"must_change_password"`
	User               *UserBrief `json:"user,omitempty"`
}

// PasswordLogin 账号密码登录。loginChallenge 非空时会驱动 Hydra 接受登录并返回跳转地址。
func (s *AuthService) PasswordLogin(ctx context.Context, username, password, loginChallenge, ip, ua string) (*LoginResult, error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		s.saveLog(ctx, 0, username, ip, ua, false, "账号不存在")
		return nil, errcode.New(SSOIAMLoginFailed, err)
	}
	if !user.CheckPassword(password) {
		s.saveLog(ctx, user.ID, username, ip, ua, false, "密码错误")
		return nil, errcode.New(SSOIAMLoginFailed, nil)
	}
	if ok, reason := user.CanLogin(); !ok {
		code := SSOIAMAccountDisabled
		if user.IsLocked {
			code = SSOIAMAccountLocked
		}
		s.saveLog(ctx, user.ID, username, ip, ua, false, reason)
		return nil, errcode.New(code, nil)
	}

	// 更新登录信息 + 记成功日志
	user.RecordLogin(ip, "password")
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.log.Warn("update last login failed", zap.Int64("user_id", user.ID), zap.Error(err))
	}
	s.saveLog(ctx, user.ID, username, ip, ua, true, "")

	token, err := jwtx.Sign(s.jwtSecret, user.ID, user.Username, user.IsAdmin, s.jwtExpire)
	if err != nil {
		return nil, errcode.New(errcode.CommonInternal, err)
	}
	result := &LoginResult{Token: token, MustChangePassword: user.MustChangePassword, User: briefOf(user)}

	if loginChallenge != "" {
		redirectTo, err := s.acceptLogin(ctx, loginChallenge, user.ID)
		if err != nil {
			return nil, err
		}
		result.RedirectTo = redirectTo
	}
	return result, nil
}

// ContinueSSO 已登录用户(持管理态 JWT)遇到新的 login_challenge 时继续授权码流程。
func (s *AuthService) ContinueSSO(ctx context.Context, userID int64, loginChallenge, ip, ua string) (*LoginResult, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, errcode.New(SSOIAMUserNotFound, err)
	}
	if ok, reason := user.CanLogin(); !ok {
		code := SSOIAMAccountDisabled
		if user.IsLocked {
			code = SSOIAMAccountLocked
		}
		s.saveLog(ctx, user.ID, user.Username, ip, ua, false, reason)
		return nil, errcode.New(code, nil)
	}
	redirectTo, err := s.acceptLogin(ctx, loginChallenge, user.ID)
	if err != nil {
		return nil, err
	}
	return &LoginResult{RedirectTo: redirectTo}, nil
}

// Logout 删除用户在 Hydra 的登录会话(实现单点登出)。
func (s *AuthService) Logout(ctx context.Context, userID int64) error {
	if userID == 0 {
		return nil
	}
	return s.hydra.DeleteLoginSession(ctx, strconv.FormatInt(userID, 10))
}

// acceptLogin 调用 Hydra 接受登录请求;challenge 已被消耗时透传 Hydra 给出的重定向地址。
func (s *AuthService) acceptLogin(ctx context.Context, challenge string, userID int64) (string, error) {
	redirect, err := s.hydra.AcceptLoginRequest(ctx, challenge, strconv.FormatInt(userID, 10))
	if err != nil {
		var gone *infrastructure.ErrLoginRequestGone
		if errors.As(err, &gone) && gone.RedirectTo != "" {
			return gone.RedirectTo, nil
		}
		s.log.Warn("accept login request failed", zap.String("challenge", challenge), zap.Error(err))
		return "", errcode.New(SSOIAMLoginChallengeInvalid, err)
	}
	return redirect.RedirectTo, nil
}

func (s *AuthService) saveLog(ctx context.Context, userID int64, username, ip, ua string, success bool, reason string) {
	if len(ua) > 255 {
		ua = ua[:255]
	}
	if err := s.logRepo.Save(ctx, &domain.LoginLog{
		UserID:     userID,
		Username:   username,
		IP:         ip,
		UserAgent:  ua,
		Method:     "password",
		Success:    success,
		FailReason: reason,
	}); err != nil {
		s.log.Warn("save login log failed", zap.Error(err))
	}
}

func briefOf(u *domain.User) *UserBrief {
	return &UserBrief{ID: u.ID, Username: u.Username, RealName: u.RealName, Avatar: u.Avatar, IsAdmin: u.IsAdmin}
}
