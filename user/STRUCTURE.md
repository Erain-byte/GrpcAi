# User Service

用户微服务，提供用户注册、登录、查询等核心功能，支持 HTTP REST API 和 gRPC 双协议访问。

## 📋 功能特性

- ✅ **用户注册**：用户名唯一性校验，密码 bcrypt 加密存储
- ✅ **用户登录**：验证用户名密码，生成 JWT Token
- ✅ **用户列表**：分页查询用户信息（需要 JWT 认证）
- ✅ **双协议支持**：同时提供 HTTP REST API 和 gRPC 接口
- ✅ **JWT 认证**：基于 HS256 算法的 Token 认证机制
- ✅ **服务注册**：自动注册到 Consul 并维持 TTL 心跳
- ✅ **优雅关闭**：支持平滑停机，确保请求处理完毕
- ✅ **Redis 缓存**：支持单节点和集群模式
- ✅ **健康检查**：提供 `/health` 端点

## 🏗️ 架构设计

```
Client Request (HTTP/gRPC)
      ↓
┌─────────────────────────────┐
│   User Service              │
│   HTTP: 8001                │
│   gRPC: 9001                │
│                             │
│  • Middleware Layer         │
│    - JWT Authentication     │
│    - Recovery               │
│                             │
│  • Handler Layer            │
│    - Parameter Validation   │
│    - Request/Response       │
│                             │
│  • Logic Layer              │
│    - Business Logic         │
│    - Password Encryption    │
│    - Token Generation       │
│                             │
│  • Data Access Layer        │
│    - GORM (MySQL)           │
│    - Redis                  │
└─────────────────────────────┘
      ↓
┌──────────┐    ┌──────────┐
│  MySQL   │    │  Redis   │
│ user_db  │    │ Cache    │
└──────────┘    └──────────┘
      ↓
┌──────────┐
│  Consul  │
│ Registry │
└──────────┘
```

## 📁 目录结构

```
user/
├── api/                        # API 定义层
│   ├── user.proto              # Protobuf 接口定义
│   ├── user.pb.go              # Protobuf 生成的消息类型
│   └── user_grpc.pb.go         # Protobuf 生成的 gRPC 代码
├── cmd/server/main.go          # 入口文件：启动 HTTP + gRPC、Consul 注册
├── etc/user.yaml               # 配置文件：数据库、Redis、JWT、Consul
├── internal/
│   ├── config/config.go        # 配置加载：解析 YAML 为 Go 结构体
│   ├── handler/handler.go      # HTTP 处理层：参数校验、调用 Logic
│   ├── logic/logic.go          # 业务逻辑层：注册、登录、查询
│   ├── middleware/middleware.go # 中间件：JWT Auth、Recovery
│   ├── registry/registry.go    # Consul 服务注册与发现
│   ├── server/
│   │   ├── server.go           # HTTP 路由注册
│   │   └── grpc.go             # gRPC 服务实现
│   ├── svc/servicecontext.go   # 依赖注入：DB、Redis
│   └── types/types.go          # 数据模型和请求/响应结构体
├── pkg/pkg.go                  # 公共工具：JWT 生成/解析、响应封装
├── go.mod                      # Go 模块依赖
├── go.sum                      # 依赖校验文件
└── STRUCTURE.md                # 项目结构说明
```

## ⚙️ 配置说明

### user.yaml 核心配置

```yaml
# 基础配置
name: "user-service"
host: "localhost"
port: 8001              # HTTP 端口
grpc_port: 9001         # gRPC 端口

# 数据库配置
database:
  driver: "mysql"
  host: "localhost"
  port: 3306
  username: "root"
  password: "123456"
  dbname: "user_db"
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600

# Redis 配置（支持单节点/集群）
redis:
  host: "localhost"
  port: 6379
  cluster_addresses: []  # 集群模式时配置地址列表
  password: ""
  db: 0
  pool_size: 100

# JWT 配置
jwt:
  secret: "your-secret-key"
  expire: 24h

# Consul 配置
consul:
  host: "localhost"
  port: 8500
  ttl: "30s"
  keepalive_interval: "10s"

# 服务配置
service:
  version: "1.0.0"
  cors_enabled: true
  public_apis:           # 公开接口（无需认证）
    - "/api/v1/user/register"
    - "/api/v1/user/login"
    - "/health"
  auth_apis:            # 受保护接口（需要认证）
    - "/api/v1/user/list"
  cors:
    allow_origins: ["*"]
    allow_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allow_headers: ["Origin", "Content-Type", "Authorization"]
    allow_credentials: true
    max_age: 12

# gRPC TLS/mTLS 配置（预留配置，暂未使用）
# 注意：User 服务当前仅作为 gRPC 服务端，此配置段为未来扩展预留
# 如需在服务端启用 TLS，需要修改 internal/server/grpc.go 加载证书
grpc:
  use_tls: false                    # 是否启用 TLS（生产环境必须启用）
  insecure_skip_verify: false       # 跳过证书验证（仅用于开发/测试）
  cert_file: ""                     # 服务端证书路径（默认：/etc/certs/user-service/server.crt）
  key_file: ""                      # 服务端私钥路径（默认：/etc/certs/user-service/server.key）
  ca_file: ""                       # CA 证书路径（默认：/etc/certs/user-service/ca.crt）
  server_name: "user-service"       # TLS Server Name

## 🔐 安全通信

User 服务支持 gRPC TLS/mTLS 加密通信，确保服务间数据传输的安全性。

### 连接模式

| 模式 | 配置 | 适用场景 |
|------|------|---------|
| **Insecure** | `use_tls: false` | 开发环境、内部网络 |
| **单向 TLS** | `use_tls: true` + 不配置 cert_file/key_file | 服务器认证 |
| **双向 mTLS** | `use_tls: true` + 配置 cert_file/key_file | 生产环境、高安全要求 |

### 证书路径规范

遵循统一规范：`/etc/certs/{service-name}/`

```bash
/etc/certs/user-service/
├── client.crt    # 客户端证书
├── client.key    # 客户端私钥
└── ca.crt        # CA 证书
```

### 生产环境配置示例

```
grpc:
  use_tls: true
  insecure_skip_verify: false
  cert_file: "/etc/certs/user-service/client.crt"
  key_file: "/etc/certs/user-service/client.key"
  ca_file: "/etc/certs/user-service/ca.crt"
  server_name: "user-service"
```

📖 **详细文档**：参考 [Gateway 安全通信指南](../gateway/README.md#-安全通信)
