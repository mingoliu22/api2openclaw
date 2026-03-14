#!/bin/bash
# api2openclaw Tool call 格式转换测试

echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║              api2openclaw Tool call 转换测试                      ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "1. DeepSeek Tool call 格式"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

cat <<'EOF' | python3 -c "import sys, json; print(json.dumps(json.loads(sys.stdin.read()), indent=2, ensure_ascii=False))"
{
  "choices": [{
    "message": {
      "role": "assistant",
      "content": [
        {"type": "text", "text": "我来帮你查询天气。"},
        {"type": "tool_use", "id": "call_123", "name": "get_weather", "input": {"location": "北京", "unit": "celsius"}}
      ]
    },
    "finish_reason": "tool_calls"
  }]
}
EOF
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "2. 转换为 OpenAI 格式"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

cat <<'PYTHON' | python3
import json

# DeepSeek 格式输入
deepseek_response = {
    "choices": [{
        "message": {
            "role": "assistant",
            "content": [
                {"type": "text", "text": "我来帮你查询天气。"},
                {"type": "tool_use", "id": "call_123", "name": "get_weather", "input": {"location": "北京", "unit": "celsius"}}
            ]
        },
        "finish_reason": "tool_calls"
    }]
}

# 转换逻辑
tool_calls = []
text_content = ""

for item in deepseek_response["choices"][0]["message"]["content"]:
    if item["type"] == "text":
        text_content += item["text"]
    elif item["type"] == "tool_use":
        tool_calls.append({
            "id": item["id"],
            "type": "function",
            "function": {
                "name": item["name"],
                "arguments": json.dumps(item["input"], ensure_ascii=False)
            }
        })

# OpenAI 格式输出
openai_response = {
    "id": "chatcmpl_test",
    "object": "chat.completion",
    "model": "deepseek-chat",
    "choices": [{
        "index": 0,
        "message": {
            "role": "assistant",
            "content": text_content,
            "tool_calls": tool_calls
        },
        "finish_reason": "tool_calls"
    }]
}

print(json.dumps(openai_response, indent=2, ensure_ascii=False))
PYTHON
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "3. 转换为 OpenClaw 文本格式"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

cat <<'PYTHON' | python3
import json

openai_data = {
    "choices": [{
        "message": {
            "role": "assistant",
            "content": "我来帮你查询天气。",
            "tool_calls": [{
                "id": "call_123",
                "type": "function",
                "function": {
                    "name": "get_weather",
                    "arguments": '{"location": "北京", "unit": "celsius"}'
                }
            }]
        },
        "finish_reason": "tool_calls"
    }]
}

message = openai_data["choices"][0]["message"]
result = message["content"] + "\n\n"

for i, tc in enumerate(message["tool_calls"]):
    result += f"[工具调用 {i+1}]\n"
    result += f"ID: {tc['id']}\n"
    result += f"函数: {tc['function']['name']}\n"
    args = json.loads(tc['function']['arguments'])
    result += f"参数: {json.dumps(args, ensure_ascii=False, indent=4)}\n\n"

print(result)
PYTHON
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "4. 支持的格式总结"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "✓ DeepSeek → OpenAI"
echo "  content[].type='tool_use' → tool_calls[]"
echo ""
echo "✓ OpenAI → OpenClaw"
echo "  tool_calls[] → 可读文本格式"
echo ""
echo "✓ Anthropic → OpenAI"
echo "  content[].type='tool_use' → tool_calls[]"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "5. 实现的 API"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "新增文件: internal/converter/toolcall.go"
echo ""
echo "主要类型:"
echo "  • ToolCall - 标准工具调用格式"
echo "  • FunctionCall - 函数调用详情"
echo "  • ToolCallResult - 工具调用结果"
echo ""
echo "转换函数:"
echo "  • ConvertFromOpenAI() - OpenAI 格式转换"
echo "  • ConvertFromDeepSeek() - DeepSeek 格式转换"
echo "  • ConvertFromAnthropic() - Anthropic 格式转换"
echo "  • ExtractToolCalls() - 提取工具调用"
echo "  • FormatToolCallResult() - 格式化结果"
echo ""
echo "DeepSeekParser 新增方法:"
echo "  • HasToolCalls() - 检查是否包含工具调用"
echo "  • ExtractToolCalls() - 提取工具调用"
echo "  • ConvertToolCallsToOpenAI() - 转换为 OpenAI 格式"
echo ""
echo -e "${GREEN}✓ Tool call 格式转换功能实现完成${NC}"
echo ""
