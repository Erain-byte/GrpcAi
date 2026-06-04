# Gateway Service

API 网关服务，负责请求路由、负载均衡、服务发现和协议转发。

## 📋 功能特性

- ✅ **动态路由**：根据 YAML 配置将 HTTP/gRPC 请求转发到对应的后端服务
- ✅ **服务发现**：从 Consul 自动发现后端服务实例，支持健康检查
- ✅ **负载均衡**：基于 Round-Robin 算法在多个服务实例间分配流量
- ✅ **双协议支持**：同时提供 HTTP REST API 和 gRPC 接口
- ✅ **CORS 支持**：统一的跨域资源共享配置，从配置文件读取
- ✅ **JWT 认证**：在网关层进行统一鉴权，支持公开/受保护接口配置
- ✅ **反向代理**：使用 `httputil.ReverseProxy` 实现高性能请求转发
- ✅ **健康检查**：提供 `/health` 端点，实时监控后端服务状态
- ✅ **优雅关闭**：支持平滑停机，确保请求处理完毕
- ✅ **Consul 注册**：自动注册 HTTP + gRPC 服务并维持 TTL 心跳
- ✅ **限流保护**：基于令牌桶算法的请求速率控制，防止雪崩
- ✅ **熔断机制**：基于断路器模式的故障隔离，提升系统稳定性
- ✅ **防重放攻击**：基于时间戳和 Nonce 的请求唯一性验证，防止请求被重复提交
- ✅ **结构化日志**：基于 zap 的 JSON 格式日志，支持文件滚动和集中收集
- ✅ **链路追踪**：基于 OpenTelemetry + Jaeger 的分布式追踪，支持跨服务调用链分析

## 🏗️ 架构设计

Gateway 采用分层架构设计，主要包含以下核心流程：

1. **接入层**：接收 HTTP (8080) 和 gRPC (9080) 请求，处理 JWT 认证、CORS 中间件及日志记录。
2. **路由层**：根据 YAML 配置进行路径匹配，确定目标后端服务。
3. **发现与负载均衡**：通过 Consul 进行服务发现，获取健康实例列表，并使用 Round-Robin 算法选择实例。
4. **转发层**：
   - **HTTP**：使用 `httputil.ReverseProxy` 进行高性能反向代理。
   - **gRPC**：通过泛型转发器将请求转发至后端 gRPC 服务。

### gRPC 转发架构

Gateway 采用**"泛型基类 + 服务拆分"**的架构模式，以实现高代码复用率和易扩展性：

- **base.go**: 定义 `BaseForwarder[T]` 泛型基类，提供通用的 gRPC 客户端获取、健康检查和连接管理能力。
- **xxxGrpc.go**: 每个业务服务（如 user, admin, ai）拥有独立的转发器文件，继承基类并实现具体业务方法的转发。
- **register.go**: 统一的服务注册中心，集中管理所有 gRPC 服务在 gRPC Server 上的注册逻辑，保持 `main.go` 简洁。
- **grpc/client.go**: `ClientManager` 负责底层的连接池管理、负载均衡策略执行以及基于 Consul 的服务发现。

这种设计使得代码复用率超过 90%，新增一个后端 gRPC 服务通常仅需约 60 行代码。

## 📁 目录结构

```
gateway/
├── cmd/server/main.go              # 入口文件：启动 HTTP + gRPC 服务、Consul 注册
├── etc/gateway.yaml                # 配置文件：路由、Consul、JWT、CORS、限流、熔断、日志、追踪等
├── internal/
│   ├── config/config.go            # 配置加载：解析 YAML 为 Go 结构体
│   ├── grpc/client.go              # gRPC 客户端管理：连接池、服务发现、负载均衡
│   ├── handler/handler.go          # HTTP 处理层（预留）
│   ├── logger/logger.go            # ⭐ 日志系统：zap 初始化、文件滚动
│   ├── logic/logic.go              # 业务逻辑层（预留）
│   ├── middleware/
│   │   ├── middleware.go           # 中间件：CORS、Recovery
│   │   ├── logger.go               # ⭐ 结构化日志中间件
│   │   ├── tracing.go              # ⭐ HTTP 链路追踪中间件
│   │   ├── grpc_tracing.go         # ⭐ gRPC 链路追踪拦截器
│   │   ├── jwt.go                  # JWT 认证中间件
│   │   ├── ratelimit.go            # ⭐ 限流中间件（令牌桶算法）
│   │   ├── circuitbreaker.go       # ⭐ 熔断中间件（断路器模式）
│   │   └── anti_replay.go          # ⭐ 防重放中间件（时间戳 + Nonce）
│   ├── registry/registry.go        # Consul 服务注册与发现
│   ├── server/
│   │   ├── base.go                 # 泛型转发器基类 BaseForwarder[T]
│   │   ├── userGrpc.go             # User 服务转发器实现
│   │   ├── adminGrpc.go            # Admin 服务转发器实现
│   │   ├── aiGrpc.go               # AI 服务转发器实现
│   │   ├── register.go             # ⭐ gRPC 服务统一注册中心
│   │   └── server.go               # HTTP 服务器和反向代理
│   ├── svc/servicecontext.go       # 依赖注入容器：Redis、Registry、GrpcClients
│   ├── tracer/tracer.go            # ⭐ 链路追踪工具包：OpenTelemetry + Jaeger
│   └── types/types.go              # 类型定义
├── logs/                           # ⭐ 日志文件目录（自动生成）
│   └── gateway.log                 # 当前日志文件
├── pkg/pkg.go                      # 公共工具包
├── go.mod                          # Go 模块依赖
├── go.sum                          # 依赖校验文件
├── test_rate_limit.sh              # ⭐ 限流功能测试脚本
├── test_logging.sh                 # ⭐ 日志系统测试脚本
└── README.md                       # 项目说明文档
```

## ⚙️ 配置说明

### gateway.yaml 核心配置

```
# 基础配置
name: "gateway-service"
host: "localhost"
port: 8080              # HTTP 端口
grpc_port: 9080         # gRPC 端口

# 路由配置：定义请求转发规则
routes:
  - name: "user-service"
    path: "/api/v1/user"
    service: "user-service"
    strip_path: true    # 剥离路径前缀
    timeout: "5s"       # 请求超时时间
  - name: "admin-service"
    path: "/api/v1/admin"
    service: "admin-service"
    strip_path: true
    timeout: "5s"
  - name: "ai-service"
    path: "/api/v1/ai"
    service: "ai-service"
    strip_path: true
    timeout: "5s"

# JWT 配置
jwt:
  secret: "your-secret-key"
  expire: 24h

# Consul 配置
consul:
  host: "localhost"
  port: 8500
  scheme: "http"
  ttl: "30s"                    # TTL 检查间隔
  keepalive_interval: "10s"     # 心跳保活间隔

# CORS 配置
service:
  cors_enabled: true
  public_apis:                  # 无需认证的公开接口
    - "/health"
  cors:
    allow_origins: ["*"]
    allow_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allow_headers: ["Origin", "Content-Type", "Authorization"]
    allow_credentials: true
    max_age: 12

# 限流配置
rate_limit:
  enabled: true                     # 是否启用限流
  requests_per_second: 100          # 每秒允许的请求数
  burst_size: 20                    # 令牌桶容量（允许突发流量）
  by_ip: true                       # 按 IP 限流
  by_api: false                     # 按 API 路径限流

# 熔断配置
circuit_breaker:
  enabled: true                     # 是否启用熔断
  max_failures: 5                   # 最大连续失败次数
  timeout: "30s"                    # 熔断器打开后的恢复等待时间
  min_requests: 10                  # 半开状态下的最小请求数
  interval: "60s"                   # 统计窗口时间

```

## 🚀 启动方式

### 前置条件

1. **Consul 必须运行**：Gateway 依赖 Consul 进行服务注册与发现
2. **后端服务已启动**：user-service、admin-service 等需先注册到 Consul

### 启动步骤

#### 1. 直接运行（开发模式）

```bash
cd gateway
go run cmd/server/main.go
```

#### 2. 指定配置文件

```bash
go run cmd/server/main.go -f etc/gateway.yaml
```

#### 3. 编译后运行（生产模式）

```bash
go build -o gateway.exe ./cmd/server
./gateway.exe -f etc/gateway.yaml
```

## 📡 API 端点

### 健康检查

```bash
GET http://localhost:8080/health
```

响应示例：
```
{
  "status": "healthy",
  "service": "gateway-service",
  "version": "1.0.0"
}
```

### 路由转发示例

Gateway 根据路由配置自动转发请求到后端服务：

```
# 用户服务（转发到 user-service:8001）
POST http://localhost:8080/api/v1/user/register
POST http://localhost:8080/api/v1/user/login
GET  http://localhost:8080/api/v1/user/list

# 管理服务（转发到 admin-service:8002）
POST http://localhost:8080/api/v1/admin/login
GET  http://localhost:8080/api/v1/admin/users

# AI 服务（转发到 ai-service）
POST http://localhost:8080/api/v1/ai/chat
```

### gRPC 接口

Gateway 同时提供 gRPC 接口，转发到后端 gRPC 服务：

- `UserService`: Register, Login, GetUserList
- `AdminService`: Login, Logout, GetAdminInfo, CreateAdmin, GetAdminList
- `AiService`: Chat, StreamChat, GetChatHistory, GetChatList

## 🔧 工作原理

### 1. 服务注册流程

Gateway 启动时执行以下步骤：

1. **加载配置**：读取 `gateway.yaml` 配置文件
2. **初始化依赖**：创建 Redis 连接、Consul Registry、gRPC Client Manager
3. **注册到 Consul**：
   - 注册 HTTP 服务（带 HTTP 健康检查）
   - 注册 gRPC 服务（带 gRPC 健康检查）
4. **启动心跳保活**：每 10 秒发送 TTL 心跳，保持服务活跃
5. **启动服务**：
   - HTTP 服务器监听 8080 端口
   - gRPC 服务器监听 9080 端口

### 2. HTTP 请求路由流程

当收到 HTTP 请求时：

1. **中间件处理**：
   - Recovery：捕获 panic，返回 500 错误
   - Logger：记录请求日志（方法、路径、状态码、IP）
   - CORS：设置跨域响应头
   - JWT Auth：验证 Token（公开接口跳过）

2. **路由匹配**：根据请求路径匹配路由配置
   - `/api/v1/user/*` → user-service
   - `/api/v1/admin/*` → admin-service
   - `/api/v1/ai/*` → ai-service

3. **服务发现**：从 Consul 获取目标服务的健康实例列表

4. **负载均衡**：使用 Round-Robin 算法选择一个实例

5. **路径处理**：
   - `strip_path: true`：剥离路径前缀
     - 请求：`/api/v1/user/list` → 转发：`/list`
   - `strip_path: false`：保留完整路径
     - 请求：`/api/v1/user/list` → 转发：`/api/v1/user/list`

6. **反向代理**：创建 `httputil.ReverseProxy` 转发请求

7. **超时控制**：根据配置的 timeout 设置请求超时

8. **返回响应**：将后端服务的响应返回给客户端

### 3. gRPC 请求转发流程

当收到 gRPC 请求时：

1. **服务发现**：从 Consul 获取目标 gRPC 服务实例（如 `user-service-grpc`）

2. **连接管理**：
   - 检查连接池中是否已有可用连接
   - 如无则创建新的 gRPC 连接并缓存

3. **负载均衡**：使用 Round-Robin 选择实例

4. **请求转发**：调用后端 gRPC 服务的方法

5. **返回响应**：将后端响应返回给客户端

### 4. 路径处理详解

#### Strip Path 模式

**配置**：`strip_path: true`

**场景**：后端服务不需要路径前缀

**示例**：
```
客户端请求：GET /api/v1/user/list
Gateway 转发：GET http://localhost:8001/list
```

#### 保留路径模式

**配置**：`strip_path: false`

**场景**：后端服务需要完整路径

**示例**：
```
客户端请求：GET /api/v1/user/list
Gateway 转发：GET http://localhost:8001/api/v1/user/list
```

### 5. 元数据管理

Gateway 从 Consul 读取各服务的元数据（metadata）：

- `public-apis`：公开接口列表（无需认证）
- `auth-required`：需要认证的接口
- `service-type`：服务类型标识
- `version`：服务版本号

这些元数据用于 JWT 认证、权限控制等功能。

## 🛠️ 技术栈

| 组件 | 技术选型 | 版本/说明 |
|------|---------|----------|
| **Web Framework** | Gin | 高性能 HTTP 框架 |
| **gRPC** | google.golang.org/grpc | gRPC 通信协议 |
| **Protocol Buffers** | protobuf | 接口定义语言 |
| **Service Discovery** | Consul | 服务注册与发现 |
| **Reverse Proxy** | net/http/httputil | 标准库反向代理 |
| **Configuration** | Viper + YAML | 配置管理 |
| **Database** | GORM + MySQL | ORM 框架（预留） |
| **Cache** | Redis | 缓存服务 |
| **JWT** | golang-jwt/jwt/v5 | Token 认证 |
| **Load Balancing** | Round-Robin | 负载均衡算法 |
| **Rate Limiting** | golang.org/x/time/rate | 令牌桶限流算法 |
| **Circuit Breaker** | sony/gobreaker | 断路器模式实现 |
| **Logging** | go.uber.org/zap | 高性能结构化日志 |
| **Log Rotation** | lumberjack.v2 | 日志文件滚动管理 |
| **Tracing** | OpenTelemetry + Jaeger | 分布式链路追踪 |

## 🔐 安全通信

### gRPC TLS/mTLS 配置

Gateway 支持分级安全策略，满足不同环境的安全需求。

#### 配置示例

```
grpc:
  use_tls: true                     # 是否启用 TLS（生产环境必须启用）
  insecure_skip_verify: false       # 跳过证书验证（仅用于开发/测试）
  cert_file: "/etc/certs/gateway/client.crt"     # 客户端证书
  key_file: "/etc/certs/gateway/client.key"      # 客户端私钥
  ca_file: "/etc/certs/gateway/ca.crt"           # CA 证书
  server_name: "gateway-service"    # TLS Server Name
```

#### 连接模式

| 模式 | 配置 | 适用场景 |
|------|------|---------|
| **Insecure** | `use_tls: false` | 开发环境、内部网络 |
| **单向 TLS** | `use_tls: true` + 不配置 cert_file/key_file | 服务器认证 |
| **双向 mTLS** | `use_tls: true` + 配置 cert_file/key_file | 生产环境、高安全要求 |

#### 证书自动推导

当证书路径未明确指定时，系统会自动使用默认路径：

```
/etc/certs/{service-name}/
├── client.crt    # 客户端证书
├── client.key    # 客户端私钥
└── ca.crt        # CA 证书
```

#### 证书健康检查

系统每 24 小时自动检查证书有效期，提前 7 天发出警告：

```
WARN  Certificate expiring soon: 6.5 days remaining
```

#### 最佳实践

1. **开发环境**：使用 `use_tls: false`，简化调试
2. **测试环境**：启用单向 TLS，验证证书链
3. **生产环境**：必须启用双向 mTLS，确保证书由统一 CA 签发
4. **证书轮换**：定期更新证书，避免过期导致服务中断

## 🔒 限流与熔断

### 限流中间件（Rate Limiting）

基于**令牌桶算法**实现请求速率控制，防止突发流量打垮后端服务。

#### 配置示例

```
rate_limit:
  enabled: true                     # 是否启用限流
  requests_per_second: 100          # 每秒允许的请求数
  burst_size: 20                    # 令牌桶容量（允许突发流量）
  by_ip: true                       # 按 IP 限流
  by_api: false                     # 按 API 路径限流
```

#### 工作原理

1. **令牌桶初始化**：每个 IP（或 API）拥有一个独立的令牌桶
2. **令牌生成**：以 `requests_per_second` 的速率持续生成令牌
3. **令牌消费**：每个请求消耗一个令牌
4. **限流触发**：当桶内无令牌时，返回 HTTP 429

#### 响应示例

```
{
  "code": 429,
  "message": "Too many requests, please try again later"
}
```

#### 限流策略

- **按 IP 限流**（推荐）：防止单个用户滥用资源
- **按 API 限流**：保护特定热点接口
- **全局限流**：控制整体流量（适合小规模服务）

---

### 熔断中间件（Circuit Breaker）

基于**断路器模式**实现故障隔离，防止雪崩效应。

#### 配置示例

```
circuit_breaker:
  enabled: true                     # 是否启用熔断
  max_failures: 5                   # 最大连续失败次数
  timeout: "30s"                    # 熔断器打开后的恢复等待时间
  min_requests: 10                  # 半开状态下的最小请求数
  interval: "60s"                   # 统计窗口时间
```

#### 三状态机

```
┌──────────┐   失败次数 > max_failures   ┌──────────┐
│  Closed  │ ──────────────────────────→ │   Open   │
│ (正常)   │                             │ (熔断)   │
└──────────┘                             └────┬─────┘
     ↑                                        │
     │           timeout 到期                 │
     │      ┌──────────────┐                  │
     └──────│ Half-Open    │←─────────────────┘
            │ (半开探测)   │
            └──────────────┘
            成功 → Closed
            失败 → Open
```

#### 工作流程

1. **Closed（关闭）**：正常状态，所有请求通过
2. **Open（打开）**：失败次数超过阈值，拒绝所有请求（返回 503）
3. **Half-Open（半开）**：等待超时后进入，允许少量请求探测
   - 成功 → 回到 Closed
   - 失败 → 回到 Open

#### 响应示例

```
{
  "code": 503,
  "message": "Service temporarily unavailable, circuit breaker is open"
}
```

#### 应用场景

- **后端服务宕机**：快速失败，避免长时间等待
- **网络抖动**：临时隔离不稳定服务
- **依赖故障**：防止级联失败

---

### 中间件注册顺序

```
engine.Use(middleware.Recovery())         // 1. 异常恢复
engine.Use(middleware.Logger())           // 2. 日志记录
engine.Use(middleware.CORS(...))          // 3. CORS 跨域
engine.Use(middleware.RateLimit(...))     // 4. ⭐ 限流（在 JWT 之前）
engine.Use(middleware.JWTAuth(...))       // 5. JWT 认证
engine.Use(middleware.CircuitBreaker(...))// 6. ⭐ 熔断（在 JWT 之后）
```

**设计理由**：
- **限流在 JWT 之前**：防止未认证请求消耗系统资源
- **熔断在 JWT 之后**：只保护已认证的后端调用，避免误伤

---

## 📝 日志系统

### 结构化日志（Structured Logging）

Gateway 使用 **zap** 日志库实现高性能结构化日志，支持 JSON 格式输出和文件滚动。

#### 配置示例

```yaml
logger:
  level: "info"                     # 日志级别：debug, info, warn, error
  format: "json"                    # 日志格式：json, console
  filename: "logs/gateway.log"      # 日志文件路径
  max_size: 100                     # 单个日志文件最大大小（MB）
  max_backups: 10                   # 保留的旧日志文件数量
  max_age: 30                       # 日志文件保留天数
  compress: true                    # 是否压缩旧日志文件
```

#### 日志输出示例

**JSON 格式（生产环境推荐）**：
```
{
  "level": "info",
  "ts": "2026-06-01T16:30:00.000Z",
  "caller": "middleware/logger.go:45",
  "msg": "HTTP Request",
  "method": "POST",
  "path": "/api/v1/user/login",
  "query": "",
  "status": 200,
  "duration": 0.045,
  "ip": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "user_id": 12345
}
```

**控制台格式（开发环境）**：
```
2026-06-01T16:30:00.000Z  INFO  middleware/logger.go:45  HTTP Request  {"method": "POST", "path": "/api/v1/user/login", "status": 200, "duration": 0.045}
```

#### 日志字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `level` | string | 日志级别（info/warn/error） |
| `ts` | string | ISO8601 时间戳 |
| `caller` | string | 代码位置（文件:行号） |
| `msg` | string | 日志消息 |
| `method` | string | HTTP 方法 |
| `path` | string | 请求路径 |
| `status` | int | HTTP 状态码 |
| `duration` | float | 请求耗时（秒） |
| `ip` | string | 客户端 IP |
| `user_agent` | string | 用户代理 |
| `user_id` | int/null | 用户 ID（已认证时） |

#### 日志级别使用规范

- **DEBUG**：详细的调试信息（开发环境）
- **INFO**：正常业务流程（请求处理、服务注册等）
- **WARN**：警告信息（客户端错误 4xx、配置问题等）
- **ERROR**：错误信息（服务端错误 5xx、数据库连接失败等）

#### 日志文件管理

**自动滚动策略**：
```
logs/
├── gateway.log              # 当前日志文件
├── gateway-2026-06-01.log   # 历史日志（按日期分割）
├── gateway-2026-05-31.log.gz # 压缩的历史日志
└── ...
```

**滚动条件**（满足任一即触发）：
1. 文件大小超过 `max_size`（默认 100 MB）
2. 每天凌晨自动轮转
3. 保留最近 `max_backups` 个文件（默认 10 个）
4. 超过 `max_age` 天的文件自动删除（默认 30 天）

#### 集中式日志收集（可选）

**方案 1：Loki + Promtail（推荐）**

```
# docker-compose.yml
version: '3'
services:
  loki:
    image: grafana/loki:latest
    ports:
      - "3100:3100"
  
  promtail:
    image: grafana/promtail:latest
    volumes:
      - ./logs:/var/log/gateway
    command: >
      -config.file=/etc/promtail/config.yml
      -client.url=http://loki:3100/loki/api/v1/push
  
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
```

**方案 2：ELK Stack**

```
# Filebeat 配置
filebeat.inputs:
  - type: log
    paths:
      - /var/log/gateway/*.log
    json.keys_under_root: true
    json.add_error_key: true

output.elasticsearch:
  hosts: ["http://elasticsearch:9200"]
```

---

## 🔍 链路追踪系统

### 分布式追踪（Distributed Tracing）

Gateway 使用 **OpenTelemetry + Jaeger** 实现跨服务调用链追踪，支持 HTTP 和 gRPC 协议的 Trace Context 传递。

#### 配置示例

```
tracing:
  enabled: true                     # 是否启用链路追踪
  service_name: "gateway-service"   # 服务名称
  endpoint: "http://localhost:14268/api/traces"  # Jaeger Collector 地址
  sampler_type: "const"             # 采样器类型：const, probabilistic, ratelimiting
  sampler_param: 1.0                # 采样参数（const: 0/1, probabilistic: 0.0-1.0）
```

#### 采样器类型说明

| 类型 | 说明 | 适用场景 |
|------|------|---------|
| `const` | 常量采样：0（不采样）或 1（全采样） | 开发环境、低流量服务 |
| `probabilistic` | 概率采样：0.0-1.0（如 0.1 = 10%） | 生产环境、高流量服务 |
| `ratelimiting` | 速率限制采样：每秒固定数量 | 需要控制成本的场景 |

#### 工作原理

**HTTP 请求追踪流程**：
```
Client → Gateway (创建 Span) → User Service (提取 Context，创建子 Span) → Database
         ↓                          ↓
    Trace ID: abc123           Trace ID: abc123
    Span ID: span-1            Parent Span ID: span-1
                               Span ID: span-2
```

**gRPC 请求追踪流程**：
```
Gateway (UnaryServerInterceptor) → User Service (UnaryClientInterceptor)
         ↓                                ↓
    提取 Metadata                    注入 Metadata
    创建 Server Span                 创建 Client Span
```

#### Trace Context 传递

**HTTP 协议**（W3C Trace Context）：
```http
GET /api/v1/user/list HTTP/1.1
traceparent: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01
```

**gRPC 协议**（Metadata）：
```
metadata: {
  key: "traceparent"
  value: "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"
}
```

#### Jaeger UI 查看追踪

**启动 Jaeger**：
```bash
docker run -d --name jaeger \
  -e COLLECTOR_ZIPKIN_HOST_PORT=:9411 \
  -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 14250:14250 \
  -p 9411:9411 \
  jaegertracing/all-in-one:latest
```

**访问 Jaeger UI**：
```
http://localhost:16686
```

**查询追踪**：
1. 选择服务：`gateway-service`
2. 输入 Trace ID（从响应头 `X-Trace-ID` 获取）
3. 查看完整的调用链拓扑图

#### Span 属性说明

**HTTP Span**：
- `http.method`: HTTP 方法（GET/POST）
- `http.url`: 完整 URL
- `http.path`: 请求路径
- `http.status_code`: 响应状态码
- `client.ip`: 客户端 IP
- `user_agent`: 用户代理

**gRPC Span**：
- `grpc.method`: gRPC 方法名（如 `/user.UserService/Login`）
- `error`: 是否发生错误

#### 性能影响

- **开销**：< 1ms/请求（异步批量导出）
- **内存占用**：每个 Span 约 500 bytes
- **网络流量**：取决于采样率（生产环境建议 10%-50%）

#### 最佳实践

1. **开发环境**：使用 `const` 采样器，`sampler_param: 1.0`（全采样）
2. **生产环境**：使用 `probabilistic` 采样器，`sampler_param: 0.1`（10% 采样）
3. **关键接口**：单独设置更高的采样率
4. **错误追踪**：所有错误请求都会被记录，不受采样率影响

#### 与日志系统集成

**关联 Trace ID 和日志**：
```json
{
  "level": "info",
  "msg": "HTTP Request",
  "trace_id": "abc123...",  // ← 从 Span 获取
  "method": "POST",
  "path": "/api/v1/user/login"
}
```

**在 Grafana/Loki 中查询**：
``logql
{job="gateway"} |= "abc123"
```

## 📝 开发规范

```
