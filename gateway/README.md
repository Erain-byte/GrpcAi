
# Gateway 项目结构说明

## 目录总览

```
gateway/
├── api/                  # Proto 文件定义
│   └── api.proto
├── cmd/                  # 程序入口
│   └── server/
│       └── main.go
├── etc/                  # 配置文件
│   └── gateway.yaml
├── internal/             # 内部包（不可外部引用）
│   ├── config/           # 配置结构定义
│   ├── handler/          # gRPC/HTTP 请求处理
│   ├── logic/            # 业务逻辑
│   ├── middleware/        # 中间件
│   ├── server/           # 服务注册与启动
│   ├── svc/              # 服务上下文（依赖注入）
│   └── types/            # 类型定义
├── pkg/                  # 可外部引用的公共包
└── go.mod                # Go 模块文件
```

## 目录详细说明

### api/
存放 `.proto` 文件，定义 gRPC 服务接口和消息类型。通过 `protoc` 编译器生成对应的 Go 代码。

- **api.proto**: 主 Proto 定义文件，包含服务（Service）和消息（Message）定义

### cmd/
程序入口目录，包含服务的启动代码。

- **server/main.go**: 应用程序主入口，负责初始化配置、注册服务并启动 gRPC/HTTP 服务

### etc/
存放配置文件，通常使用 YAML 格式。

- **gateway.yaml**: 主配置文件，包含服务端口、数据库连接、Redis 配置、依赖服务地址等

### internal/
内部包目录，其下的代码不可被外部模块引用，保证内部实现封装性。

#### internal/config/
配置结构定义，将 YAML 配置文件映射为 Go 结构体。

- **config.go**: 定义 `Config` 结构体，与 `etc/gateway.yaml` 一一对应

#### internal/handler/
请求处理层，负责接收 gRPC/HTTP 请求并调用对应的 Logic 层处理。

- **handler.go**: 注册路由，将请求分发到对应的 Logic 处理函数

#### internal/logic/
业务逻辑层，核心业务代码均在此编写，与请求处理解耦。

- **logic.go**: 实现具体的业务逻辑，调用 svc 中的服务上下文完成数据处理

#### internal/middleware/
中间件层，用于请求的前置/后置处理。

- **middleware.go**: 定义中间件，如鉴权、日志、限流、链路追踪等

#### internal/server/
服务注册与启动，定义 gRPC Server 或 HTTP Server。

- **server.go**: 注册 gRPC 服务实现，绑定服务到端口

#### internal/svc/
服务上下文，管理所有依赖资源，实现依赖注入。

- **servicecontext.go**: 定义 `ServiceContext` 结构体，包含数据库连接、Redis 客户端、其他 gRPC 客户端等依赖

#### internal/types/
类型定义，存放请求/响应等自定义类型。

- **types.go**: 定义请求（Request）和响应（Response）结构体

### pkg/
公共包目录，可被外部模块引用，存放通用工具函数和公共逻辑。

- **pkg.go**: 通用工具代码，如错误码、响应封装、加密解密等

### go.mod
Go 模块依赖管理文件，定义模块路径和第三方依赖版本。

## 调用流程

```
请求 → handler → middleware → logic → svc（依赖资源）→ 响应
```

1. **handler** 接收请求，提取参数
2. **middleware** 进行鉴权、日志等前置处理
3. **logic** 执行核心业务逻辑
4. **svc** 提供数据库、缓存、RPC 客户端等依赖
5. **logic** 组装响应返回给 handler
6. **handler** 返回最终响应

## 开发规范

1. 新增接口时，先在 `api/` 下定义 Proto 文件，再生成代码
2. 业务逻辑必须写在 `internal/logic/` 中，handler 层不写业务代码
3. 所有依赖资源通过 `internal/svc/` 统一管理
4. 公共工具函数放在 `pkg/` 下，内部专用工具放在 `internal/` 下
5. 配置项必须同时在 `etc/gateway.yaml` 和 `internal/config/config.go` 中定义
