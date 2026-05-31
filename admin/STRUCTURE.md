# Admin Service

管理后台微服务，提供管理员登录、权限管理、角色管理等核心功能，支持 HTTP REST API 和 gRPC 双协议访问。

## 📋 功能特性（待实现）

- ⏳ **管理员登录**：验证管理员账号密码，生成 JWT Token
- ⏳ **管理员登出**：注销登录状态
- ⏳ **管理员信息管理**：创建、查询、更新管理员信息
- ⏳ **角色管理**：角色的增删改查
- ⏳ **权限管理**：权限分配和控制
- ⏳ **双协议支持**：同时提供 HTTP REST API 和 gRPC 接口
- ⏳ **JWT 认证**：基于 HS256 算法的 Token 认证机制
- ⏳ **服务注册**：自动注册到 Consul 并维持 TTL 心跳
- ⏳ **优雅关闭**：支持平滑停机
- ⏳ **健康检查**：提供 `/health` 端点

## 🏗️ 架构设计（规划）

```
Client Request (HTTP/gRPC)
      ↓
┌─────────────────────────────┐
│   Admin Service             │
│   HTTP: 8002                │
│   gRPC: 9002                │
│                             │
│  • Middleware Layer         │
│    - JWT Authentication     │
│    - Recovery               │
│    - CORS                   │
│                             │
│  • Handler Layer            │
│    - Parameter Validation   │
│    - Request/Response       │
│                             │
│  • Logic Layer              │
│    - Business Logic         │
│    - Password Encryption    │
│    - Token Generation       │
│    - Permission Check       │
│                             │
│  • Data Access Layer        │
│    - GORM (MySQL)           │
│    - Redis                  │
└─────────────────────────────┘
      ↓
┌──────────┐    ┌──────────┐
│  MySQL   │    │  Redis   │
│ admin_db │    │ Cache    │
└──────────┘    └──────────┘
      ↓
┌──────────┐
│  Consul  │
│ Registry │
└──────────┘
```

## 📁 目录结构（待完善）

```
admin/
├── api/                        # API 定义层
│   └── admin.proto             # Protobuf 接口定义（待创建）
├── cmd/server/main.go          # 入口文件（待实现）
├── etc/admin.yaml              # 配置文件（已完成）
├── internal/
│   ├── config/config.go        # 配置加载（待实现）
│   ├── handler/handler.go      # HTTP 处理层（待实现）
│   ├── logic/logic.go          # 业务逻辑层（待实现）
│   ├── middleware/middleware.go # 中间件（待实现）
│   ├── registry/registry.go    # Consul 服务注册（待实现）
│   ├── server/
│   │   ├── server.go           # HTTP 路由注册（待实现）
│   │   └── grpc.go             # gRPC 服务实现（待实现）
│   ├── svc/servicecontext.go   # 依赖注入（待实现）
│   └── types/types.go          # 数据模型（待实现）
├── pkg/pkg.go                  # 公共工具（待实现）
├── go.mod                      # Go 模块依赖
├── go.sum                      # 依赖校验文件
└── STRUCTURE.md                # 项目结构说明
```

## ⚙️ 配置说明

### admin.yaml 核心配置

```yaml
# 基础配置
name: "admin-service"
host: "localhost"
port: 8002              # HTTP 端口
grpc_port: 9002         # gRPC 端口

# 数据库配置
database:
  driver: "mysql"
  host: "localhost"
  port: 3306
  username: "root"
  password: "123456"
  dbname: "admin_db"
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
  secret: "g0-s3cr3t-k3y-for-jwt-t0k3n-auth3nt1c4t10n-2024"
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
    - "/api/v1/admin/login"
    - "/health"
  auth_apis:            # 受保护接口（需要认证）
    - "/api/v1/admin/users"
    - "/api/v1/admin/roles"
  tags:
    - "http"
    - "admin-api"
```

## 🚀 启动方式（待实现）

### 前置条件

1. **MySQL 数据库**：创建 `admin_db` 数据库
2. **Redis 服务**：确保 Redis 正常运行
3. **Consul 服务**：用于服务注册与发现

### 启动步骤（参考 User Service）

#### 1. 直接运行（开发模式）

```bash
cd admin
go run cmd/server/main.go -f etc/admin.yaml
```

#### 2. 编译后运行（生产模式）

```bash
go build -o admin.exe ./cmd/server
./admin.exe -f etc/admin.yaml
```

## 📡 API 端点（规划）

### HTTP REST API（待实现）

#### 健康检查

```bash
GET http://localhost:8002/health
```

预期响应：
```json
{
  "code": 200,
  "message": "ok"
}
```

#### 管理员登录

```bash
POST http://localhost:8002/api/v1/admin/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}
```

预期响应：
```json
{
  "code": 200,
  "message": "login success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "admin_info": {
      "id": 1,
      "username": "admin",
      "role": "super_admin"
    }
  }
}
```

#### 获取管理员列表（需要认证）

```bash
GET http://localhost:8002/api/v1/admin/users?page=1&page_size=10
Authorization: Bearer {token}
```

预期响应：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "username": "admin",
        "email": "admin@example.com",
        "role": "super_admin",
        "status": 1,
        "created_at": "2026-05-31T10:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "page_size": 10
  }
}
```

#### 创建管理员（需要认证）

```bash
POST http://localhost:8002/api/v1/admin/users
Authorization: Bearer {token}
Content-Type: application/json

{
  "username": "newadmin",
  "password": "password123",
  "email": "newadmin@example.com",
  "role": "admin"
}
```

#### 获取角色列表（需要认证）

```bash
GET http://localhost:8002/api/v1/admin/roles
Authorization: Bearer {token}
```

### gRPC 接口（规划）

Admin Service 计划提供以下 gRPC 方法：

| 方法 | 说明 | 认证 |
|------|------|------|
| `Login` | 管理员登录 | 无 |
| `Logout` | 管理员登出 | 需要 |
| `GetAdminInfo` | 获取管理员信息 | 需要 |
| `CreateAdmin` | 创建管理员 | 需要 |
| `GetAdminList` | 获取管理员列表 | 需要 |

## 🔧 实现指南

### 1. 需要实现的核心模块

#### Config 层（internal/config/config.go）

参考 User Service 的实现，解析 `admin.yaml` 配置文件。

#### Types 层（internal/types/types.go）

定义数据模型和请求/响应结构体：

```go
// Admin 管理员模型
type Admin struct {
    ID        uint      `json:"id" gorm:"primarykey"`
    Username  string    `json:"username" gorm:"unique;not null"`
    Password  string    `json:"-" gorm:"not null"`
    Email     string    `json:"email" gorm:"unique"`
    Role      string    `json:"role" gorm:"default:admin"`
    Status    int       `json:"status" gorm:"default:1"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// LoginRequest 登录请求
type LoginRequest struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
    Token     string    `json:"token"`
    AdminInfo AdminInfo `json:"admin_info"`
}

// AdminInfo 管理员信息
type AdminInfo struct {
    ID       uint   `json:"id"`
    Username string `json:"username"`
    Role     string `json:"role"`
}
```

#### Svc 层（internal/svc/servicecontext.go）

初始化数据库和 Redis 连接：

```go
type ServiceContext struct {
    Config config.Config
    DB     *gorm.DB
    Redis  redis.Cmdable
}

func NewServiceContext(c config.Config) *ServiceContext {
    // 初始化 MySQL
    db, err := gorm.Open(mysql.Open(GetDSN(c.Database)), &gorm.Config{})
    if err != nil {
        panic("failed to connect database")
    }
    
    // 初始化 Redis
    rdb := redis.NewClient(&redis.Options{
        Addr:     c.Redis.GetAddr(),
        Password: c.Redis.Password,
        DB:       c.Redis.DB,
        PoolSize: c.Redis.PoolSize,
    })
    
    return &ServiceContext{
        Config: c,
        DB:     db,
        Redis:  rdb,
    }
}
```

#### Logic 层（internal/logic/logic.go）

实现核心业务逻辑：

```go
type Logic struct {
    svcCtx *svc.ServiceContext
}

// Login 管理员登录
func (l *Logic) Login(req *types.LoginRequest) (*types.LoginResponse, error) {
    // 1. 查询管理员
    var admin types.Admin
    if err := l.svcCtx.DB.Where("username = ?", req.Username).First(&admin).Error; err != nil {
        return nil, errors.New("admin not found")
    }
    
    // 2. 验证密码
    if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(req.Password)); err != nil {
        return nil, errors.New("invalid password")
    }
    
    // 3. 生成 Token
    token, err := pkg.GenerateToken(admin.ID, admin.Username, admin.Role)
    if err != nil {
        return nil, err
    }
    
    return &types.LoginResponse{
        Token: token,
        AdminInfo: types.AdminInfo{
            ID:       admin.ID,
            Username: admin.Username,
            Role:     admin.Role,
        },
    }, nil
}

// CreateAdmin 创建管理员
func (l *Logic) CreateAdmin(req *types.CreateAdminRequest) error {
    // 1. 检查用户名是否已存在
    // 2. 密码加密
    // 3. 创建记录
}

// GetAdminList 获取管理员列表
func (l *Logic) GetAdminList(req *types.PageRequest) (*types.PageResponse, error) {
    // 1. 查询总数
    // 2. 分页查询
    // 3. 返回结果
}
```

#### Handler 层（internal/handler/handler.go）

处理 HTTP 请求：

```go
type Handler struct {
    logic *logic.Logic
}

func NewHandler(svcCtx *svc.ServiceContext) *Handler {
    return &Handler{
        logic: logic.NewLogic(svcCtx),
    }
}

// Login 管理员登录
func (h *Handler) Login(c *gin.Context) {
    var req types.LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, types.Response{
            Code:    http.StatusBadRequest,
            Message: err.Error(),
        })
        return
    }
    
    resp, err := h.logic.Login(&req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, types.Response{
            Code:    http.StatusInternalServerError,
            Message: err.Error(),
        })
        return
    }
    
    c.JSON(http.StatusOK, types.Response{
        Code:    http.StatusOK,
        Message: "login success",
        Data:    resp,
    })
}
```

#### Server 层（internal/server/server.go）

注册路由和中间件：

```go
func NewServer(svcCtx *svc.ServiceContext) *gin.Engine {
    r := gin.Default()
    
    // 中间件
    r.Use(middleware.Recovery())
    r.Use(middleware.CORS(svcCtx.Config.Service.CORS))
    
    // 初始化处理器
    h := handler.NewHandler(svcCtx)
    
    // 健康检查
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, types.Response{
            Code:    200,
            Message: "ok",
        })
    })
    
    // API v1 路由组
    v1 := r.Group("/api/v1")
    {
        // 管理员相关路由（公开接口）
        admin := v1.Group("/admin")
        {
            admin.POST("/login", h.Login)
        }
        
        // 管理员相关路由（需要认证）
        adminAuth := v1.Group("/admin").Use(middleware.JWTAuth())
        {
            adminAuth.GET("/users", h.GetAdminList)
            adminAuth.POST("/users", h.CreateAdmin)
            adminAuth.GET("/roles", h.GetRoleList)
        }
    }
    
    return r
}
```

#### Main 入口（cmd/server/main.go）

参考 User Service 的 main.go 实现，启动 HTTP + gRPC 服务并注册到 Consul。

### 2. 数据库表设计

```sql
-- 管理员表
CREATE TABLE admins (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(32) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    email VARCHAR(100) UNIQUE,
    role VARCHAR(32) DEFAULT 'admin',
    status TINYINT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 角色表
CREATE TABLE roles (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(32) NOT NULL UNIQUE,
    description VARCHAR(255),
    permissions JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 权限表
CREATE TABLE permissions (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,
    description VARCHAR(255),
    resource VARCHAR(64),
    action VARCHAR(32)
);
```

### 3. Protobuf 定义（api/admin.proto）

```protobuf
syntax = "proto3";

package admin;

option go_package = "github.com/Erain-byte/GrpcAi/proto/admin";

service AdminService {
  rpc Login(AdminLoginRequest) returns (AdminLoginResponse);
  rpc Logout(LogoutRequest) returns (LogoutResponse);
  rpc GetAdminInfo(GetAdminInfoRequest) returns (GetAdminInfoResponse);
  rpc CreateAdmin(CreateAdminRequest) returns (CreateAdminResponse);
  rpc GetAdminList(GetAdminListRequest) returns (GetAdminListResponse);
}

message AdminLoginRequest {
  string username = 1;
  string password = 2;
}

message AdminLoginResponse {
  string token = 1;
  AdminInfo admin_info = 2;
  string message = 3;
}

message AdminInfo {
  uint64 id = 1;
  string username = 2;
  string email = 3;
  string role = 4;
  int32 status = 5;
}

// ... 其他消息定义
```

## 🛠️ 技术栈（规划）

| 组件 | 技术选型 | 版本/说明 |
|------|---------|----------|
| **Web Framework** | Gin | 高性能 HTTP 框架 |
| **gRPC** | google.golang.org/grpc | gRPC 通信协议 |
| **Protocol Buffers** | protobuf | 接口定义语言 |
| **ORM** | GORM | Go 语言 ORM 框架 |
| **Database** | MySQL | 关系型数据库 |
| **Cache** | Redis | 缓存服务（支持集群） |
| **Service Discovery** | Consul | 服务注册与发现 |
| **Configuration** | Viper + YAML | 配置管理 |
| **JWT** | golang-jwt/jwt/v5 | Token 认证（HS256） |
| **Password Hashing** | bcrypt | 密码加密算法 |

## 📝 开发规范

1. **参考 User Service**：保持与 User Service 相同的架构和代码风格
2. **分层清晰**：Handler → Logic → Svc，职责明确
3. **统一响应格式**：使用 `types.Response` 结构
4. **密码安全**：使用 bcrypt 加密
5. **JWT 密钥一致**：与 Gateway、User Service 保持一致
6. **错误处理**：所有 error 必须处理
7. **日志记录**：关键操作打印日志

## 🔄 实现优先级

### Phase 1：基础功能
- [ ] 实现 Config、Types、Svc 层
- [ ] 实现管理员登录接口
- [ ] 实现 HTTP 服务器启动
- [ ] 注册到 Consul

### Phase 2：核心功能
- [ ] 实现管理员 CRUD 接口
- [ ] 实现 JWT 认证中间件
- [ ] 实现 gRPC 接口

### Phase 3：高级功能
- [ ] 实现角色管理
- [ ] 实现权限管理
- [ ] 添加 Redis 缓存
- [ ] 实现操作日志

## ⚠️ 注意事项

1. **JWT 密钥一致性**：Gateway、User Service、Admin Service 的 JWT Secret 必须一致

2. **数据库表初始化**：首次启动前需创建 `admin_db` 数据库和相关表

3. **默认管理员**：建议初始化一个超级管理员账号

4. **权限控制**：不同角色应有不同的权限范围

5. **服务依赖**：
   - MySQL 必须先启动
   - Redis 可选
   - Consul 必须运行

6. **安全性**：
   - 生产环境修改 JWT Secret
   - 限制 CORS 允许的域名
   - 实施密码强度策略

## 📊 性能目标

- **HTTP QPS**：> 1000 req/s
- **gRPC QPS**：> 2000 req/s
- **内存占用**：< 50 MB
- **启动时间**：< 1 秒
- **登录响应时间**：< 50ms

## 🔗 相关文档

- [User Service 实现参考](../user/STRUCTURE.md)
- [Gateway Service 文档](../gateway/README.md)
- [Proto 定义](../../proto/admin/admin.proto)

---

**状态**: 🚧 开发中  
**最后更新**: 2026-05-31  
**维护者**: goGit Team
