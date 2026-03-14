.PHONY: build run test clean docker docker-build docker-run deps migrate

BINARY_NAME=api2openclaw
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)'"

# 默认目标
all: deps build

# 安装依赖
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# 构建
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/api2openclaw/main.go

# 运行
run:
	@go run cmd/api2openclaw/main.go -config configs/config.yaml

# 测试
test:
	@echo "Running tests..."
	@go test -v -race -cover ./...

test-integration:
	@echo "Running integration tests..."
	@go test -v -tags=integration ./test/integration/...

# 清理
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean

# 数据库迁移
migrate-up:
	@echo "Running database migrations..."
	@go run cmd/migrate/main.go up

migrate-down:
	@echo "Rolling back database migrations..."
	@go run cmd/migrate/main.go down

# Docker
docker:
	@docker build -t api2openclaw:$(VERSION) -f deployments/docker/Dockerfile .

docker-build:
	@docker compose -f deployments/docker/docker-compose.yml build

docker-run:
	@docker compose -f deployments/docker/docker-compose.yml up -d

docker-down:
	@docker compose -f deployments/docker/docker-compose.yml down

docker-logs:
	@docker compose -f deployments/docker/docker-compose.yml logs -f

# 安装到系统
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

# 开发工具
dev-deps: deps
	@echo "Installing development tools..."
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 热重载开发
dev:
	@air -c .air.toml

# 代码检查
lint:
	@golangci-lint run ./...

# 格式化代码
fmt:
	@go fmt ./...
	@goimports -w .

# 帮助
help:
	@echo "Available targets:"
	@echo "  all              - Install deps and build (default)"
	@echo "  deps             - Install Go dependencies"
	@echo "  build            - Build the binary"
	@echo "  run              - Run the application"
	@echo "  test             - Run tests"
	@echo "  test-integration - Run integration tests"
	@echo "  clean            - Clean build artifacts"
	@echo "  migrate-up       - Run database migrations"
	@echo "  migrate-down     - Rollback migrations"
	@echo "  docker           - Build Docker image"
	@echo "  docker-build     - Build with docker-compose"
	@echo "  docker-run       - Run with docker-compose"
	@echo "  docker-down      - Stop docker-compose"
	@echo "  install          - Install to /usr/local/bin"
	@echo "  dev-deps         - Install development tools"
	@echo "  dev              - Run with hot reload"
	@echo "  lint             - Run linter"
	@echo "  fmt              - Format code"
	@echo "  help             - Show this help"
