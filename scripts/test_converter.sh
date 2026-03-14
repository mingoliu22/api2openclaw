#!/bin/bash
# 测试格式转换功能

set -e

echo "=== api2openclaw 格式转换测试 ==="
echo ""

# 测试数据：DeepSeek 格式
TEST_DATA='{
  "choices": [{
    "message": {
      "role": "assistant",
      "content": [
        {"type": "text", "text": "Hello, "},
        {"type": "text", "text": "this is a test!"}
      ]
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 20,
    "total_tokens": 30
  }
}'

echo "输入数据 (DeepSeek 格式):"
echo "$TEST_DATA"
echo ""

echo "期望输出 (OpenClaw 纯文本):"
echo "Hello, this is a test!"
echo ""

# 测试单元转换
echo "=== 运行单元测试 ==="
cd /Users/liu/openclaw/api2openclaw
export PATH=$PATH:~/go-install/go/bin
export GOPATH=~/go

go test -v ./internal/converter/... -run TestDeepSeekConverter

echo ""
echo "=== 测试完成 ==="
