# hongzewei.sso

基于 **Ory Hydra v2** 的开源单点登录(SSO)系统,Go 实现。

> 一次登录,多处通行。

## ✨ 特性

- **单点登录**:用户在 A 应用登录后,访问 B 应用无需再次认证(Hydra login session)
- **OAuth2 / OIDC**:完整的授权码流程,支持 `openid`、`offline`、`profile`、`email` scope
- **单点登出**:登出时自动销毁所有应用的 Hydra 会话
- **零外部依赖**:不绑定第三方 IdP(钉钉/微信等),纯用户名密码认证
- **DDD 分层架构**:domain / application / infrastructure / interfaces,职责清晰,易于扩展
- **单二进制部署**:前端通过 `go:embed` 内嵌,编译后一个可执行文件即可运行
- **安全设计**:
  - bcrypt 密码哈希
  - JWT 管理态签发,服务端重签校验
  - trace_id 贯穿全链路,便于问题定位
  - 错误响应统一结构,避免敏感信息泄露

## 🏗️ 架构

```
┌─────────────────────────────────────────────────────────┐
│                    浏览器 / 子应用                        │
└──────────────┬──────────────────────────┬───────────────┘
               │                          │
               ▼                          ▼
┌──────────────────────────┐  ┌──────────────────────────┐
│      hzw.sso (本项目)     │  │    Ory Hydra v2           │
│  - 登录/授权页(web/)      │←→│  - OAuth2/OIDC 协议引擎   │
│  - IAM 领域层(internal/)  │  │  - 令牌签发/校验           │
│  - 用户/日志(GORM+MySQL) │  │  - JWKS 管理               │
└──────────────────────────┘  └──────────────────────────┘
```

**职责分离**:
- **Hydra**:只做协议引擎(授权码流、令牌签发),不存用户、不管密码
- **hzw.sso**:实现 Login/Consent Provider,负责「谁在登录」(用户认证 + 授权同意)

## 🚀 快速开始

### 方式一:Docker 一键启动(推荐)

只需 Docker,无需安装任何其他依赖:

```bash
git clone https://github.com/<你的用户名>/hongzewei.sso.git
cd hongzewei.sso

# 一键启动全部服务(MySQL + Hydra + SSO) + 初始化种子数据
make dev-up
```

> 默认端口:SSO `:8080` · Hydra Public `:4446` · Hydra Admin `:4447` · MySQL `:3307`
>
> 如需修改端口,直接编辑 `docker/docker-compose.yml` 对应的 ports 映射即可。

### 方式二:本地开发(需 Go 环境)

```bash
# 1. 起依赖(MySQL + Hydra)
make docker-up

# 2. 准备配置(复制 example 后按需修改)
cp configs/config.example.yaml configs/config.yaml

# 3. 编译并启动 SSO
make run

# 4. 另一个终端,初始化种子数据
make seed
```

### 方式三:下载 Release 二进制

前往 [Releases](https://github.com/<你的用户名>/hongzewei.sso/releases) 下载对应平台的预编译二进制,解压后:

```bash
# 1. 启动依赖(MySQL + Hydra)
docker compose up -d

# 2. 初始化数据库 + 种子数据
./sso-server install -config configs/config.example.yaml

# 3. 启动服务
./sso-server -config configs/config.example.yaml
```

### 验证

浏览器访问 `http://localhost:8080/login`,看到登录页即成功。

- 管理员账号:`admin` / `admin123`
- 管理后台:`http://localhost:8080/admin`
- demo OAuth Client:`demo-app` / `demo-secret`

## 📡 API

### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/sso/v1/auth/login` | 账号密码登录(可带 `login_challenge`) |
| GET  | `/api/sso/v1/hydra/consent` | Hydra consent 回调(自动放行) |
| GET  | `/api/sso/v1/hydra/logout` | Hydra logout 回调(自动 accept) |
| GET  | `/login` | 登录页(内嵌 HTML) |

### 需登录(SSO JWT)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET  | `/api/sso/v1/users/me` | 当前用户信息 |
| POST | `/api/sso/v1/auth/logout` | 登出(删 Hydra session) |
| POST | `/api/sso/v1/auth/change-password` | 改密 |

### 需管理员

| 方法 | 路径 | 说明 |
|------|------|------|
| GET  | `/api/sso/v1/admin/users` | 用户列表 |
| POST | `/api/sso/v1/admin/users` | 创建用户 |
| PUT  | `/api/sso/v1/admin/users/:id` | 更新用户 |
| DELETE | `/api/sso/v1/admin/users/:id` | 删除用户 |
| POST | `/api/sso/v1/admin/clients` | 注册 OAuth Client 到 Hydra |

## 🗂️ 项目结构

```
hongzewei.sso/
├── cmd/                 # 入口(server + seed_hash)
├── internal/
│   ├── config/          # 配置加载
│   ├── shared/          # 跨模块公共:response / errcode / middleware / jwtx
│   └── sso/
│       ├── app.go       # 路由装配
│       └── iam/         # IAM 限界上下文
│           ├── domain/          # 领域层(零依赖)
│           ├── application/     # 应用层(auth/consent/user service)
│           ├── infrastructure/  # 基础设施(Hydra client + GORM 仓储)
│           └── interfaces/http/ # 接口层(handler)
── web/                 # 内嵌前端(go:embed)
├── examples/            # Demo 子应用(app-a / app-b)
├├── docker/              # Docker 相关(Dockerfile + docker-compose)
├── migration/           # 数据库初始化脚本
├── configs/             # 配置样例 + Docker 配置
├── docs/                # 正式文档
├── Makefile             # 统一构建入口
├── Dockerfile           # 多阶段构建
├── .goreleaser.yml      # GitHub Releases 自动发版
├── .github/workflows/   # GitHub Actions CI
└── .golangci.yml        # 代码质量门禁
```

## 🛠️ Make 命令

> **Windows 用户**: 使用 `mingw32-make` 替代 `make`(已随 Git Bash / TDM-GCC 安装)

```bash
make build        # 编译二进制        (Windows: mingw32-make build)
make run          # 编译并启动
make test         # 单元测试(-race)
make lint         # 代码质量检查
make docker-up    # 启动依赖(MySQL + Hydra + SSO)
make docker-down  # 停止依赖
make seed         # 初始化种子数据
make dev-up       # 一键启动(依赖 + 种子)
make clean        # 清理构建产物
make help         # 显示所有命令
```

## 🔒 安全设计

- **双重鉴权**:所有管理接口后端必须校验 JWT,前端检查仅作辅助
- **错误码集中注册**:业务错误统一在 `errors.go` 定义,禁止硬编码文案
- **trace_id**:每个请求生成随机 trace_id,贯穿日志,问题可追溯
- **敏感信息隔离**:对外响应只暴露错误码 + 友好文案,底层错误(DB/网络)进日志不入响应

## 📦 技术栈

| 组件 | 选型 |
|------|------|
| 语言 | Go 1.26 |
| Web 框架 | Gin |
| ORM | GORM + MySQL 8.0 |
| 缓存 | go-redis v9(可选) |
| **授权服务器** | **Ory Hydra v2** |
| JWT | golang-jwt/jwt v5 |
| 配置 | viper |
| 日志 | zap + lumberjack |
| HTTP Client | resty v2 |
| 容器化 | Docker 多阶段构建 |
| CI | GitHub Actions + golangci-lint |

## 📦 发版(Releases)

本项目使用 [GoReleaser](https://goreleaser.com/) 自动构建多平台二进制并推送 GitHub Releases。

推送 tag 后 GitHub Actions 会自动触发:

```bash
git tag -a v0.1.0 -m "first release"
git push origin v0.1.0
```

发布产物包括:
- `sso-server_*_linux_amd64.tar.gz`
- `sso-server_*_linux_arm64.tar.gz`
- `sso-server_*_darwin_amd64.tar.gz`
- `sso-server_*_darwin_arm64.tar.gz`
- `sso-server_*_windows_amd64.zip`
- Docker 镜像自动推送

## 🤝 贡献

欢迎 Issue 和 PR。

## 📄 许可证

MIT License
