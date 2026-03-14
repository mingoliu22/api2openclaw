#!/bin/bash
# api2openclaw 完整功能测试脚本

set -e

echo "=== api2openclaw 完整功能测试 ==="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试结果统计
PASS=0
FAIL=0

# 测试函数
test_case() {
    local name="$1"
    local command="$2"
    local expected="$3"

    echo -n "测试: $name ... "

    local result
    result=$(eval "$command" 2>&1)

    if echo "$result" | grep -q "$expected"; then
        echo -e "${GREEN}PASS${NC}"
        ((PASS++))
        return 0
    else
        echo -e "${RED}FAIL${NC}"
        echo "  期望包含: $expected"
        echo "  实际结果: $result"
        ((FAIL++))
        return 1
    fi
}

# 检查服务器是否运行
echo "1. 检查服务器状态"
if curl -s http://localhost:8080/health | grep -q "ok"; then
    echo -e "${GREEN}✓ 服务器运行正常${NC}"
else
    echo -e "${RED}✗ 服务器未运行${NC}"
    exit 1
fi
echo ""

# 测试健康检查
echo "2. 健康检查接口"
test_case \
    "健康检查返回 200" \
    "curl -s -w '%{http_code}' -o /dev/null http://localhost:8080/health" \
    "200"
echo ""

# 测试格式转换
echo "3. 格式转换功能"
echo "   输入: DeepSeek 格式 [{'type':'text','text':'Hello, World!'}]"
echo "   期望输出: Hello, World!"

RESULT=$(curl -s -X POST http://localhost:8080/v1/convert \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "choices": [{
        "message": {
          "role": "assistant",
          "content": [
            {"type": "text", "text": "Hello, World!"}
          ]
        },
        "finish_reason": "stop"
      }]
    }
  }')

if [ "$RESULT" = "Hello, World!" ]; then
    echo -e "${GREEN}✓ 格式转换正确${NC}"
    ((PASS++))
else
    echo -e "${RED}✗ 格式转换错误${NC}"
    echo "  期望: Hello, World!"
    echo "  实际: $RESULT"
    ((FAIL++))
fi
echo ""

# 测试复杂格式转换
echo "4. 复杂格式转换 (多个文本块)"
RESULT=$(curl -s -X POST http://localhost:8080/v1/convert \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "choices": [{
        "message": {
          "content": [
            {"type": "text", "text": "AI"},
            {"type": "text", "text": " "},
            {"type": "text", "text": "response"}
          ]
        }
      }]
    }
  }')

if [ "$RESULT" = "AI response" ]; then
    echo -e "${GREEN}✓ 多文本块转换正确${NC}"
    ((PASS++))
else
    echo -e "${RED}✗ 多文本块转换错误${NC}"
    echo "  期望: AI response"
    echo "  实际: $RESULT"
    ((FAIL++))
fi
echo ""

# 测试空内容处理
echo "5. 空内容处理"
RESULT=$(curl -s -X POST http://localhost:8080/v1/convert \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "choices": [{
        "message": {"content": []}
      }]
    }
  }')

if [ -z "$RESULT" ]; then
    echo -e "${GREEN}✓ 空内容处理正确${NC}"
    ((PASS++))
else
    echo -e "${YELLOW}⚠ 空内容返回: '$RESULT'${NC}"
    ((PASS++))
fi
echo ""

# 数据库验证
echo "6. 数据库验证"
if docker exec api2openclaw-postgres psql -U api2openclaw -d api2openclaw -c "SELECT COUNT(*) FROM tenants;" | grep -q "1"; then
    echo -e "${GREEN}✓ 默认租户存在${NC}"
    ((PASS++))
else
    echo -e "${RED}✗ 租户数据验证失败${NC}"
    ((FAIL++))
fi

if docker exec api2openclaw-postgres psql -U api2openclaw -d api2openclaw -c "SELECT COUNT(*) FROM api_keys;" | grep -q "1"; then
    echo -e "${GREEN}✓ 默认 API Key 存在${NC}"
    ((PASS++))
else
    echo -e "${RED}✗ API Key 数据验证失败${NC}"
    ((FAIL++))
fi
echo ""

# 显示数据库表信息
echo "7. 数据库表结构"
echo "   租户信息:"
docker exec api2openclaw-postgres psql -U api2openclaw -d api2openclaw -c "SELECT id, name, tier FROM tenants;" 2>/dev/null | sed 's/^/   /'
echo ""
echo "   API Keys:"
docker exec api2openclaw-postgres psql -U api2openclaw -d api2openclaw -c "SELECT id, tenant_id, status FROM api_keys;" 2>/dev/null | sed 's/^/   /'
echo ""

# 测试总结
echo "======================================"
echo "测试总结:"
echo "  通过: $PASS"
echo "  失败: $FAIL"
echo "======================================"

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}所有测试通过! ✓${NC}"
    exit 0
else
    echo -e "${RED}有 $FAIL 个测试失败${NC}"
    exit 1
fi
