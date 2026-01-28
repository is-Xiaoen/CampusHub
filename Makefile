# Activity Platform Makefile
# 校园活动平台构建脚本

.PHONY: all build clean run test lint proto docker help

# 默认目标
all: build

# ==================== 变量定义 ====================
GO := go
GOFLAGS := -v
BUILD_DIR := ./bin
PROTO_DIR := ./app

# 服务列表
SERVICES := gateway user activity chat job

# ==================== 构建命令 ====================

# 构建所有服务
build: build-gateway build-user

# 构建 Gateway API
build-gateway:
	@echo "Building gateway-api..."
	@$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/gateway-api ./app/gateway/api/gateway.go

# 构建 User RPC
build-user:
	@echo "Building user-rpc..."
	@$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/user-rpc ./app/user/rpc/user.go

# 构建 Activity RPC (TODO)
build-activity:
	@echo "Building activity-rpc..."
	@$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/activity-rpc ./app/activity/rpc/activity.go

# 构建 Chat RPC (TODO)
build-chat:
	@echo "Building chat-rpc..."
	@$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/chat-rpc ./app/chat/rpc/chat.go

# 构建 Job Service
build-job:
	@echo "Building job service..."
	@$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/job ./app/job/job.go

# ==================== 运行命令 ====================

# 运行 Gateway API
run-gateway:
	@echo "Starting gateway-api..."
	@$(GO) run ./app/gateway/api/gateway.go -f ./app/gateway/api/etc/gateway.yaml

# 运行 User RPC
run-user:
	@echo "Starting user-rpc..."
	@$(GO) run ./app/user/rpc/user.go -f ./app/user/rpc/etc/user.yaml

# 运行 Job Service
run-job:
	@echo "Starting job service..."
	@$(GO) run ./app/job/job.go -f ./app/job/etc/job.yaml

# ==================== 依赖管理 ====================

# 下载依赖
deps:
	@echo "Downloading dependencies..."
	@$(GO) mod download

# 整理依赖
tidy:
	@echo "Tidying dependencies..."
	@$(GO) mod tidy

# 更新依赖
update:
	@echo "Updating dependencies..."
	@$(GO) get -u ./...
	@$(GO) mod tidy

# ==================== 代码质量 ====================

# 代码格式化
fmt:
	@echo "Formatting code..."
	@$(GO) fmt ./...
	@gofmt -s -w .

# 静态检查
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

# 代码检查（简化版）
vet:
	@echo "Running go vet..."
	@$(GO) vet ./...

# ==================== 测试 ====================

# 运行所有测试
test:
	@echo "Running tests..."
	@$(GO) test -v ./...

# 运行测试并生成覆盖率报告
test-coverage:
	@echo "Running tests with coverage..."
	@$(GO) test -v -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html

# 运行竞态检测
test-race:
	@echo "Running tests with race detector..."
	@$(GO) test -race -v ./...

# ==================== Proto 生成 ====================

# 生成所有 Proto 文件
proto: proto-user proto-activity proto-chat

# 生成 User Proto
proto-user:
	@echo "Generating user proto..."
	@protoc --go_out=. --go-grpc_out=. ./app/user/rpc/user.proto

# 生成 Activity Proto
proto-activity:
	@echo "Generating activity proto..."
	@protoc --go_out=. --go-grpc_out=. ./app/activity/rpc/activity.proto

# 生成 Chat Proto
proto-chat:
	@echo "Generating chat proto..."
	@protoc --go_out=. --go-grpc_out=. ./app/chat/rpc/chat.proto

# ==================== Docker ====================

# 启动开发环境（MySQL + Redis）
docker-up:
	@echo "Starting development environment..."
	@docker-compose -f deploy/docker/docker-compose.yaml up -d

# 停止开发环境
docker-down:
	@echo "Stopping development environment..."
	@docker-compose -f deploy/docker/docker-compose.yaml down

# 查看日志
docker-logs:
	@docker-compose -f deploy/docker/docker-compose.yaml logs -f

# 重建并启动
docker-rebuild:
	@echo "Rebuilding and starting..."
	@docker-compose -f deploy/docker/docker-compose.yaml up -d --build

# ==================== 清理 ====================

# 清理构建产物
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# ==================== 数据库 ====================

# 初始化数据库（需要先启动 Docker）
db-init:
	@echo "Initializing databases..."
	@docker exec -i activity-mysql mysql -uroot -proot123456 < deploy/sql/user.sql
	@docker exec -i activity-mysql mysql -uroot -proot123456 < deploy/sql/activity.sql
	@docker exec -i activity-mysql mysql -uroot -proot123456 < deploy/sql/chat.sql

# ==================== 帮助 ====================

help:
	@echo "Activity Platform Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  build           Build all services"
	@echo "  build-gateway   Build gateway API service"
	@echo "  build-user      Build user RPC service"
	@echo "  run-gateway     Run gateway API service"
	@echo "  run-user        Run user RPC service"
	@echo ""
	@echo "  deps            Download dependencies"
	@echo "  tidy            Tidy go.mod"
	@echo "  update          Update dependencies"
	@echo ""
	@echo "  fmt             Format code"
	@echo "  lint            Run linter"
	@echo "  vet             Run go vet"
	@echo ""
	@echo "  test            Run tests"
	@echo "  test-coverage   Run tests with coverage"
	@echo "  test-race       Run tests with race detector"
	@echo ""
	@echo "  proto           Generate all proto files"
	@echo "  proto-user      Generate user proto"
	@echo ""
	@echo "  docker-up       Start development environment"
	@echo "  docker-down     Stop development environment"
	@echo "  docker-logs     View Docker logs"
	@echo ""
	@echo "  db-init         Initialize databases"
	@echo "  clean           Clean build artifacts"
	@echo "  help            Show this help"
