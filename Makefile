# hzw.sso Makefile
# Windows 用户使用 Git Bash 或 mingw32-make

APP_NAME    := sso-server
BUILD_DIR   := bin
GO          := go
DOCKER      := docker

# ──── 开发 ────

.PHONY: build
build: ## 编译二进制
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server/

.PHONY: run
run: build ## 编译并启动服务
	$(BUILD_DIR)/$(APP_NAME) -config configs/config.yaml

.PHONY: test
test: ## 运行单元测试(-race 检测数据竞争)
	$(GO) test -race -count=1 ./...

.PHONY: test-cover
test-cover: ## 测试覆盖率
	$(GO) test -race -coverprofile=coverage.txt -covermode=atomic ./...
	$(GO) tool cover -func=coverage.txt

.PHONY: lint
lint: ## 代码质量检查(golangci-lint)
	golangci-lint run ./...

.PHONY: vet
vet: ## go vet 静态检查
	$(GO) vet ./...

# ──── Docker ────

.PHONY: docker-build
docker-build: ## 构建 Docker 镜像
	$(DOCKER) build -f docker/Dockerfile -t hzw-sso:latest .

.PHONY: docker-up
docker-up: ## docker compose 启动全部服务(MySQL + Hydra + SSO)
	cd docker && docker compose up -d

.PHONY: docker-down
docker-down: ## docker compose 停止全部服务
	cd docker && docker compose down

.PHONY: docker-logs
docker-logs: ## 查看容器日志
	cd docker && docker compose logs -f

# ─── 数据 ───

.PHONY: seed
seed: build ## 初始化种子数据(admin + demo client,需 MySQL 已启动)
	@echo "==> 等待 MySQL 就绪..."
	@sleep 5
	@./bin/sso-server install -config configs/config.yaml

.PHONY: dev-up
dev-up: docker-up seed ## 一键启动开发环境(依赖 + 种子数据)

# ──── 清理 ────

.PHONY: clean
clean: ## 清理构建产物
	rm -rf $(BUILD_DIR) coverage.txt server.exe *.out

# ──── 帮助 ────

.PHONY: help
help: ## 显示所有可用命令
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
