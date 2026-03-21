#!/bin/bash
# api2openclaw Docker 停止脚本

set -e

echo "🛑 停止 api2openclaw 服务..."

# 使用 docker compose (新版) 或 docker-compose (旧版)
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

# 停止并删除容器
$DOCKER_COMPOSE down

# 询问是否删除数据卷
read -p "是否删除数据卷？(包括数据库数据) [y/N]: " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "🗑️  删除数据卷..."
    $DOCKER_COMPOSE down -v
    echo "✅ 数据卷已删除"
else
    echo "📦 数据卷已保留"
fi

echo "✅ api2openclaw 服务已停止"
