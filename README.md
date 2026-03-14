# api2openclaw

本地 LLM 统一网关与格式转换服务

## 功能特性

### 阶段一（已完成）
- ✅ **认证网关** - 基于 PostgreSQL 的 API Key 管理与权限控制
- ✅ **格式转换引擎** - DeepSeek 格式转换为 OpenClaw 纯文本

### 阶段二（规划中）
- ⏳ 模型路由器
- ⏳ 用量监控

### 阶段三（规划中）
- ⏳ Web 管理后台

## 格式转换说明

### DeepSeek 格式 → OpenClaw 纯文本

**输入 (DeepSeek):**
```json
{
  "choices": [{
    "message": {
      "role": "assistant",
      "content": [
        {"type": "text", "text": "Hello, "}
      ]
    }
  }]
}
```

**输出 (OpenClaw):**
```
Hello, !
```

## 快速开始

### Docker 部署

```bash
cd api2openclaw/deployments/docker
docker-compose up -d
```

### Systemd 部署

```bash
# 安装
sudo cp deployments/systemd/api2openclaw.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable api2openclaw
sudo systemctl start api2openclaw
```

### 本地开发

```bash
# 安装依赖
make deps

# 运行
make run

# 构建
make build
```

## API 接口

### 格式转换

```bash
POST /v1/convert
Content-Type: application/json

{
  "data": [
    {
      "choices": [{
        "message": {
          "content": [
            {"type": "text", "text": "Hello, world!"}
          ]
        }
      }]
    }
  ]
}
```

### 管理 API（需要认证）

```bash
# 创建租户
POST /v1/admin/tenants
Authorization: Bearer <admin-key>
Content-Type: application/json

{
  "name": "My Team",
  "tier": "pro"
}

# 创建 API Key
POST /v1/admin/api-keys
Authorization: Bearer <admin-key>
Content-Type: application/json

{
  "tenant_id": "tenant_xxx",
  "allowed_models": ["*"],
  "rate_limit": {
    "requests_per_minute": 100
  }
}
```

## 配置

配置文件位置: `configs/config.yaml`

```yaml
server:
  port: 8080

auth:
  enabled: true
  database:
    host: localhost
    port: 5432
    user: api2openclaw
    password: changeme
    database: api2openclaw

converter:
  input_format: deepseek
  output_format: openclaw
  templates:
    message: "%s"
```

## 架构

```
┌─────────────────┐
│   客户端请求    │
└────────┬────────┘
         ↓
┌─────────────────┐
│   认证网关      │ ← API Key 验证
│   (PostgreSQL)  │
└────────┬────────┘
         ↓
┌─────────────────┐
│  格式转换引擎   │ ← DeepSeek → OpenClaw
└────────┬────────┘
         ↓
┌─────────────────┐
│   本地模型      │
│  (deepseek-v3)  │
└─────────────────┘
```

## 许可证

MIT
