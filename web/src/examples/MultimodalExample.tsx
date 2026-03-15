/**
 * 多模态 API 调用示例
 *
 * 本示例演示如何使用 api2openclaw 的多模态功能发送包含图片的消息
 */

import { useState, useRef } from 'react';

interface Message {
  role: 'user' | 'assistant';
  content: string;
  imageUrl?: string;
}

interface MultimodalContent {
  type: 'text' | 'image_url';
  text?: string;
  image_url?: {
    url: string;
    detail?: 'low' | 'high' | 'auto';
  };
}

export default function MultimodalExample() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [selectedImage, setSelectedImage] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // 将图片转换为 Base64
  const encodeImageToBase64 = (file: File): Promise<string> => {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onloadend = () => {
        const base64 = reader.result as string;
        resolve(base64);
      };
      reader.onerror = reject;
      reader.readAsDataURL(file);
    });
  };

  // 处理文件选择
  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // 验证文件类型
    const allowedTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];
    if (!allowedTypes.includes(file.type)) {
      alert('请选择 JPG、PNG、GIF 或 WebP 格式的图片');
      return;
    }

    // 验证文件大小（20MB 限制）
    const maxSize = 20 * 1024 * 1024;
    if (file.size > maxSize) {
      alert('图片大小不能超过 20MB');
      return;
    }

    try {
      const base64 = await encodeImageToBase64(file);
      setSelectedImage(base64);
    } catch (error) {
      console.error('编码图片失败:', error);
      alert('编码图片失败');
    }
  };

  // 发送多模态消息
  const handleSend = async () => {
    if (!input.trim() && !selectedImage) return;

    setLoading(true);

    // 构建内容数组
    const content: MultimodalContent[] = [];

    // 添加文本内容
    if (input.trim()) {
      content.push({
        type: 'text',
        text: input,
      });
    }

    // 添加图片内容
    if (selectedImage) {
      content.push({
        type: 'image_url',
        image_url: {
          url: selectedImage,
          detail: 'high', // 可选: 'low', 'high', 'auto'
        },
      });
    }

    // 添加用户消息到界面
    const userMessage: Message = {
      role: 'user',
      content: input,
      imageUrl: selectedImage || undefined,
    };
    setMessages((prev) => [...prev, userMessage]);

    try {
      // 调用 api2openclaw API
      const response = await fetch('/v1/chat/completions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          // 如果需要认证，添加 Authorization header
          // 'Authorization': 'Bearer YOUR_API_KEY',
        },
        body: JSON.stringify({
          model: 'gpt-4-vision-preview', // 或其他支持视觉的模型
          messages: [
            {
              role: 'user',
              content: content,
            },
          ],
          max_tokens: 500,
        }),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();

      // 添加助手回复到界面
      if (data.choices && data.choices[0]) {
        const assistantMessage: Message = {
          role: 'assistant',
          content: data.choices[0].message.content,
        };
        setMessages((prev) => [...prev, assistantMessage]);
      }
    } catch (error) {
      console.error('请求失败:', error);
      alert('请求失败，请检查网络连接和 API 配置');
    } finally {
      setLoading(false);
      setInput('');
      setSelectedImage(null);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  // 清除图片
  const clearImage = () => {
    setSelectedImage(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  return (
    <div className="max-w-4xl mx-auto p-6">
      <h1 className="text-3xl font-bold mb-4">多模态 API 示例</h1>
      <p className="text-gray-600 mb-6">
        演示如何发送包含图片的消息到支持视觉的 AI 模型
      </p>

      {/* 消息历史 */}
      <div className="bg-gray-50 rounded-lg p-4 mb-4 h-96 overflow-y-auto">
        {messages.length === 0 ? (
          <p className="text-gray-400 text-center">请输入消息或选择图片开始对话</p>
        ) : (
          messages.map((msg, idx) => (
            <div
              key={idx}
              className={`mb-4 ${msg.role === 'user' ? 'text-right' : 'text-left'}`}
            >
              <div
                className={`inline-block max-w-[80%] p-3 rounded-lg ${
                  msg.role === 'user'
                    ? 'bg-blue-500 text-white'
                    : 'bg-white border border-gray-200'
                }`}
              >
                {msg.imageUrl && (
                  <img
                    src={msg.imageUrl}
                    alt="上传的图片"
                    className="max-w-full h-auto mb-2 rounded"
                  />
                )}
                <p className="whitespace-pre-wrap">{msg.content}</p>
              </div>
            </div>
          ))}
          {loading && (
            <div className="text-left">
              <div className="inline-block bg-gray-200 px-4 py-2 rounded-lg">
                <span className="animate-pulse">正在思考...</span>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* 输入区域 */}
      <div className="border border-gray-300 rounded-lg p-4 bg-white">
        {/* 图片预览 */}
        {selectedImage && (
          <div className="mb-4 relative inline-block">
            <img
              src={selectedImage}
              alt="预览"
              className="max-w-xs h-auto rounded border border-gray-300"
            />
            <button
              onClick={clearImage}
              className="absolute top-0 right-0 bg-red-500 text-white rounded-full w-6 h-6 flex items-center justify-center hover:bg-red-600"
            >
              ×
            </button>
          </div>
        )}

        {/* 文本输入 */}
        <textarea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="输入你的问题..."
          className="w-full p-3 border border-gray-300 rounded-lg resize-none focus:outline-none focus:ring-2 focus:ring-blue-500"
          rows={3}
          disabled={loading}
        />

        {/* 操作按钮 */}
        <div className="flex justify-between items-center mt-3">
          <div>
            <input
              ref={fileInputRef}
              type="file"
              accept="image/jpeg,image/png,image/gif,image/webp"
              onChange={handleFileSelect}
              className="hidden"
            />
            <button
              onClick={() => fileInputRef.current?.click()}
              disabled={loading}
              className="px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 disabled:opacity-50"
            >
              📷 选择图片
            </button>
            <span className="ml-2 text-sm text-gray-500">
              支持 JPG、PNG、GIF、WebP，最大 20MB
            </span>
          </div>

          <button
            onClick={handleSend}
            disabled={loading || (!input.trim() && !selectedImage)}
            className="px-6 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:opacity-50"
          >
            {loading ? '发送中...' : '发送'}
          </button>
        </div>
      </div>

      {/* 使用说明 */}
      <div className="mt-6 p-4 bg-blue-50 border border-blue-200 rounded-lg">
        <h3 className="font-semibold text-blue-800 mb-2">使用说明</h3>
        <ul className="text-sm text-blue-700 space-y-1">
          <li>• 可以同时发送文本和图片</li>
          <li>• 图片支持 Base64 编码或 URL</li>
          <li>• Detail 模式影响图片质量和 token 消耗</li>
          <li>• Low detail: 85 tokens, High detail: ~255 tokens</li>
        </ul>
      </div>
    </div>
  );
}

/**
 * 使用其他语言的示例：
 */

/**
 * Python 示例
 */
export const pythonExample = `
import base64
import requests

# 编码图片
def encode_image(image_path):
    with open(image_path, "rb") as f:
        return base64.b64encode(f.read()).decode('utf-8')

base64_image = encode_image("path/to/image.jpg")

# 发送请求
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
                            "url": f"data:image/jpeg;base64,{base64_image}",
                            "detail": "high"
                        }
                    }
                ]
            }
        ]
    }
)

print(response.json())
`;

/**
 * JavaScript/Node.js 示例
 */
export const jsExample = `
const fs = require('fs');
const fetch = require('node-fetch');

// 编码图片
function encodeImage(imagePath) {
  const imageBuffer = fs.readFileSync(imagePath);
  return imageBuffer.toString('base64');
}

const base64Image = encodeImage('path/to/image.jpg');

// 发送请求
async function sendMultimodalMessage() {
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
                url: \`data:image/jpeg;base64,\${base64Image}\`,
                detail: 'high'
              }
            }
          ]
        }
      ]
    })
  });

  const data = await response.json();
  console.log(data);
}

sendMultimodalMessage();
`;

/**
 * cURL 示例
 */
export const curlExample = `
# 先编码图片
BASE64_IMAGE=$(base64 -i image.jpg)

# 发送请求
curl http://localhost:8080/v1/chat/completions \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4-vision-preview",
    "messages": [
      {
        "role": "user",
        "content": [
          {"type": "text", "text": "这张图片是什么？"},
          {
            "type": "image_url",
            "image_url": {
              "url": "data:image/jpeg;base64,'"$BASE64_IMAGE"'"
            }
          }
        ]
      }
    ]
  }'
`;
