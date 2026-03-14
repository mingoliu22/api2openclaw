# api2openclaw 项目总结

## 项目概述

**api2openclaw** 是一个本地 LLM 统一网关与格式转换服务，用于将本地部署的开源模型（如 DeepSeek）统一暴露为标准 API，并提供格式转换功能。

---

## 完成状态

### ✅ 阶段一：核心功能
- 认证网关（PostgreSQL 存储）
- 格式转换引擎（DeepSeek → OpenClaw）
- HTTP API 服务
- 数据库迁移

### ✅ 阶段二：路由与监控
- 模型路由器（4种策略）
- 健康检查
- 限流器（内存存储）
- 熔断器
- 聊天完成接口

### ✅ 阶段三：完整功能
- Prometheus 指标暴露
- 后端请求转发
- 活跃请求跟踪
- 管理接口端点

---

## 项目结构

```
api2openclaw/
├── cmd/api2openclaw/         # 入口文件
├── internal/
│   ├── auth/                 # 认证网关
│   │   ├── manager.go
│   │   ├── postgres_store.go
│   │   ├── middleware.go
│   │   └── errors.go
│   ├── converter/            # 格式转换
│   │   ├── converter.go
│   │   └── deepseek_parser.go
│   ├── router/               # 模型路由器
│   │   ├── router.go
│   │   ├── strategy.go
│   │   ├── health.go
│   │   └── forwarder.go
│   ├── monitor/              # 用量监控
│   │   ├── metrics.go
│   │   ├── ratelimit.go
│   │   ├── circuit.go
│   │   └── prometheus.go
│   ├── server/               # HTTP 服务器
│   │   └── server.go
│   ├── config/               # 配置系统
│   │   ├── config.go
│   │   └── loader.go
│   └── models/               # 数据模型
│       ├── tenant.go
│       ├── apikey.go
│       ├── backend.go
│       └── model.go
├── configs/                  # 配置文件
├── migrations/               # 数据库迁移
├── deployments/
│   ├── docker/
│   ├── kubernetes/
│   └── systemd/
├── scripts/                  # 脚本
│   ├── test_full_api.sh
│   ├── test_converter.sh
│   ├── test_stage2.sh
│   ├── test_stage3.sh
│   └── demo.sh
├── docs/                     # 文档
│   ├── PROJECT_SUMMARY.md
│   └── PROJECT_STATUS.md
├── go.mod
├── Makefile
└── README.md
```

---

## API 接口

### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查（含后端状态、活跃请求） |
| POST | `/v1/convert` | 格式转换 |

### 认证接口 (需要 Bearer Token)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/v1/chat/completions` | 聊天完成（路由到后端） |

### 管理接口 (需要 admin 权限)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/v1/admin/backends` | 列出后端 |
| GET | `/v1/admin/models` | 列出模型 |
| GET | `/v1/admin/stats` | 获取统计信息 |
| POST | `/v1/admin/tenants` | 创建租户 |
| GET | `/v1/admin/tenants` | 列出租户 |
| POST | `/v1/admin/api-keys` | 创建 API Key |
| GET | `/v1/admin/api-keys` | 列出 API Keys |
| DELETE | `/v1/admin/api-keys/:id` | 删除 API Key |

---

## 核心功能

### 1. 格式转换

```
DeepSeek 格式 → OpenClaw 格式

输入: [{"type":"text","text":"Hello, World!"}]
输出: Hello, World!
```

### 2. 模型路由

支持 4 种路由策略：
- `direct` - 直接策略（选择第一个）
- `round-robin` - 轮询策略
- `least-connections` - 最少连接策略
- `random` - 随机策略

### 3. 限流

支持三种限流窗口：
- 每分钟请求数
- 每小时请求数
- 每天请求数

### 4. 熔断器

自动检测后端故障并熔断：
- 连续错误阈值触发熔断
- 超时后自动尝试恢复
- 半开状态验证恢复

---

## 技术栈

| 组件 | 技术 |
|------|------|
| 语言 | Go 1.23 |
| 框架 | Gin |
| 数据库 | PostgreSQL 16 |
| 监控 | Prometheus |
| 容器 | Docker |
| 部署 | Systemd / K8s |

---

## 快速开始

### Docker 部署

```bash
cd deployments/docker
docker-compose up -d
```

### 本地运行

```bash
make deps
make run
```

### 测试

```bash
./scripts/test_full_api.sh    # 完整功能测试
./scripts/test_stage3.sh      # 阶段三测试
```

---

## 配置示例

```yaml
server:
  port: 8080

auth:
  enabled: true
  database:
    host: "localhost"
    port: 5432
    user: "api2openclaw"
    password: "api2openclaw123"
    database: "api2openclaw"

router:
  backends:
    - id: "deepseek-primary"
      name: "DeepSeek Primary"
      type: "openai-compatible"
      base_url: "http://localhost:8000/v1"
      health_check:
        enabled: true
        interval: 30s

  models:
    - name: "deepseek-chat"
      backend_group: ["deepseek-primary"]
      routing_strategy: "direct"

converter:
  input_format: "deepseek"
  output_format: "openclaw"

monitor:
  enabled: true
  prometheus:
    enabled: true
    listen_address: ":9090"
```

---

## 服务端点

- **API**: http://localhost:8080
- **Health**: http://localhost:8080/health
- **Metrics**: http://localhost:8080/v1/metrics

---

## 许可证

MIT
