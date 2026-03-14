#!/usr/bin/env python3
"""
模拟 LLM 服务器 - 用于测试 api2openclaw 的流式输出功能

在 8000 端口启动一个简单的 HTTP 服务器，模拟 OpenAI 兼容的流式 API。
"""

from http.server import HTTPServer, BaseHTTPRequestHandler
import json
import time
from datetime import datetime

class MockLLMHandler(BaseHTTPRequestHandler):
    """模拟 LLM 请求处理器"""

    def do_GET(self):
        """处理 GET 请求"""
        if self.path == '/health' or self.path == '/v1/models':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()

            if self.path == '/health':
                response = {"status": "ok"}
            else:  # /v1/models
                response = {
                    "object": "list",
                    "data": [
                        {
                            "id": "deepseek-chat",
                            "object": "model",
                            "created": 1677610602,
                            "owned_by": "deepseek"
                        }
                    ]
                }
            self.wfile.write(json.dumps(response).encode())
        else:
            self.send_response(404)
            self.end_headers()

    def do_POST(self):
        """处理 POST 请求"""
        if self.path == '/v1/chat/completions':
            try:
                content_length = int(self.headers['Content-Length'])
                body = self.rfile.read(content_length)
                data = json.loads(body.decode())
                stream = data.get('stream', False)

                if stream:
                    # 流式响应
                    self.send_response(200)
                    self.send_header('Content-Type', 'text/event-stream')
                    self.send_header('Cache-Control', 'no-cache')
                    self.send_header('Connection', 'keep-alive')
                    self.end_headers()

                    created = int(datetime.now().timestamp())
                    chunk_id = f"chatcmpl-{created}"

                    # 模拟逐字符输出
                    text = "这是来自模拟 LLM 的流式响应。"
                    for char in text:
                        chunk = {
                            "id": chunk_id,
                            "object": "chat.completion.chunk",
                            "created": created,
                            "model": data.get("model", "deepseek-chat"),
                            "choices": [
                                {
                                    "index": 0,
                                    "delta": {"content": char},
                                    "finish_reason": None
                                }
                            ]
                        }
                        self.wfile.write(f"data: {json.dumps(chunk)}\n\n".encode())
                        self.wfile.flush()
                        time.sleep(0.03)

                    # 发送结束信号
                    end_chunk = {
                        "id": chunk_id,
                        "object": "chat.completion.chunk",
                        "created": created,
                        "model": data.get("model", "deepseek-chat"),
                        "choices": [
                            {
                                "index": 0,
                                "delta": {},
                                "finish_reason": "stop"
                            }
                        ]
                    }
                    self.wfile.write(f"data: {json.dumps(end_chunk)}\n\n".encode())
                    self.wfile.write(b"data: [DONE]\n\n")
                else:
                    # 非流式响应
                    self.send_response(200)
                    self.send_header('Content-Type', 'application/json')
                    self.end_headers()

                    created = int(datetime.now().timestamp())
                    response = {
                        "id": f"chatcmpl-{created}",
                        "object": "chat.completion",
                        "created": created,
                        "model": data.get("model", "deepseek-chat"),
                        "choices": [
                            {
                                "index": 0,
                                "message": {
                                    "role": "assistant",
                                    "content": "这是来自模拟 LLM 的响应。"
                                },
                                "finish_reason": "stop"
                            }
                        ],
                        "usage": {
                            "prompt_tokens": 10,
                            "completion_tokens": 20,
                            "total_tokens": 30
                        }
                    }
                    self.wfile.write(json.dumps(response).encode())
            except Exception as e:
                self.send_response(500)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                error = {"error": {"message": str(e), "type": "internal_error"}}
                self.wfile.write(json.dumps(error).encode())
        else:
            self.send_response(404)
            self.end_headers()

    def log_message(self, format, *args):
        """自定义日志输出"""
        print(f"[{datetime.now().strftime('%H:%M:%S')}] {format % args}")

def run_server():
    """启动服务器"""
    server_address = ('0.0.0.0', 8000)
    httpd = HTTPServer(server_address, MockLLMHandler)

    print("╔══════════════════════════════════════════════════════════════════╗")
    print("║              模拟 LLM 服务器                                      ║")
    print("╚══════════════════════════════════════════════════════════════════╝")
    print("")
    print("启动地址: http://localhost:8000")
    print("健康检查: http://localhost:8000/health")
    print("模型列表: http://localhost:8000/v1/models")
    print("聊天完成: POST http://localhost:8000/v1/chat/completions")
    print("")
    print("按 Ctrl+C 停止服务器")
    print("")

    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("\n服务器已停止")
        httpd.server_close()

if __name__ == '__main__':
    run_server()
