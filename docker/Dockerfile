# ──── 阶段一:构建 ────
# 生产环境建议: golang:1.26-alpine + alpine:3.20
FROM golang:1.23.6-alpine AS builder

# 安装 ca-certificates(HTTPS 根证书)和 tzdata(时区数据)
RUN apk add --no-cache ca-certificates tzdata git

WORKDIR /src

# GOTOOLCHAIN=auto 允许 Go 自动下载 go.mod 指定的版本
# GOPROXY 使用中国镜像(解决国内网络问题)
ENV GOTOOLCHAIN=auto
ENV GOPROXY=https://goproxy.cn,direct

# 先拷贝依赖文件,利用 Docker 缓存(go mod download 只在依赖变化时重新执行)
COPY go.mod go.sum ./
RUN go mod download

# 拷贝源码并构建
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /sso-server ./cmd/server/

# ──── 阶段二:运行 ────
# 生产环境建议: FROM alpine:3.20
FROM golang:1.23.6-alpine AS runtime

RUN apk add --no-cache ca-certificates tzdata curl \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

# 非 root 用户运行(安全最佳实践)
RUN addgroup -S sso && adduser -S sso -G sso
USER sso

WORKDIR /app

# 从构建阶段拷贝二进制 + 默认配置
COPY --from=builder /sso-server .
COPY --from=builder /src/configs/config.example.yaml ./configs/config.yaml

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD curl -sf http://localhost:8080/health || exit 1

ENTRYPOINT ["./sso-server"]
