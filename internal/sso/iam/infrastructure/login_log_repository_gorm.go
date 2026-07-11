package infrastructure

import (
	"context"

	"hzw.sso/internal/sso/iam/domain"

	"gorm.io/gorm"
)

// LoginLogRepositoryGorm 基于 GORM 的登录日志仓储实现
type LoginLogRepositoryGorm struct {
	db *gorm.DB
}

// NewLoginLogRepository 创建登录日志仓储
func NewLoginLogRepository(db *gorm.DB) domain.LoginLogRepository {
	return &LoginLogRepositoryGorm{db: db}
}

func (r *LoginLogRepositoryGorm) Save(ctx context.Context, l *domain.LoginLog) error {
	return r.db.WithContext(ctx).Create(l).Error
}

func (r *LoginLogRepositoryGorm) List(ctx context.Context, offset, limit int) ([]*domain.LoginLog, int64, error) {
	var (
		logs  []*domain.LoginLog
		total int64
	)
	q := r.db.WithContext(ctx).Model(&domain.LoginLog{})
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
