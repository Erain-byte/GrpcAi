# Go 微服务开发 SKILL

## 一、项目结构规范

```
{service}/
├── api/                        # Protobuf 接口定义及生成代码
│   ├── {service}.proto
│   ├── {service}.pb.go
│   └── {service}_grpc.pb.go
├── cmd/
│   └── server/
│       └── main.go             # 入口：配置初始化 → 组件初始化 → 服务注册 → 启动 → 优雅关闭
├── etc/
│   └── {service}.yaml          # 所有可配置项，禁止硬编码
├── internal/
│   ├── config/config.go        # 配置结构体 + 加载逻辑
│   ├── handler/handler.go      # HTTP处理层，只做参数校验和转发
│   ├── logic/logic.go          # 业务逻辑层，唯一实现位置
│   ├── middleware/middleware.go # 中间件（JWT、CORS、Recovery）
│   ├── registry/registry.go    # 服务注册（Consul）
│   ├── server/
│   │   ├── server.go           # Gin引擎 + 路由注册
│   │   └── grpc.go             # gRPC服务实现
│   ├── svc/servicecontext.go   # 依赖注入容器
│   └── types/types.go          # 请求/响应结构体
├── pkg/pkg.go                  # 公共工具（JWT、响应封装）
├── go.mod
└── go.sum
```

## 二、配置规范（零硬编码原则）

### 2.1 所有值必须可配置
以下内容**禁止**硬编码，必须从配置文件读取：

| 类别         | 配置项                          | 默认值        |
| ------------ | ------------------------------- | ------------- |
| 服务         | name, host, port, grpc_port     | -             |
| 数据库       | driver, host, port, username, password, dbname, max_idle_conns, max_open_conns, conn_max_lifetime | - |
| Redis        | host, port, cluster_addresses, password, db, pool_size | -       |
| JWT          | secret, expire, issuer          | -             |
| Consul       | host, port, addresses, token, scheme       | -             |
| Consul健康检查 | check_interval, check_timeout, ttl, deregister_critical_after, keepalive_interval | 10s, 5s, 30s, 90s, 10s |
| 日志         | level, format                   | info, json    |
| 优雅关闭     | shutdown_timeout                | 5s            |

### 2.2 配置结构体模板

```go
type Config struct {
    Name     string         `yaml:"name"`
    Host     string         `yaml:"host" default:"localhost"`
    Port     int            `yaml:"port"`
    GRPCPort int            `yaml:"grpc_port"`
    Consul   ConsulConfig   `yaml:"consul"`
    Database DBConfig       `yaml:"database"`
    Redis    RedisConfig    `yaml:"redis"`
    JWT      JWTConfig      `yaml:"jwt"`
    Logger   LoggerConfig   `yaml:"logger"`
    Shutdown ShutdownConfig `yaml:"shutdown"`
}

type ConsulConfig struct {
    Addresses               []string `yaml:"addresses"`  // 集群地址列表，如 ["192.168.1.1:8500", "192.168.1.2:8500"]
    Host                    string   `yaml:"host" default:"localhost"`   // 单节点地址（addresses为空时使用）
    Port                    int      `yaml:"port" default:"8500"`        // 单节点端口（addresses为空时使用）
    Token                   string   `yaml:"token" default:""`
    Scheme                  string   `yaml:"scheme" default:"http"`      // 支持 http / https
    CheckInterval           string   `yaml:"check_interval" default:"10s"`
    CheckTimeout            string   `yaml:"check_timeout" default:"5s"`
    TTL                     string   `yaml:"ttl" default:"30s"`
    DeregisterCriticalAfter string   `yaml:"deregister_critical_after" default:"90s"`
    KeepAliveInterval       string   `yaml:"keepalive_interval" default:"10s"`
}

// GetAddresses 获取Consul地址列表，优先使用集群地址
func (c *ConsulConfig) GetAddresses() []string {
    if len(c.Addresses) > 0 {
        return c.Addresses
    }
    return []string{fmt.Sprintf("%s:%d", c.Host, c.Port)}
}

type RedisConfig struct {
    Host             string   `yaml:"host" default:"localhost"`     // 单节点地址（cluster_addresses为空时使用）
    Port             int      `yaml:"port" default:"6379"`          // 单节点端口（cluster_addresses为空时使用）
    ClusterAddresses []string `yaml:"cluster_addresses"`            // 集群地址列表
    Password         string   `yaml:"password" default:""`
    DB               int      `yaml:"db" default:"0"`
    PoolSize         int      `yaml:"pool_size" default:"100"`
}

// IsCluster 是否集群模式
func (r *RedisConfig) IsCluster() bool {
    return len(r.ClusterAddresses) > 0
}

// GetAddr 获取单节点地址
func (r *RedisConfig) GetAddr() string {
    return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type JWTConfig struct {
    Secret string `yaml:"secret"`
    Expire string `yaml:"expire" default:"24h"`
    Issuer string `yaml:"issuer"`  // 通常等于服务名
}

type ShutdownConfig struct {
    Timeout string `yaml:"timeout" default:"5s"`
}
```

### 2.3 YAML配置模板

```yaml
name: "{service-name}"
host: "localhost"
port: {http_port}
grpc_port: {grpc_port}

database:
  driver: "mysql"
  host: "localhost"
  port: 3306
  username: "root"
  password: "123456"
  dbname: "{service}_db"
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600

# Redis：单节点模式配置host+port，集群模式配置cluster_addresses
redis:
  host: "localhost"
  port: 6379
  cluster_addresses:
    # - "192.168.1.1:6379"
    # - "192.168.1.2:6379"
    # - "192.168.1.3:6379"
  password: ""
  db: 0
  pool_size: 100

jwt:
  secret: "{jwt-secret}"
  expire: 24h
  issuer: "{service-name}"

# Consul：单节点模式配置host+port，集群模式配置addresses
consul:
  host: "localhost"
  port: 8500
  addresses:
    # - "192.168.1.1:8500"
    # - "192.168.1.2:8500"
    # - "192.168.1.3:8500"
  token: ""
  scheme: "http"
  check_interval: "10s"
  check_timeout: "5s"
  ttl: "30s"
  deregister_critical_after: "90s"
  keepalive_interval: "10s"

logger:
  level: "info"
  format: "json"

shutdown:
  timeout: "5s"
```

## 三、服务注册规范

### 3.1 注册流程
1. 创建Consul客户端
2. 注册HTTP服务（带HTTP健康检查 + TTL）
3. 注册gRPC服务（带gRPC健康检查 + TTL）
4. 启动KeepAlive协程（心跳保活）

### 3.2 健康检查
- **HTTP检查**：`{scheme}://{host}:{port}/health`，scheme从配置读取
- **gRPC检查**：`{host}:{grpc_port}`
- **TTL**：从配置读取，默认30s
- **自动注销**：服务不可用超过 `deregister_critical_after` 后自动注销

### 3.3 KeepAlive
- 间隔从配置读取，默认10s
- 使用 `stopCh` 通道控制退出
- 解析配置失败时使用默认值并打印警告

## 四、JWT规范

### 4.1 初始化
```go
pkg.InitJWT(cfg.JWT.Secret, cfg.JWT.Expire, cfg.JWT.Issuer)
```

### 4.2 生成Token
- 过期时间从 `jwtExpire` 读取
- Issuer从 `jwtIssuer` 读取
- 签名算法：HS256

### 4.3 中间件
- 公开接口（注册、登录）不需要认证
- 受保护接口使用 `middleware.JWTAuth()` 中间件
- 验证成功后将 `user_id`、`username` 写入 `gin.Context`

## 五、优雅关闭规范

关闭顺序（严格按此顺序）：
1. 停止TTL心跳 `close(stopKeepAlive)`
2. 从Consul注销服务 `consulRegistry.Deregister()`
3. 关闭HTTP服务器 `httpSrv.Shutdown(ctx)`，超时从配置读取
4. 关闭gRPC服务器 `grpcSrv.GracefulStop()`

## 六、路由规范

```
GET  /health                    # 健康检查（无认证）
POST /api/v1/{service}/register # 注册（无认证）
POST /api/v1/{service}/login    # 登录（无认证）
GET  /api/v1/{service}/list     # 列表（需认证）
```

## 七、分层职责

| 层       | 职责                                   | 禁止                         |
| -------- | -------------------------------------- | ---------------------------- |
| Handler  | 参数校验、调用Logic、返回响应          | 包含业务逻辑                 |
| Logic    | 核心业务逻辑、数据操作                 | 直接处理HTTP请求/响应        |
| Svc      | 依赖注入、管理共享资源                 | 包含业务逻辑                 |
| Types    | 结构体定义                             | 包含任何逻辑代码             |
| Pkg      | 可复用工具（JWT、响应封装）            | 引用internal包               |
| Registry | 服务注册/注销、心跳保活                | 包含业务逻辑                 |
| Config   | 配置结构体定义、配置加载               | 包含业务逻辑                 |

## 八、编码规范

1. **零硬编码**：所有可变值从配置读取，包括协议（http/https）、地址、端口、超时、间隔等
2. **配置容错**：解析duration配置失败时使用默认值并打印警告日志
3. **结构体字段**：未使用的字段不要定义；定义了就必须使用
4. **导入管理**：删除字段后同步清理对应的import
5. **错误处理**：所有error必须处理，不允许静默忽略
6. **日志规范**：关键操作（注册、注销、启动、关闭）必须打印日志
