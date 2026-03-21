#!/bin/bash
# api2openclaw Docker 启动脚本

set -e

echo "🚀 启动 api2openclaw 服务..."

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null; then
    echo "❌ 错误: 未找到 Docker，请先安装 Docker"
    exit 1
fi

if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "❌ 错误: 未找到 Docker Compose，请先安装 Docker Compose"
    exit 1
fi

# 使用 docker compose (新版) 或 docker-compose (旧版)
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

# 检查 .env 文件
if [ ! -f .env ]; then
    echo "📝 创建 .env 文件..."
    cp .env.example .env
    echo "✅ .env 文件已创建（使用默认配置）"
fi

# 构建并启动服务
echo "🔨 构建镜像..."
$DOCKER_COMPOSE build

echo "📦 启动服务..."
$DOCKER_COMPOSE up -d

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 5

# 检查服务状态
echo ""
echo "📊 服务状态:"
$DOCKER_COMPOSE ps

echo ""
echo "✅ api2openclaw 服务已启动！"
echo ""
echo "🌐 服务访问地址:"
echo "   - API 服务:     http://localhost:8080"
echo "   - API 文档:     http://localhost:8080/v1/docs"
echo "   - Prometheus:   http://localhost:9090/metrics"
echo "   - 前端 (开发):  http://localhost:5173 (需先启动: docker compose --profile frontend up -d)"
echo "   - pgAdmin:      http://localhost:5050 (需先启动: docker compose --profile admin up -d)"
echo ""
echo "📝 查看日志: $DOCKER_COMPOSE logs -f"
echo "🛑 停止服务: $DOCKER_COMPOSE down"
