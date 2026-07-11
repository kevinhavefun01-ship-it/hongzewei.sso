// Package domain 是 iam 限界上下文的领域层,零框架依赖,只表达业务规则。
package domain

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User SSO 用户实体
type User struct {
	ID                 int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username           string     `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Password           string     `gorm:"size:255" json:"-"`
	RealName           string     `gorm:"size:50" json:"real_name"`
	Avatar             string     `gorm:"size:500;not null;default:''" json:"avatar"`
	IsActive           bool       `gorm:"default:true" json:"is_active"`
	IsAdmin            bool       `gorm:"default:false" json:"is_admin"`
	IsLocked           bool       `gorm:"default:false" json:"is_locked"`
	MustChangePassword bool       `gorm:"default:false" json:"must_change_password"`
	LockedAt           *time.Time `json:"locked_at,omitempty"`
	LastLoginAt        *time.Time `json:"last_login_at,omitempty"`
	LastLoginIP        string     `gorm:"size:45" json:"last_login_ip,omitempty"`
	LastLoginMethod    string     `gorm:"size:20" json:"last_login_method,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// TableName 指定表名(中性前缀 sso_)
func (*User) TableName() string { return "sso_users" }

// SetPassword 使用 bcrypt 加密并写入密码
func (u *User) SetPassword(plain string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)
	return nil
}

// CheckPassword 校验明文密码是否匹配
func (u *User) CheckPassword(plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(plain)) == nil
}

// CanLogin 判断账号当前是否允许登录
func (u *User) CanLogin() (bool, string) {
	if u.IsLocked {
		return false, "账号已被锁定,请联系管理员解锁"
	}
	if !u.IsActive {
		return false, "账号已被禁用"
	}
	return true, ""
}

// RecordLogin 记录本次登录的时间/IP/方式
func (u *User) RecordLogin(ip, method string) {
	now := time.Now()
	u.LastLoginAt = &now
	u.LastLoginIP = ip
	u.LastLoginMethod = method
}

// MarkPasswordChanged 用户主动改密后清除强制改密标记
func (u *User) MarkPasswordChanged() {
	u.MustChangePassword = false
}

// Lock 锁定账号
func (u *User) Lock() {
	now := time.Now()
	u.IsLocked = true
	u.LockedAt = &now
}

// Unlock 解锁账号
func (u *User) Unlock() {
	u.IsLocked = false
	u.LockedAt = nil
}
