package application

import (
	"context"
	"errors"

	"hongzewei.sso/internal/shared/errcode"
	"hongzewei.sso/internal/sso/iam/domain"
	"hongzewei.sso/internal/sso/iam/infrastructure"

	"go.uber.org/zap"
)

// UserService 用户管理用例(管理员 CRUD + 自助改密)。
type UserService struct {
	userRepo domain.UserRepository
	log      *zap.Logger
}

// NewUserService 构造用户服务
func NewUserService(userRepo domain.UserRepository, log *zap.Logger) *UserService {
	return &UserService{userRepo: userRepo, log: log}
}

// CreateUserInput 创建用户入参
type CreateUserInput struct {
	Username string
	Password string
	RealName string
	IsAdmin  bool
}

// UpdateUserInput 更新用户入参(nil 字段表示不修改)
type UpdateUserInput struct {
	RealName *string
	IsActive *bool
	IsAdmin  *bool
}

// GetByID 按 ID 查询
func (s *UserService) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	u, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errcode.New(SSOIAMUserNotFound, err)
	}
	return u, nil
}

// List 分页列出用户
func (s *UserService) List(ctx context.Context, page, pageSize int) ([]*domain.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.userRepo.List(ctx, (page-1)*pageSize, pageSize)
}

// Create 创建用户(用户名唯一校验)
func (s *UserService) Create(ctx context.Context, in *CreateUserInput) (*domain.User, error) {
	if len(in.Password) < 6 {
		return nil, errcode.New(SSOIAMPasswordTooShort, nil)
	}
	if _, err := s.userRepo.FindByUsername(ctx, in.Username); err == nil {
		return nil, errcode.New(SSOIAMUsernameExists, nil)
	} else if !errors.Is(err, infrastructure.ErrUserNotFound) {
		return nil, errcode.New(SSOIAMUserCreateFailed, err)
	}
	u := &domain.User{Username: in.Username, RealName: in.RealName, IsActive: true, IsAdmin: in.IsAdmin}
	if err := u.SetPassword(in.Password); err != nil {
		return nil, errcode.New(SSOIAMUserCreateFailed, err)
	}
	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, errcode.New(SSOIAMUserCreateFailed, err)
	}
	return u, nil
}

// Update 更新用户资料/状态
func (s *UserService) Update(ctx context.Context, id int64, in *UpdateUserInput) error {
	u, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return errcode.New(SSOIAMUserNotFound, err)
	}
	if in.RealName != nil {
		u.RealName = *in.RealName
	}
	if in.IsActive != nil {
		u.IsActive = *in.IsActive
	}
	if in.IsAdmin != nil {
		u.IsAdmin = *in.IsAdmin
	}
	if err := s.userRepo.Update(ctx, u); err != nil {
		return errcode.New(SSOIAMUserUpdateFailed, err)
	}
	return nil
}

// Delete 删除用户(不能删除自己)
func (s *UserService) Delete(ctx context.Context, id, operatorID int64) error {
	if id == operatorID {
		return errcode.New(SSOIAMCannotDeleteSelf, nil)
	}
	if err := s.userRepo.Delete(ctx, id); err != nil {
		return errcode.New(SSOIAMUserUpdateFailed, err)
	}
	return nil
}

// ResetPassword 管理员重置他人密码,标记强制改密
func (s *UserService) ResetPassword(ctx context.Context, id int64, newPassword string) error {
	if len(newPassword) < 6 {
		return errcode.New(SSOIAMPasswordTooShort, nil)
	}
	u, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return errcode.New(SSOIAMUserNotFound, err)
	}
	if err := u.SetPassword(newPassword); err != nil {
		return errcode.New(SSOIAMUserUpdateFailed, err)
	}
	u.MustChangePassword = true
	return s.userRepo.Update(ctx, u)
}

// ChangeOwnPassword 用户自助改密(校验原密码)
func (s *UserService) ChangeOwnPassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
	if len(newPassword) < 6 {
		return errcode.New(SSOIAMPasswordTooShort, nil)
	}
	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return errcode.New(SSOIAMUserNotFound, err)
	}
	if !u.CheckPassword(oldPassword) {
		return errcode.New(SSOIAMOldPasswordWrong, nil)
	}
	if err := u.SetPassword(newPassword); err != nil {
		return errcode.New(SSOIAMUserUpdateFailed, err)
	}
	u.MarkPasswordChanged()
	return s.userRepo.Update(ctx, u)
}
