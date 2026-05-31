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
├── etc/gateway.yaml                # 配置文件：路由、Consul、JWT、CORS 等
├── internal/
│   ├── config/config.go            # 配置加载：解析 YAML 为 Go 结构体
│   ├── grpc/client.go              # gRPC 客户端管理：连接池、服务发现、负载均衡
│   ├── handler/handler.go          # HTTP 处理层（预留）
│   ├── logic/logic.go              # 业务逻辑层（预留）
│   ├── middleware/
│   │   ├── middleware.go           # 中间件：CORS、Recovery、Logger
│   │   └── jwt.go                  # JWT 认证中间件
│   ├── registry/registry.go        # Consul 服务注册与发现
│   ├── server/
│   │   ├── base.go                 # 泛型转发器基类 BaseForwarder[T]
│   │   ├── userGrpc.go             # User 服务转发器实现
│   │   ├── adminGrpc.go            # Admin 服务转发器实现
│   │   ├── aiGrpc.go               # AI 服务转发器实现
│   │   ├── register.go             # ⭐ gRPC 服务统一注册中心
│   │   └── server.go               # HTTP 服务器和反向代理
│   ├── svc/servicecontext.go       # 依赖注入容器：Redis、Registry、GrpcClients
│   └── types/types.go              # 类型定义
├── pkg/pkg.go                      # 公共工具包
├── go.mod                          # Go 模块依赖
├── go.sum                          # 依赖校验文件
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

## 📝 开发规范

1. **零硬编码原则**：所有配置（路由、超时、CORS 等）从 YAML 文件读取
2. **分层清晰**：Handler → Logic → Svc，职责明确
3. **错误处理**：所有 error 必须处理，不得忽略
4. **日志记录**：关键操作（注册、发现、转发）打印日志
5. **优雅关闭**：按顺序停止心跳 → 注销服务 → 关闭 gRPC → 关闭 HTTP
6. **连接复用**：gRPC 连接使用连接池，避免频繁创建
7. **超时控制**：所有外部调用必须设置超时时间
8. **服务注册集中化**：所有 gRPC 服务注册逻辑必须在 `register.go` 中统一管理

**最后更新**: 2026-05-31  
**维护者**: goGit Team
