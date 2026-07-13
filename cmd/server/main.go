// Package main 是 SSO 服务的程序入口。
// 支持两个子命令:
//   - server (默认): 启动 HTTP 服务
//   - install:       初始化数据库 + 创建 admin 用户 + 注册 demo client
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"hongzewei.sso/internal/config"
	"hongzewei.sso/internal/sso/iam"
	"hongzewei.sso/internal/sso/iam/domain"
	"hongzewei.sso/internal/sso/iam/infrastructure"
	applogger "hongzewei.sso/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func main() {
	// ──── 解析子命令 ────
	cmd := "server"
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		cmd = os.Args[1]
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}

	// ─── 全局 help ────
	for _, arg := range os.Args {
		if arg == "-h" || arg == "--help" || arg == "-help" {
			printUsage()
			os.Exit(0)
		}
	}

	switch cmd {
	case "install":
		runInstall()
	case "server", "run", "":
		runServer()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`hzw.sso — 基于 Ory Hydra 的单点登录服务

用法:
  sso-server [命令] [选项]

命令:
  server    启动 HTTP 服务(默认)
  install   初始化数据库 + 创建 admin 用户 + 注册 demo client

示例:
  sso-server                                    # 启动服务
  sso-server -config configs/config.yaml        # 指定配置启动
  sso-server install                            # 初始化数据库和种子数据
  sso-server install -config configs/config.yaml
  sso-server install -admin-password mypass123  # 自定义 admin 密码`)
}

// ────────────────────────────────────────
// server: 启动 HTTP 服务
// ───────────────────────────────────────

func runServer() {
	configPath := flag.String("config", "", "配置文件路径,为空时使用默认查找策略")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	log, err := applogger.New(cfg.Log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	db, err := initDB(cfg, log)
	if err != nil {
		log.Fatal("初始化数据库失败", zap.Error(err))
	}

	router := sso.NewRouter(&sso.Deps{
		DB:     db,
		Config: cfg,
		Logger: log,
	})

	srv := &http.Server{
		Addr:         cfg.App.Addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("SSO 服务启动", zap.String("addr", cfg.App.Addr), zap.String("base_url", cfg.App.BaseURL))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP 服务异常退出", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("收到退出信号,开始优雅关闭...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("优雅关闭失败", zap.Error(err))
	}
	log.Info("服务已安全退出")
}

// ────────────────────────────────────────
// install: 初始化数据库 + 种子数据
// ────────────────────────────────────────

func runInstall() {
	configPath := flag.String("config", "", "配置文件路径")
	adminPwd := flag.String("admin-password", "admin123", "admin 用户初始密码")
	flag.Parse()

	fmt.Println("==> 加载配置...")
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	log, err := applogger.New(cfg.Log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	db, err := initDB(cfg, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化数据库失败: %v\n", err)
		os.Exit(1)
	}

	// ──── Step 1: 确保表结构 ────
	fmt.Println("==> 检查表结构(AutoMigrate)...")
	if err := db.AutoMigrate(&domain.User{}, &domain.LoginLog{}); err != nil {
		fmt.Fprintf(os.Stderr, "AutoMigrate 失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("    表结构就绪")

	// ──── Step 2: 创建 admin 用户 ────
	fmt.Printf("==> 创建 admin 用户(密码: %s)...\n", *adminPwd)
	var count int64
	db.Model(&domain.User{}).Where("username = ?", "admin").Count(&count)
	if count > 0 {
		fmt.Println("    admin 已存在,跳过")
	} else {
		admin := &domain.User{
			Username: "admin",
			RealName: "管理员",
			IsActive: true,
			IsAdmin:  true,
		}
		if err := admin.SetPassword(*adminPwd); err != nil {
			fmt.Fprintf(os.Stderr, "密码哈希失败: %v\n", err)
			os.Exit(1)
		}
		if err := db.Create(admin).Error; err != nil {
			fmt.Fprintf(os.Stderr, "创建 admin 失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("    admin 创建成功")
	}

	// ─── Step 3: 注册 demo client 到 Hydra(可选,Hydra 不可达时跳过) ────
	fmt.Println("==> 注册 demo-app OAuth Client...")
	hydra := infrastructure.NewHydraClient(cfg.SSO.Hydra.AdminURL, log)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := hydra.Ping(ctx); err != nil {
		fmt.Printf("     Hydra 不可达(%v),跳过 client 注册(可稍后手动注册)\n", err)
	} else {
		_, err := hydra.CreateOAuth2Client(context.Background(), &infrastructure.OAuth2Client{
			ClientID:     "demo-app",
			ClientSecret: randomHex(32),
			ClientName:   "Demo Application",
			RedirectURIs: []string{"http://localhost:5001/callback", "http://localhost:5002/callback"},
			GrantTypes:   []string{"authorization_code", "refresh_token"},
			ResponseTypes: []string{"code"},
			Scope:        "openid offline profile email",
			TokenEndpointAuthMethod: "client_secret_post",
		})
		if err != nil {
			if strings.Contains(err.Error(), "already exists") {
				fmt.Println("    demo-app 已存在,跳过")
			} else {
				fmt.Printf("    ⚠ 注册失败: %v\n", err)
			}
		} else {
			fmt.Println("    demo-app 注册成功")
		}
	}

	fmt.Println()
	fmt.Println("==> ✅ 初始化完成!")
	fmt.Println("    admin 账号: admin / " + *adminPwd)
	fmt.Println("    运行服务:   ./sso-server -config configs/config.yaml")
	fmt.Println("    访问:       http://localhost:8080/login")
}

// ────────────────────────────────────────
// 公共: 初始化 DB
// ────────────────────────────────────────

func initDB(cfg *config.Config, log *zap.Logger) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	}
	if cfg.Log.Level == "debug" {
		gormCfg.Logger = gormlogger.Default.LogMode(gormlogger.Info)
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN()), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("连接 MySQL 失败: %w", err)
	}

	log.Info("数据库连接成功", zap.String("database", cfg.MySQL.Database))
	return db, nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
