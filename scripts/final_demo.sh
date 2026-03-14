#!/bin/bash
# api2openclaw 完整功能演示

echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║                                                                 ║"
echo "║                    api2openclaw                                 ║"
echo "║              本地 LLM 网关与格式转换服务                          ║"
echo "║                                                                 ║"
echo "║                    🎉 项目完成!                                  ║"
echo "║                                                                 ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📍 服务状态"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 检查服务器
if curl -s http://localhost:8080/health > /dev/null; then
    echo -e "${GREEN}✓${NC} API 服务运行中 (http://localhost:8080)"
else
    echo -e "${YELLOW}⚠ API 服务未运行${NC}"
fi

# 检查数据库
if docker ps | grep -q api2openclaw-postgres; then
    echo -e "${GREEN}✓${NC} PostgreSQL 运行中"
else
    echo -e "${YELLOW}⚠ PostgreSQL 未运行${NC}"
fi
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🎯 核心功能演示"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

echo ""
echo -e "${BLUE}1. 健康检查${NC}"
echo "$ curl http://localhost:8080/health"
echo ""
curl -s http://localhost:8080/health | jq . 2>/dev/null || curl -s http://localhost:8080/health
echo ""

echo -e "${BLUE}2. 格式转换${NC}"
echo "输入: DeepSeek 格式 [{'type':'text','text':'Hello, World!'}]"
echo "输出: OpenClaw 格式"
echo ""
RESULT=$(curl -s -X POST http://localhost:8080/v1/convert \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "choices": [{
        "message": {
          "content": [{"type": "text", "text": "Hello from api2openclaw!"}]
        }
      }]
    }
  }')
echo "结果: $RESULT"
echo ""

echo -e "${BLUE}3. 后端状态${NC}"
docker exec api2openclaw-postgres psql -U api2openclaw -d api2openclaw -t -c "SELECT COUNT(*) FROM backends;" 2>/dev/null | xargs
echo "后端数量: $(curl -s http://localhost:8080/health | jq -r '.backends.total' 2>/dev/null)"
echo "健康后端: $(curl -s http://localhost:8080/health | jq -r '.backends.healthy' 2>/dev/null)"
echo ""

echo -e "${BLUE}4. 活跃请求${NC}"
ACTIVE=$(curl -s http://localhost:8080/health | jq -r '.active_requests' 2>/dev/null)
echo "当前活跃请求: $ACTIVE"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 系统架构"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "
                    ┌─────────────────┐
                    │   客户端请求    │
                    └────────┬────────┘
                             ↓
                    ┌─────────────────┐
                    │   认证网关      │ ← API Key 验证
                    │  (PostgreSQL)   │
                    └────────┬────────┘
                             ↓
                    ┌─────────────────┐
                    │   限流器         │ ← 速率限制
                    └────────┬────────┘
                             ↓
                    ┌─────────────────┐
                    │   模型路由器     │ ← 4种策略
                    │   + 熔断器        │
                    └────────┬────────┘
                             ↓
                    ┌─────────────────┐
                    │   后端转发器     │ ← 实际请求
                    └────────┬────────┘
                             ↓
                    ┌─────────────────┐
                    │   格式转换引擎   │ ← DeepSeek → OpenClaw
                    └────────┬────────┘
                             ↓
                    ┌─────────────────┐
                    │   本地 LLM 模型  │
                    │  (deepseek-v3)   │
                    └─────────────────┘
"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 已实现功能清单"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "
✅ 认证网关
   • 多租户管理
   • API Key 生成与验证
   • PostgreSQL 持久化
   • 权限检查

✅ 格式转换
   • DeepSeek 格式解析
   • OpenClaw 纯文本输出
   • 可配置模板
   • 流式输出支持

✅ 模型路由器
   • 4种路由策略
   • 自动健康检查
   • 故障转移
   • 模型别名配置

✅ 用量监控
   • 指标采集框架
   • 限流器（内存存储）
   • 熔断器
   • Prometheus 指标暴露

✅ HTTP API
   • RESTful 接口
   • 健康检查
   • 管理接口
   • 中间件支持

✅ 部署支持
   • Docker Compose
   • Systemd 服务
   • K8s 配置
"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📂 项目结构"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "
api2openclaw/
├── cmd/api2openclaw/         # 入口文件
├── internal/
│   ├── auth/                 # 认证网关 ✅
│   ├── converter/            # 格式转换 ✅
│   ├── router/               # 模型路由器 ✅
│   ├── monitor/              # 用量监控 ✅
│   ├── server/               # HTTP 服务器 ✅
│   ├── config/               # 配置系统 ✅
│   └── models/               # 数据模型 ✅
├── configs/                  # 配置文件
├── migrations/               # 数据库迁移
├── deployments/              # 部署配置
├── scripts/                  # 测试脚本
└── docs/                     # 文档
"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🚀 快速命令"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "
# 构建项目
cd /Users/liu/openclaw/api2openclaw
make build

# 运行服务
/tmp/api2openclaw --config configs/config.yaml

# 查看日志
tail -f /tmp/api2openclaw.log

# 运行测试
./scripts/test_full_api.sh
./scripts/test_stage3.sh

# 停止服务
pkill -f api2openclaw

# Docker 部署
cd deployments/docker
docker-compose up -d
"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📖 更多信息"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "
项目文档: /Users/liu/openclaw/api2openclaw/docs/
配置文件: /Users/liu/openclaw/api2openclaw/configs/
测试脚本: /Users/liu/openclaw/api2openclaw/scripts/

GitHub: https://github.com/openclaw/api2openclaw
"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}🎉 项目开发完成！感谢使用 api2openclaw！${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
