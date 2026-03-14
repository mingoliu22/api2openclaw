# api2openclaw 项目总结

## 项目概述

**api2openclaw** 是一个本地 LLM 统一网关与格式转换服务，用于将本地部署的开源模型（如 DeepSeek）统一暴露为标准 API，并提供格式转换功能。

---

## 已完成功能

### 阶段一：核心功能 ✅

#### 1. 认证网关 (Auth Gateway)
- 多租户管理 (Tenant)
- API Key 生成与验证
- 基于 PostgreSQL 的持久化存储
- 权限检查和配额管理
- 认证中间件

**文件结构：**
```
internal/auth/
├── manager.go         # 认证管理器
├── postgres_store.go  # PostgreSQL 存储
├── middleware.go      # 认证中间件
└── errors.go          # 错误定义
```

#### 2. 格式转换引擎 (Format Converter)
- DeepSeek 格式解析：`[{'type':'text','text':'xxx'}]` → `xxx`
- OpenClaw 纯文本输出
- 支持多种输入格式 (DeepSeek, OpenAI JSON)
- 可配置输出模板
- 流式输出支持

**文件结构：**
```
internal/converter/
├── converter.go       # 转换器接口
└── deepseek_parser.go # DeepSeek 解析器
```

#### 3. HTTP API 服务
- RESTful API 接口
- 健康检查端点
- 格式转换端点
- 管理接口

### 阶段二：路由与监控 ✅

#### 4. 模型路由器 (Model Router)
- 多后端实例管理
- 4种路由策略：
  - `direct` - 直接策略
  - `round-robin` - 轮询策略
  - `least-connections` - 最少连接策略
  - `random` - 随机策略
- 自动健康检查
- 故障自动转移
- 模型别名配置

**文件结构：**
```
internal/router/
├── router.go      # 路由器核心
├── strategy.go    # 路由策略
└── health.go      # 健康检查
```

#### 5. 用量监控 (Usage Monitor)
- 指标采集框架
- 速率限制器（内存存储）
- 熔断器
- 请求上下文跟踪

**文件结构：**
```
internal/monitor/
├── metrics.go    # 指标采集
├── ratelimit.go  # 限流器
└── circuit.go    # 熔断器
```

---

## API 接口

### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查（含后端状态） |
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
| POST | `/v1/admin/tenants` | 创建租户 |
| GET | `/v1/admin/tenants` | 列出租户 |
| POST | `/v1/admin/api-keys` | 创建 API Key |
| GET | `/v1/admin/api-keys` | 列出 API Keys |
| DELETE | `/v1/admin/api-keys/:id` | 删除 API Key |

---

## 项目结构

```
api2openclaw/
├── cmd/api2openclaw/         # 入口文件
├── internal/
│   ├── auth/                 # 认证网关
│   ├── converter/            # 格式转换
│   ├── router/               # 模型路由器
│   ├── monitor/              # 用量监控
│   ├── server/               # HTTP 服务器
│   ├── config/               # 配置系统
│   └── models/               # 数据模型
├── configs/                  # 配置文件
├── migrations/               # 数据库迁移
├── deployments/
│   ├── docker/               # Docker 配置
│   ├── kubernetes/           # K8s 配置
│   └── systemd/              # Systemd 服务
├── scripts/                  # 脚本
│   ├── test_full_api.sh      # 完整功能测试
│   ├── test_converter.sh     # 转换器测试
│   ├── test_stage2.sh        # 阶段二测试
│   └── demo.sh               # 使用演示
├── go.mod
├── Makefile
└── README.md
```

---

## 快速开始

### Docker 部署

```bash
cd deployments/docker
docker-compose up -d
```

### Systemd 部署

```bash
make build
sudo make install
sudo systemctl start api2openclaw
```

### 本地开发

```bash
make deps
make run
```

---

## 测试

```bash
# 完整功能测试
./scripts/test_full_api.sh

# 阶段二测试
./scripts/test_stage2.sh

# 单元测试
go test -v ./...
```

---

## 配置示例

```yaml
server:
  host: "0.0.0.0"
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
  rate_limiting:
    storage: "memory"
```

---

## 待实现功能

- [ ] 实际后端请求转发
- [ ] 指标持久化存储
- [ ] Prometheus 指标暴露
- [ ] Web 管理后台
- [ ] 流式输出端到端支持

---

## 技术栈

| 组件 | 技术 |
|------|------|
| 语言 | Go 1.23 |
| 框架 | Gin |
| 数据库 | PostgreSQL 16 |
| 容器 | Docker |
| 部署 | Systemd / K8s |

---

## 许可证

MIT
