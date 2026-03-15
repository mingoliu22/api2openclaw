# 多模态 API 使用指南

api2openclaw 支持多模态输入，包括图片、音频等内容类型。

## 概述

多模态支持框架允许客户端发送包含图片、音频等非文本内容的消息。系统会：
- 解析多模态内容
- 验证文件大小和类型
- 转换为目标模型格式
- 记录多模态调用日志

## 支持的内容类型

| 类型 | 说明 | MIME 类型 |
|------|------|-----------|
| text | 纯文本 | text/plain |
| image_url | 图片 URL 或 Base64 | image/jpeg, image/png, image/gif, image/webp |
| audio_url | 音频 URL 或 Base64 | audio/mpeg, audio/wav, audio/ogg, audio/mp4 |

## 请求格式

### OpenAI 兼容格式

```json
{
  "model": "gpt-4-vision-preview",
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "这张图片里有什么？"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://example.com/image.jpg",
            "detail": "high"
          }
        }
      ]
    }
  ]
}
```

### 使用 Base64 编码的图片

```json
{
  "model": "gpt-4-vision-preview",
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "描述这张图片"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD/2wBD..."
          }
        }
      ]
    }
  ]
}
```

### 混合文本和图片

```json
{
  "model": "gpt-4-vision-preview",
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "比较这两张图片的异同点。"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://example.com/image1.jpg"
          }
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://example.com/image2.jpg"
          }
        }
      ]
    }
  ]
}
```

### 音频输入（未来支持）

```json
{
  "model": "whisper-1",
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "转录这段音频"
        },
        {
          "type": "audio_url",
          "audio_url": {
            "url": "data:audio/wav;base64,UklGRiQAAABXQVZFZm10IBAAAAABAAEARKwAAIhYAQACABAA..."
          }
        }
      ]
    }
  ]
}
```

## 配置多模态支持

### 后端配置 (config.yaml)

```yaml
converter:
  input_format: openai-multimodal  # 启用多模态输入解析
  output_format: openclaw
  enable_multimodal: true          # 启用多模态支持
  templates:
    message: "%s"
    stream_chunk: "%s"
```

### 环境变量

```bash
# 启用多模态支持
ENABLE_MULTIMODAL=true
```

## 限制

### 图片限制

| 项目 | 限制 |
|------|------|
| 单张图片大小 | 20MB |
| Base64 编码 | 支持 |
| 支持格式 | JPEG, PNG, GIF, WebP |
| Detail 模式 | low, high, auto |

### 音频限制

| 项目 | 限制 |
|------|------|
| 单个音频大小 | 50MB |
| 支持格式 | MP3, WAV, OGG, M4A |

### Token 计算

多模态消息的 token 计算规则：

- **文本**: 约 4 字符/token (英文), 2 字符/token (中文)
- **图片**:
  - Low detail: 85 tokens
  - High detail: 根据分辨率计算 (约 255 tokens 保守估计)
- **音频**: 取决于音频时长和模型

## 客户端示例

### Python

```python
import base64
import requests

# 方式1: 使用图片 URL
response = requests.post(
    "http://localhost:8080/v1/chat/completions",
    headers={
        "Authorization": "Bearer YOUR_API_KEY",
        "Content-Type": "application/json"
    },
    json={
        "model": "gpt-4-vision-preview",
        "messages": [
            {
                "role": "user",
                "content": [
                    {"type": "text", "text": "这张图片是什么？"},
                    {
                        "type": "image_url",
                        "image_url": {
                            "url": "https://example.com/image.jpg"
                        }
                    }
                ]
            }
        ]
    }
)

# 方式2: 使用 Base64 编码
def encode_image(image_path):
    with open(image_path, "rb") as f:
        return base64.b64encode(f.read()).decode('utf-8')

base64_image = encode_image("path/to/image.jpg")

response = requests.post(
    "http://localhost:8080/v1/chat/completions",
    headers={
        "Authorization": "Bearer YOUR_API_KEY",
        "Content-Type": "application/json"
    },
    json={
        "model": "gpt-4-vision-preview",
        "messages": [
            {
                "role": "user",
                "content": [
                    {"type": "text", "text": "描述这张图片"},
                    {
                        "type": "image_url",
                        "image_url": {
                            "url": f"data:image/jpeg;base64,{base64_image}"
                        }
                    }
                ]
            }
        ]
    }
)

print(response.json())
```

### JavaScript

```javascript
// 方式1: 使用图片 URL
const response = await fetch('http://localhost:8080/v1/chat/completions', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer YOUR_API_KEY',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    model: 'gpt-4-vision-preview',
    messages: [
      {
        role: 'user',
        content: [
          { type: 'text', text: '这张图片是什么？' },
          {
            type: 'image_url',
            image_url: {
              url: 'https://example.com/image.jpg'
            }
          }
        ]
      }
    ]
  })
});

// 方式2: 使用 Base64 编码
async function encodeImage(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onloadend = () => resolve(reader.result.split(',')[1]);
    reader.onerror = reject;
    reader.readAsDataURL(file);
  });
}

const fileInput = document.querySelector('#image-input');
const file = fileInput.files[0];
const base64 = await encodeImage(file);

const response = await fetch('http://localhost:8080/v1/chat/completions', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer YOUR_API_KEY',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    model: 'gpt-4-vision-preview',
    messages: [
      {
        role: 'user',
        content: [
          { type: 'text', text: '描述这张图片' },
          {
            type: 'image_url',
            image_url: {
              url: `data:image/jpeg;base64,${base64}`
            }
          }
        ]
      }
    ]
  })
});

const data = await response.json();
console.log(data);
```

### cURL

```bash
# 使用图片 URL
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4-vision-preview",
    "messages": [
      {
        "role": "user",
        "content": [
          {"type": "text", "text": "这张图片是什么？"},
          {
            "type": "image_url",
            "image_url": {"url": "https://example.com/image.jpg"}
          }
        ]
      }
    ]
  }'

# 使用 Base64 编码（需要先编码图片）
BASE64_IMAGE=$(base64 -i image.jpg)

curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d "{
    \"model\": \"gpt-4-vision-preview\",
    \"messages\": [
      {
        \"role\": \"user\",
        \"content\": [
          {\"type\": \"text\", \"text\": \"描述这张图片\"},
          {
            \"type\": \"image_url\",
            \"image_url\": {\"url\": \"data:image/jpeg;base64,$BASE64_IMAGE\"}
          }
        ]
      }
    ]
  }"
```

## 响应格式

多模态请求的响应与标准文本请求相同：

```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1699000000,
  "model": "gpt-4-vision-preview",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "这是一张风景照片，展示了..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 1100,
    "completion_tokens": 50,
    "total_tokens": 1150
  }
}
```

## 注意事项

1. **性能考虑**: 图片和音频处理会增加请求时间和 token 消耗
2. **安全**: 建议对上传的文件进行病毒扫描和内容审核
3. **隐私**: Base64 编码会增加约 33% 的数据大小
4. **兼容性**: 不是所有后端模型都支持多模态输入，请确认目标模型能力
5. **流式响应**: 多模态内容在流式响应中只返回文本部分

## 未来扩展

- [ ] 视频输入支持
- [ ] 文件上传接口
- [ ] 多模态输出（生成图片）
- [ ] 语音输入/输出
- [ ] 实时视频流处理
