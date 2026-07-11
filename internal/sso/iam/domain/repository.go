package domain

import (
	"context"
	"time"
)

// UserRepository 用户仓储接口(实现在 infrastructure 层)
type UserRepository interface {
	FindByID(ctx context.Context, id int64) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	Create(ctx context.Context, u *User) error
	Update(ctx context.Context, u *User) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, offset, limit int) ([]*User, int64, error)
}

// LoginLog 登录日志实体
type LoginLog struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int64     `gorm:"index" json:"user_id"`
	Username   string    `gorm:"size:50" json:"username"`
	IP         string    `gorm:"size:45" json:"ip"`
	UserAgent  string    `gorm:"size:255" json:"user_agent"`
	Method     string    `gorm:"size:20" json:"method"`
	Success    bool      `json:"success"`
	FailReason string    `gorm:"size:100" json:"fail_reason,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// TableName 指定表名
func (LoginLog) TableName() string { return "sso_login_logs" }

// LoginLogRepository 登录日志仓储接口
type LoginLogRepository interface {
	Save(ctx context.Context, l *LoginLog) error
	List(ctx context.Context, offset, limit int) ([]*LoginLog, int64, error)
}
