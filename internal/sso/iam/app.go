// Package sso 负责 iam 限界上下文的路由装配与依赖注入。
package sso

import (
	"net/http"

	"hongzewei.sso/internal/config"
	"hongzewei.sso/internal/shared/middleware"
	iamhttp "hongzewei.sso/internal/sso/iam/interfaces/http"
	"hongzewei.sso/internal/sso/iam/application"
	"hongzewei.sso/internal/sso/iam/infrastructure"
	"hongzewei.sso/web"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Deps 路由装配所需的外部依赖(由 main.go 组装后传入)
type Deps struct {
	DB      *gorm.DB
	Config  *config.Config
	Logger  *zap.Logger
}

// NewRouter 装配所有依赖、注册全部路由,返回可用的 Gin Engine。
func NewRouter(deps *Deps) *gin.Engine {
	cfg := deps.Config
	log := deps.Logger

	// ──── 基础设施层 ────
	hydra := infrastructure.NewHydraClient(cfg.SSO.Hydra.AdminURL, log)
	userRepo := infrastructure.NewUserRepository(deps.DB)
	logRepo := infrastructure.NewLoginLogRepository(deps.DB)

	// ──── 应用层 ────
	authSvc := application.NewAuthService(
		userRepo, logRepo, hydra,
		cfg.SSO.JWT.Secret, cfg.SSO.JWT.Expire, log,
	)
	consentSvc := application.NewConsentService(hydra, userRepo, log)
	userSvc := application.NewUserService(userRepo, log)

	// ──── 接口层 ────
	authH := iamhttp.NewAuthHandler(authSvc)
	consentH := iamhttp.NewConsentHandler(consentSvc, cfg.App.BaseURL)
	logoutH := iamhttp.NewLogoutHandler(hydra, cfg.App.BaseURL+"/login", log)
	userH := iamhttp.NewUserHandler(userSvc)
	adminUserH := iamhttp.NewAdminUserHandler(userSvc)
	adminClientH := iamhttp.NewAdminClientHandler(hydra)

	// ──── Gin Engine ────
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware.Trace(), middleware.CORS(), middleware.Recovery(log))

	// ──── 静态页面(go:embed) ────
	r.GET("/login", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", web.LoginHTML)
	})
	r.GET("/admin", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", web.AdminHTML)
	})

	// ──── API 路由 ────
	api := r.Group("/api/sso/v1")

	// 公开接口(无需登录)
	api.POST("/auth/login", authH.Login)
	api.GET("/hydra/consent", consentH.HandleConsent)
	api.GET("/hydra/logout", logoutH.HandleLogout)

	// 需登录
	authed := api.Group("")
	authed.Use(middleware.Auth(cfg.SSO.JWT.Secret))
	authH.RegisterAuthenticatedRoutes(authed)
	users := authed.Group("/users")
	userH.RegisterRoutes(users)

	// 需管理员
	admin := api.Group("/admin")
	admin.Use(middleware.AdminAuth(cfg.SSO.JWT.Secret))
	adminUserH.RegisterRoutes(admin)
	adminClientH.RegisterRoutes(admin)

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 兜底 404
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "路由不存在"})
	})

	return r
}
