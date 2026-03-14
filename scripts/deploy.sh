#!/bin/bash

# api2openclaw 部署脚本

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 项目配置
PROJECT_NAME="api2openclaw"
DOCKER_REGISTRY="${DOCKER_REGISTRY:-localhost:5000}"
IMAGE_NAME="${PROJECT_NAME}"
VERSION="${VERSION:-latest}"

# 辅助函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查 Docker 是否安装
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        log_error "Docker Compose is not installed"
        exit 1
    fi
}

# 构建镜像
build_image() {
    log_info "Building Docker image..."

    docker build \
        -f deployments/docker/Dockerfile \
        -t "${IMAGE_NAME}:${VERSION}" \
        -t "${IMAGE_NAME}:latest" \
        .

    log_info "Build completed: ${IMAGE_NAME}:${VERSION}"
}

# 推送镜像
push_image() {
    log_info "Pushing image to registry..."

    docker tag "${IMAGE_NAME}:${VERSION}" "${DOCKER_REGISTRY}/${IMAGE_NAME}:${VERSION}"
    docker push "${DOCKER_REGISTRY}/${IMAGE_NAME}:${VERSION}"

    log_info "Push completed"
}

# 启动服务
start_services() {
    log_info "Starting services..."

    cd deployments/docker

    # 使用 docker compose 或 docker-compose
    if docker compose version &> /dev/null; then
        docker compose up -d
    else
        docker-compose up -d
    fi

    log_info "Services started"
}

# 停止服务
stop_services() {
    log_info "Stopping services..."

    cd deployments/docker

    if docker compose version &> /dev/null; then
        docker compose down
    else
        docker-compose down
    fi

    log_info "Services stopped"
}

# 查看日志
view_logs() {
    local service="${1:-api2openclaw}"

    cd deployments/docker

    if docker compose version &> /dev/null; then
        docker compose logs -f "${service}"
    else
        docker-compose logs -f "${service}"
    fi
}

# 查看状态
status() {
    cd deployments/docker

    if docker compose version &> /dev/null; then
        docker compose ps
    else
        docker-compose ps
    fi
}

# 重启服务
restart() {
    log_info "Restarting services..."
    stop_services
    start_services
}

# 清理资源
clean() {
    log_info "Cleaning up..."

    cd deployments/docker

    if docker compose version &> /dev/null; then
        docker compose down -v
    else
        docker-compose down -v
    fi

    # 删除悬空镜像
    docker image prune -f

    log_info "Cleanup completed"
}

# 初始化数据库
init_db() {
    log_info "Initializing database..."

    cd deployments/docker

    # 等待 PostgreSQL 启动
    log_info "Waiting for PostgreSQL to be ready..."
    for i in {1..30}; do
        if docker compose exec -T postgres pg_isready -U api2openclaw &> /dev/null; then
            break
        fi
        echo -n "."
        sleep 1
    done
    echo

    # 运行迁移
    log_info "Running database migrations..."
    if docker compose exec -T postgres psql -U api2openclaw -d api2openclaw < /docker-entrypoint-initdb.d/000001_create_schema.sql &> /dev/null; then
        log_info "Schema created"
    fi

    log_info "Database initialized"
}

# 显示帮助
show_help() {
    cat << EOF
api2openclaw 部署脚本

用法: ./deploy.sh [命令] [选项]

命令:
  build              构建 Docker 镜像
  push               推送镜像到注册表
  start              启动服务
  stop               停止服务
  restart            重启服务
  logs [service]     查看日志 (默认: api2openclaw)
  status             查看服务状态
  init-db            初始化数据库
  clean              清理资源
  help               显示帮助信息

环境变量:
  VERSION            镜像版本 (默认: latest)
  DOCKER_REGISTRY    Docker 注册表 (默认: localhost:5000)

示例:
  ./deploy.sh build
  ./deploy.sh start
  ./deploy.sh logs api2openclaw
  VERSION=v1.0.0 ./deploy.sh build

EOF
}

# 主函数
main() {
    check_docker

    case "${1:-help}" in
        build)
            build_image
            ;;
        push)
            push_image
            ;;
        start)
            start_services
            ;;
        stop)
            stop_services
            ;;
        restart)
            restart
            ;;
        logs)
            view_logs "$2"
            ;;
        status)
            status
            ;;
        init-db)
            init_db
            ;;
        clean)
            clean
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "Unknown command: $1"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
