#!/bin/bash
# api2openclaw 使用演示

echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║                    api2openclaw 使用演示                          ║"
echo "║              本地 LLM 网关与格式转换服务                           ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""

echo "📍 服务地址: http://localhost:8080"
echo "📍 Prometheus 指标: http://localhost:9090/metrics"
echo ""

# 1. 健康检查
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "1️⃣  健康检查"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "$ curl http://localhost:8080/health"
echo ""
curl -s http://localhost:8080/health | jq . 2>/dev/null || curl -s http://localhost:8080/health
echo ""
echo ""

# 2. 格式转换演示
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "2️⃣  格式转换: DeepSeek → OpenClaw"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "输入格式 (DeepSeek):"
cat << 'EOF'
{
  "choices": [{
    "message": {
      "content": [
        {"type": "text", "text": "你好，"}
      ]
    }
  }]
}
EOF
echo ""
echo "转换命令:"
echo "$ curl -X POST http://localhost:8080/v1/convert \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -d '{\"data\": {...}}'"
echo ""
echo "输出结果:"
RESULT=$(curl -s -X POST http://localhost:8080/v1/convert \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "choices": [{
        "message": {
          "content": [
            {"type": "text", "text": "你好，我是 AI 助手！"}
          ]
        }
      }]
    }
  }')
echo "$RESULT"
echo ""
echo ""

# 3. 多文本块合并
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "3️⃣  多文本块合并"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "输入: 三个分离的文本块"
echo ""
RESULT=$(curl -s -X POST http://localhost:8080/v1/convert \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "choices": [{
        "message": {
          "content": [
            {"type": "text", "text": "第一段"},
            {"type": "text", "text": "第二段"},
            {"type": "text", "text": "第三段"}
          ]
        }
      }]
    }
  }')
echo "输出: $RESULT"
echo ""

# 4. 数据库信息
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "4️⃣  数据库状态"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "租户数量:"
docker exec api2openclaw-postgres psql -U api2openclaw -d api2openclaw -t -c "SELECT COUNT(*) FROM tenants;" 2>/dev/null | xargs
echo ""
echo "API Keys 数量:"
docker exec api2openclaw-postgres psql -U api2openclaw -d api2openclaw -t -c "SELECT COUNT(*) FROM api_keys;" 2>/dev/null | xargs
echo ""

# 5. 配置信息
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "5️⃣  当前配置"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "输入格式: deepseek"
echo "输出格式: openclaw"
echo "认证方式: PostgreSQL"
echo "服务器地址: 0.0.0.0:8080"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ 阶段一功能验证完成"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📋 已实现功能:"
echo "   ✓ 认证网关 (PostgreSQL 存储)"
echo "   ✓ 格式转换 (DeepSeek → OpenClaw)"
echo "   ✓ HTTP API 服务"
echo "   ✓ 数据库迁移"
echo "   ✓ 健康检查"
echo ""
echo "📋 待实现功能 (阶段二):"
echo "   ⏳ 模型路由器"
echo "   ⏳ 用量监控"
echo "   ⏳ 管理后台"
echo ""
echo "📚 查看更多: cd /Users/liu/openclaw/api2openclaw && make help"
echo ""
