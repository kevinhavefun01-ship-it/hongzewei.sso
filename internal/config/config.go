// Package config 负责从 config.yaml 加载全局配置(viper)。
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config 全局配置根
type Config struct {
	App   AppConfig   `mapstructure:"app"`
	SSO   SSOConfig   `mapstructure:"sso"`
	MySQL MySQLConfig `mapstructure:"mysql"`
	Redis RedisConfig `mapstructure:"redis"`
	Log   LogConfig   `mapstructure:"log"`
}

// AppConfig 应用级配置
type AppConfig struct {
	Name    string `mapstructure:"name"`
	BaseURL string `mapstructure:"base_url"`
	Addr    string `mapstructure:"addr"`
}

// SSOConfig SSO 子系统配置
type SSOConfig struct {
	Hydra HydraConfig `mapstructure:"hydra"`
	JWT   JWTConfig   `mapstructure:"jwt"`
}

// HydraConfig Ory Hydra 端点配置
type HydraConfig struct {
	AdminURL  string `mapstructure:"admin_url"`
	PublicURL string `mapstructure:"public_url"`
}

// JWTConfig 管理态 JWT 配置
type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Expire int    `mapstructure:"expire"`
}

// MySQLConfig 数据库配置
type MySQLConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	Charset  string `mapstructure:"charset"`
}

// DSN 拼接 GORM 使用的 MySQL DSN
func (m MySQLConfig) DSN() string {
	charset := m.Charset
	if charset == "" {
		charset = "utf8mb4"
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		m.User, m.Password, m.Host, m.Port, m.Database, charset)
}

// RedisConfig 缓存配置
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

// Load 从指定路径加载配置文件。path 为空时按默认查找 configs/config.yaml。
// 环境变量可覆盖配置文件中的值(12-Factor 风格,便于 Docker 部署)。
// 环境变量命名规则:前缀 HSW_ + 大写键名(点号/下划线分隔),如 HSW_MYSQL_HOST。
func Load(path string) (*Config, error) {
	v := viper.New()

	// ──── 环境变量支持 ────
	v.SetEnvPrefix("HSW")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("configs")
		v.AddConfigPath(".")
	}
	if err := v.ReadInConfig(); err != nil {
		// 配置文件不存在时,如果必要的环境变量已设置,则允许继续
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// 文件存在但解析失败,报错
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("解析配置文件失败: %w", err)
			}
		}
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}
	return &cfg, nil
}
