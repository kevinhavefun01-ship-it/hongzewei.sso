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

### 1. 克隆 & 配置

```bash
git clone https://github.com/<你的用户名>/hongzewei.sso.git
cd hongzewei.sso
cp configs/config.example.yaml configs/config.yaml
# 编辑 config.yaml,至少修改:
# - app.base_url: SSO 服务对外地址
# - sso.hydra.admin_url: Hydra Admin API 地址(内网)
# - sso.jwt.secret: 管理态 JWT 密钥(生产必须替换)
# - mysql / redis 连接信息
```

### 2. 启动依赖(Hydra + MySQL + Redis)

```bash
cd deploy
docker compose up -d
```

> `deploy/docker-compose.yml` 会拉起:
> - `hydra`:Ory Hydra v2 授权服务器(Admin API `:4445`,Public API `:4444`)
> - `hydra-migrate`:Hydra 数据库初始化
> - `mysql:5.7`:Hydra + hzw.sso 共享
> - `redis:7`:登录日志限流

### 3. 初始化种子数据

```bash
bash scripts/seed.sh
# 默认创建:
# - admin 用户(用户名:admin, 密码:admin123, 管理员)
# - demo-client OAuth Client(client_id:demo-app, client_secret:demo-secret)
```

### 4. 启动 SSO 服务

```bash
bash scripts/run.sh
# 或手动:
go run cmd/server/main.go
```

### 5. 验证

浏览器访问 `http://localhost:8080/login`,看到登录页即成功。

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
├── cmd/server/          # 入口
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
├── web/                 # 内嵌前端(go:embed)
├── deploy/              # docker-compose(Hydra + MySQL + Redis)
├── scripts/             # run/seed 脚本
├── configs/             # 配置样例
└── docs/                # 正式文档
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
| ORM | GORM |
| 缓存 | go-redis v9 |
| **授权服务器** | **Ory Hydra v2** |
| JWT | golang-jwt/jwt v5 |
| 配置 | viper |
| 日志 | zap + lumberjack |
| HTTP Client | resty v2 |

## 🤝 贡献

欢迎 Issue 和 PR。

## 📄 许可证

MIT License
