# Admin 项目结构说明

## 项目概述

Admin 是一个基于 **Gin + GORM** 的后台管理服务，采用经典分层架构，职责清晰，便于维护和扩展。

## 目录结构

```
admin/
├── api/                    # API 定义层
│   └── admin.proto         # Protobuf 接口定义文件，定义 gRPC 服务接口
├── cmd/                    # 程序入口
│   └── server/
│       └── main.go         # 主入口文件，启动服务
├── etc/                    # 配置文件
│   └── admin.yaml          # 应用配置文件（数据库、端口等）
├── internal/               # 内部逻辑（不对外暴露）
│   ├── config/             # 配置加载
│   │   └── config.go       # 定义配置结构体，读取并解析 admin.yaml
│   ├── handler/            # 处理层（控制器）
│   │   └── handler.go      # 接收 HTTP 请求，参数校验，调用 logic 层
│   ├── logic/              # 业务逻辑层
│   │   └── logic.go        # 核心业务逻辑处理，与数据库交互
│   ├── middleware/          # 中间件
│   │   └── middleware.go   # Gin 中间件（鉴权、日志、跨域等）
│   ├── server/             # 服务启动层
│   │   └── server.go       # 初始化并启动 HTTP/gRPC 服务
│   ├── svc/                # 服务上下文
│   │   └── servicecontext.go  # 依赖注入容器，持有 DB、Redis 等连接
│   └── types/              # 类型定义
│       └── types.go        # 请求/响应结构体等公共类型定义
├── pkg/                    # 公共工具包（可对外暴露）
│   └── pkg.go              # 通用工具函数（加密、响应封装等）
├── go.mod                  # Go 模块依赖管理
└── go.sum                  # 依赖校验文件
```

## 分层架构说明

```
请求流程：HTTP Request → Middleware → Handler → Logic → Database
```

| 层级          | 目录          | 职责                                      |
| ------------- | ------------- | ----------------------------------------- |
| **入口层**    | `cmd/server`  | 程序启动入口，初始化配置和依赖            |
| **配置层**    | `etc/`        | 存放 YAML 配置文件                        |
| **配置加载**  | `internal/config` | 解析配置文件为 Go 结构体               |
| **中间件层**  | `internal/middleware` | 请求拦截处理（鉴权、日志、CORS 等） |
| **处理层**    | `internal/handler` | 接收请求、参数校验、调用 logic、返回响应 |
| **业务逻辑层**| `internal/logic` | 核心业务逻辑，数据操作与处理           |
| **服务上下文**| `internal/svc` | 依赖注入，管理 DB、Redis 等共享资源     |
| **类型定义**  | `internal/types` | 请求/响应结构体定义                     |
| **服务启动**  | `internal/server` | 注册路由，启动 HTTP/gRPC 服务          |
| **API 定义**  | `api/`        | Protobuf 接口定义文件                     |
| **公共工具**  | `pkg/`        | 可复用的通用工具包                        |

## 技术栈

- **Web 框架**：Gin
- **ORM 框架**：GORM
- **数据库驱动**：MySQL
- **配置格式**：YAML
- **接口定义**：Protobuf（gRPC）

## 开发规范

1. **Handler 层**只做参数校验和转发，不包含业务逻辑
2. **Logic 层**是业务逻辑的唯一实现位置
3. **Svc 层**统一管理依赖，通过依赖注入传递给 Logic
4. **Types 层**定义所有请求和响应的结构体
5. **Pkg 层**放置可被外部项目引用的通用工具
