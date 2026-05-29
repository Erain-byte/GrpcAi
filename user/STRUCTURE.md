# User 项目结构说明

## 项目概述

User 是一个基于 **Gin + GORM + gRPC** 的用户微服务，采用经典分层架构，支持 HTTP 和 gRPC 双协议访问，集成 Consul 服务注册与发现，具备 TTL 心跳保活和优雅关闭能力。

## 目录结构

```
user/
├── api/                    # API 定义层
│   ├── user.proto          # Protobuf 接口定义文件，定义 gRPC 服务接口
│   ├── user.pb.go          # Protobuf 自动生成的消息类型代码
│   └── user_grpc.pb.go     # Protobuf 自动生成的 gRPC 服务端/客户端代码
├── cmd/                    # 程序入口
│   └── server/
│       └── main.go         # 主入口文件，启动 HTTP/gRPC 服务、Consul 注册、优雅关闭
├── etc/                    # 配置文件
│   └── user.yaml           # 应用配置文件（数据库、Redis、JWT、Consul 等）
├── internal/               # 内部逻辑（不对外暴露）
│   ├── config/             # 配置加载
│   │   └── config.go       # 定义配置结构体（DB、Redis、JWT、Consul、Logger），读取并解析 user.yaml
│   ├── handler/            # 处理层（控制器）
│   │   └── handler.go      # 接收 HTTP 请求，参数校验，调用 logic 层
│   ├── logic/              # 业务逻辑层
│   │   └── logic.go        # 核心业务逻辑处理（注册、登录、用户列表），与数据库交互
│   ├── middleware/          # 中间件
│   │   └── middleware.go    # Gin 中间件（JWT 鉴权、CORS 跨域、Recovery 异常恢复）
│   ├── registry/           # 服务注册
│   │   └── registry.go     # Consul 服务注册/注销，TTL 心跳保活
│   ├── server/             # 服务启动层
│   │   ├── server.go       # 初始化 Gin 引擎，注册路由，应用中间件
│   │   └── grpc.go         # gRPC 服务实现，适配 UserServiceServer 接口
│   ├── svc/                # 服务上下文
│   │   └── servicecontext.go  # 依赖注入容器，持有 DB、Redis 等连接
│   └── types/              # 类型定义
│       └── types.go        # 请求/响应结构体等公共类型定义
├── pkg/                    # 公共工具包（可对外暴露）
│   └── pkg.go              # 通用工具函数（JWT 生成/解析、响应封装等）
├── go.mod                  # Go 模块依赖管理
└── go.sum                  # 依赖校验文件
```

## 分层架构说明

```
请求流程：
  HTTP:  Request → CORS → Recovery → [JWTAuth] → Handler → Logic → Database
  gRPC:  Request → UserServiceServer → Logic → Database
```

| 层级            | 目录                  | 职责                                                |
| --------------- | --------------------- | --------------------------------------------------- |
| **入口层**      | `cmd/server`          | 程序启动入口，初始化配置、JWT、服务注册、优雅关闭   |
| **配置层**      | `etc/`                | 存放 YAML 配置文件                                  |
| **配置加载**    | `internal/config`     | 解析配置文件为 Go 结构体（DB、Redis、JWT、Consul）  |
| **中间件层**    | `internal/middleware` | 请求拦截处理（JWT 鉴权、CORS 跨域、Recovery 恢复） |
| **处理层**      | `internal/handler`    | 接收 HTTP 请求、参数校验、调用 logic、返回响应      |
| **业务逻辑层**  | `internal/logic`      | 核心业务逻辑，数据操作与处理                        |
| **服务注册**    | `internal/registry`   | Consul 服务注册/注销，TTL 心跳保活                  |
| **服务启动**    | `internal/server`     | 注册路由（HTTP）、实现 gRPC 接口                    |
| **服务上下文**  | `internal/svc`        | 依赖注入，管理 DB、Redis 等共享资源                 |
| **类型定义**    | `internal/types`      | 请求/响应结构体定义                                 |
| **API 定义**    | `api/`                | Protobuf 接口定义及自动生成代码                     |
| **公共工具**    | `pkg/`                | 可复用的通用工具包（JWT、响应封装）                 |

## 路由设计

### HTTP 路由

| 方法   | 路径                  | 说明         | 认证   |
| ------ | --------------------- | ------------ | ------ |
| GET    | `/health`             | 健康检查     | 无     |
| POST   | `/api/v1/user/register` | 用户注册   | 无     |
| POST   | `/api/v1/user/login`  | 用户登录     | 无     |
| GET    | `/api/v1/user/list`   | 获取用户列表 | JWT    |

### gRPC 服务

| 方法           | 说明         |
| -------------- | ------------ |
| `Register`     | 用户注册     |
| `Login`        | 用户登录     |
| `GetUserList`  | 获取用户列表 |

## 服务注册与发现

- **注册中心**：Consul
- **注册内容**：HTTP 服务（端口 8001）+ gRPC 服务（端口 9001）
- **健康检查**：
  - HTTP：`/health` 端点主动检查（10s 间隔）
  - gRPC：gRPC 健康检查协议（10s 间隔）
  - TTL：30s 心跳保活，每 10s 发送一次
- **自动注销**：服务不可用 90s 后自动从 Consul 注销
- **优雅关闭**：收到信号后停止心跳 → 注销服务 → 关闭 HTTP → 关闭 gRPC

## 配置说明

```yaml
name: "user-service"       # 服务名称
port: 8001                  # HTTP 端口
grpc_port: 9001             # gRPC 端口

database:                   # MySQL 数据库配置
  host: "localhost"
  port: 3306
  username: "root"
  password: "123456"
  dbname: "user_db"

redis:                      # Redis 配置
  host: "localhost"
  port: 6379

jwt:                        # JWT 配置
  secret: "your-secret-key"
  expire: 24h

consul:                     # Consul 配置
  host: "localhost"
  port: 8500
  token: ""

logger:                     # 日志配置
  level: "info"
  format: "json"
```

## 技术栈

- **Web 框架**：Gin
- **ORM 框架**：GORM
- **数据库**：MySQL
- **缓存**：Redis
- **配置格式**：YAML（Viper）
- **接口定义**：Protobuf（gRPC）
- **服务注册**：Consul
- **认证方式**：JWT（HS256）

## 开发规范

1. **Handler 层**只做参数校验和转发，不包含业务逻辑
2. **Logic 层**是业务逻辑的唯一实现位置
3. **Svc 层**统一管理依赖，通过依赖注入传递给 Logic
4. **Types 层**定义所有请求和响应的结构体
5. **Pkg 层**放置可被外部项目引用的通用工具
6. **Registry 层**负责服务注册与发现，与 Consul 交互
7. **公开接口**（注册、登录）无需认证，**受保护接口**需通过 JWT 中间件验证
8. **优雅关闭**：先停止心跳 → 注销服务 → 关闭 HTTP → 关闭 gRPC，确保请求处理完毕

