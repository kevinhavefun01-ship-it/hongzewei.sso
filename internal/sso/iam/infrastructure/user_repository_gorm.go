package infrastructure

import (
	"context"
	"errors"

	"hzw.sso/internal/sso/iam/domain"

	"gorm.io/gorm"
)

// UserRepositoryGorm 基于 GORM 的用户仓储实现
type UserRepositoryGorm struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓储
func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &UserRepositoryGorm{db: db}
}

// ErrUserNotFound 用户不存在(供 application 层判定)
var ErrUserNotFound = errors.New("user not found")

func (r *UserRepositoryGorm) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User
	if err := r.db.WithContext(ctx).First(&u, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepositoryGorm) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	var u domain.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepositoryGorm) Create(ctx context.Context, u *domain.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *UserRepositoryGorm) Update(ctx context.Context, u *domain.User) error {
	return r.db.WithContext(ctx).Save(u).Error
}

func (r *UserRepositoryGorm) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&domain.User{}, id).Error
}

func (r *UserRepositoryGorm) List(ctx context.Context, offset, limit int) ([]*domain.User, int64, error) {
	var (
		users []*domain.User
		total int64
	)
	q := r.db.WithContext(ctx).Model(&domain.User{})
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}
