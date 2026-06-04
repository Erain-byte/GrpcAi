# Go & PHP 微服务开发 SKILL

## 零、开发者资质声明

**技术背景：**
- **Go 语言开发经验：5+ 年**
  - 精通 Go 1.18+ 泛型特性
  - 深入理解 goroutine、channel、context 并发模型
  - 熟练掌握 gRPC、Protobuf、Gin、GORM 等生态
  - 具备生产级微服务架构设计与实施能力

- **PHP 开发经验：10+ 年**
  - 精通 PHP 7.x/8.x 现代特性
  - 熟悉 Laravel、Symfony、ThinkPHP 等主流框架
  - 具备大型电商、CMS、ERP 系统开发经验
  - 深刻理解 PHP-FPM、Swoole、Workerman 运行机制

**核心原则：**
1. **严格遵循最佳实践**：所有代码必须符合行业标准和官方推荐
2. **禁止使用废弃 API**：定期检查并替换已弃用的函数和方法
3. **目录结构规范化**：严格遵循分层架构和模块化设计
4. **代码质量优先**：可读性 > 可维护性 > 性能 > 开发效率
5. **安全性第一**：所有敏感配置必须通过环境变量管理

---

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

```
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

## 九、Go 语言严格规范（5+年经验标准）

### 9.1 版本与特性要求

```go
// ✅ 正确：使用 Go 1.18+ 泛型
func CreateClient[T any](conn grpc.ClientConnInterface) T {
    return grpc.NewClient[T](conn)
}

// ❌ 错误：使用旧版 interface{}
func CreateClient(conn interface{}) interface{} {
    // ...
}

// ✅ 正确：使用 any 替代 interface{}
type Handler func(ctx context.Context, req any) (any, error)

// ❌ 错误：继续使用 interface{}
type Handler func(ctx context.Context, req interface{}) (interface{}, error)
```

**强制要求：**
- Go 版本：≥ 1.18（当前项目使用 1.26.2）
- 必须使用 `any` 而非 `interface{}`
- 合理使用泛型减少代码重复
- 禁止使用已废弃的包和函数

### 9.2 Context 使用规范

```go
// ✅ 正确：立即调用 cancel
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel() // 或使用后立即 cancel
err := someOperation(ctx)
cancel() // 操作完成后立即释放

// ❌ 错误：在匿名函数中 defer cancel（可能导致超时失效）
defer func() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel() // 这个 defer 会在 Shutdown 之后才执行
    tracer.Shutdown(ctx)
}()
```

**核心原则：**
- Context 仅用于传递超时、取消信号和请求元信息
- 禁止通过 Context 传递业务数据
- `cancel()` 必须在操作完成后立即调用，不可依赖嵌套 defer

### 9.3 错误处理规范

```go
// ✅ 正确：完整的错误处理
result, err := doSomething()
if err != nil {
    logger.Errorf("Failed to do something: %v", err)
    return nil, fmt.Errorf("do something failed: %w", err)
}

// ❌ 错误：忽略错误
result, _ := doSomething()

// ❌ 错误：静默处理
if err := doSomething(); err != nil {
    // 什么都不做
}
```

### 9.4 并发安全规范

```go
// ✅ 正确：使用 sync.RWMutex 保护共享资源
type ClientManager struct {
    mu      sync.RWMutex
    clients map[string]*grpc.ClientConn
}

func (m *ClientManager) GetClient(name string) *grpc.ClientConn {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.clients[name]
}

// ❌ 错误：无锁访问共享资源
func (m *ClientManager) GetClient(name string) *grpc.ClientConn {
    return m.clients[name] // 竞态条件
}
```

### 9.5 禁止使用的废弃 API

| 废弃 API | 替代方案 | 说明 |
|---------|---------|------|
| `ioutil.ReadFile` | `os.ReadFile` | Go 1.16+ |
| `ioutil.WriteFile` | `os.WriteFile` | Go 1.16+ |
| `ioutil.ReadDir` | `os.ReadDir` | Go 1.16+ |
| `ioutil.TempFile` | `os.CreateTemp` | Go 1.17+ |
| `context.WithCancelCause` | 谨慎使用 | Go 1.20+，需明确场景 |
| `grpc.Dial` | `grpc.NewClient` | gRPC 1.64+ |
| `grpc.WithBlock` | 移除或使用 context | 已弃用 |

### 9.6 代码风格规范

```go
// ✅ 正确：短变量声明
conn, err := grpc.NewClient(target, opts...)

// ✅ 正确：错误优先处理
if err != nil {
    return nil, err
}

// ✅ 正确：接口命名以 er 结尾
type Reader interface {
    Read(p []byte) (n int, err error)
}

// ✅ 正确：包名简短小写
package user // 不是 UserService 或 users

// ❌ 错误：使用驼峰命名包
package userService
```

---

## 十、PHP 语言严格规范（10+年经验标准）

### 10.1 版本与特性要求

**强制要求：**
- PHP 版本：≥ 8.0（推荐 8.2+）
- 必须使用类型声明（strict_types）
- 使用 Constructor Property Promotion（PHP 8.0+）
- 使用 Match 表达式替代 switch（PHP 8.0+）
- 使用 Nullsafe 运算符（PHP 8.0+）

```php
<?php
declare(strict_types=1);

// ✅ 正确：PHP 8.0+ Constructor Property Promotion
class UserService
{
    public function __construct(
        private UserRepository $repository,
        private LoggerInterface $logger,
    ) {}
    
    // ✅ 正确：使用 match 表达式
    public function getStatusLabel(int $status): string
    {
        return match($status) {
            0 => 'Inactive',
            1 => 'Active',
            2 => 'Suspended',
            default => 'Unknown',
        };
    }
    
    // ✅ 正确：使用 nullsafe 运算符
    public function getUserName(?int $userId): ?string
    {
        return $this->repository->findById($userId)?->getName();
    }
}

// ❌ 错误：旧式构造函数
class UserService
{
    private $repository;
    private $logger;
    
    public function __construct(UserRepository $repository, LoggerInterface $logger)
    {
        $this->repository = $repository;
        $this->logger = $logger;
    }
}
```

### 10.2 禁止使用的废弃函数

| 废弃函数 | 替代方案 | PHP 版本 |
|---------|---------|---------|
| `create_function` | 匿名函数 `fn()` | 7.2+ 废弃，8.0+ 移除 |
| `each()` | `foreach` | 7.2+ 废弃，8.0+ 移除 |
| `gmp_random()` | `gmp_random_bits()` | 7.2+ 废弃 |
| `parse_str` (无第二参数) | `parse_str($str, $result)` | 7.2+ 废弃 |
| `assert` (字符串参数) | `assert(bool expression)` | 7.2+ 废弃 |
| `implode` (参数顺序颠倒) | `implode($glue, $pieces)` | 7.4+ 警告 |
| `curl_setopt` CURLEASY_XXX | CURLMOPT_XXX | 根据场景 |

### 10.3 PSR 规范遵循

**必须遵循的标准：**
- PSR-1：基本编码规范
- PSR-4：自动加载规范
- PSR-12：扩展编码风格指南
- PSR-3：日志接口规范
- PSR-7：HTTP 消息接口
- PSR-11：容器接口
- PSR-18：HTTP 客户端

```php
<?php
declare(strict_types=1);

namespace App\Service;

use Psr\Log\LoggerInterface;
use App\Repository\UserRepository;

// ✅ 正确：PSR-12 规范
final class UserService
{
    public function __construct(
        private UserRepository $repository,
        private LoggerInterface $logger,
    ) {}
    
    public function findById(int $id): ?User
    {
        try {
            return $this->repository->find($id);
        } catch (\Exception $e) {
            $this->logger->error('Failed to find user', [
                'user_id' => $id,
                'error' => $e->getMessage(),
            ]);
            return null;
        }
    }
}
```

### 10.4 依赖注入规范

```php
<?php
// ✅ 正确：使用接口依赖，便于测试和替换
interface PaymentGatewayInterface
{
    public function charge(float $amount): bool;
}

class StripePaymentGateway implements PaymentGatewayInterface
{
    public function charge(float $amount): bool
    {
        // Stripe 实现
    }
}

class OrderService
{
    public function __construct(
        private PaymentGatewayInterface $paymentGateway,
    ) {}
    
    public function placeOrder(Order $order): bool
    {
        return $this->paymentGateway->charge($order->getTotal());
    }
}

// ❌ 错误：直接依赖具体实现
class OrderService
{
    private StripePaymentGateway $paymentGateway; // 紧耦合
}
```

### 10.5 安全编码规范

```php
<?php
// ✅ 正确：防止 SQL 注入（使用 PDO 预处理语句）
$stmt = $pdo->prepare('SELECT * FROM users WHERE email = :email');
$stmt->execute(['email' => $email]);

// ✅ 正确：防止 XSS
echo htmlspecialchars($userInput, ENT_QUOTES, 'UTF-8');

// ✅ 正确：密码哈希
$hash = password_hash($password, PASSWORD_BCRYPT, ['cost' => 12]);
$isValid = password_verify($password, $hash);

// ✅ 正确：CSRF 保护
session_start();
if (!isset($_SESSION['csrf_token'])) {
    $_SESSION['csrf_token'] = bin2hex(random_bytes(32));
}

// ❌ 错误：直接拼接 SQL
$query = "SELECT * FROM users WHERE email = '$email'"; // SQL 注入风险

// ❌ 错误：直接输出用户输入
echo $userInput; // XSS 风险

// ❌ 错误：使用 md5 存储密码
$passwordHash = md5($password); // 不安全
```

---

## 十一、微服务架构设计规范

### 11.1 服务拆分原则

**单一职责原则：**
- 每个服务只负责一个业务领域
- 服务之间通过 API 通信，不共享数据库
- 服务可以独立部署、独立扩展

**示例：**
```
✅ 正确拆分：
- user-service：用户管理
- order-service：订单管理
- payment-service：支付处理
- notification-service：消息通知

❌ 错误拆分：
- user-order-service：用户和订单耦合
- all-in-one-service：单体应用伪装成微服务
```

### 11.2 服务通信规范

**同步通信（gRPC）：**
- 适用于需要即时响应的场景
- 必须设置超时时间
- 必须实现重试机制

**异步通信（消息队列）：**
- 适用于解耦和削峰填谷
- 必须保证消息幂等性
- 必须实现死信队列

### 11.3 数据一致性保障

**最终一致性策略：**
- 使用 Saga 模式处理分布式事务
- 使用事件溯源记录状态变化
- 使用补偿机制处理失败回滚

```go
// ✅ 正确：Saga 模式示例
type OrderSaga struct {
    steps []SagaStep
}

func (s *OrderSaga) Execute(ctx context.Context, order *Order) error {
    for i, step := range s.steps {
        if err := step.Execute(ctx, order); err != nil {
            // 执行补偿操作
            for j := i - 1; j >= 0; j-- {
                s.steps[j].Compensate(ctx, order)
            }
            return err
        }
    }
    return nil
}
```

---

## 十二、测试规范

### 12.1 Go 测试规范

```go
// ✅ 正确：单元测试
func TestUserService_CreateUser(t *testing.T) {
    // Arrange
    repo := &MockUserRepository{}
    service := NewUserService(repo)
    
    // Act
    user, err := service.CreateUser(context.Background(), &CreateUserRequest{
        Name:  "John",
        Email: "john@example.com",
    })
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, user)
    assert.Equal(t, "John", user.Name)
}

// ✅ 正确：基准测试
func BenchmarkUserService_CreateUser(b *testing.B) {
    service := setupBenchmarkService()
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _, _ = service.CreateUser(context.Background(), &CreateUserRequest{
            Name:  fmt.Sprintf("User%d", i),
            Email: fmt.Sprintf("user%d@example.com", i),
        })
    }
}
```

### 12.2 PHP 测试规范

```php
<?php
// ✅ 正确：PHPUnit 测试
class UserServiceTest extends TestCase
{
    public function testCreateUser(): void
    {
        // Arrange
        $repository = $this->createMock(UserRepository::class);
        $service = new UserService($repository);
        
        // Act
        $user = $service->createUser('John', 'john@example.com');
        
        // Assert
        $this->assertNotNull($user);
        $this->assertEquals('John', $user->getName());
    }
}
```

---

## 十三、性能优化规范

### 13.1 Go 性能优化

```go
// ✅ 正确：使用 sync.Pool 复用对象
var bufferPool = sync.Pool{
    New: func() interface{} {
        return bytes.NewBuffer(make([]byte, 0, 1024))
    },
}

func ProcessData(data []byte) []byte {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()
    
    buf.Write(data)
    // 处理逻辑...
    return buf.Bytes()
}

// ✅ 正确：预分配切片容量
items := make([]Item, 0, len(input)) // 避免多次扩容

// ❌ 错误：动态增长切片
items := []Item{}
for _, v := range input {
    items = append(items, v) // 可能多次扩容
}
```

### 13.2 PHP 性能优化

```php
<?php
// ✅ 正确：使用生成器处理大数据集
function readLargeFile(string $filename): Generator
{
    $handle = fopen($filename, 'r');
    while (($line = fgets($handle)) !== false) {
        yield trim($line);
    }
    fclose($handle);
}

// ✅ 正确：使用 OPcache
// php.ini 配置：
// opcache.enable=1
// opcache.memory_consumption=256
// opcache.max_accelerated_files=10000

// ❌ 错误：一次性加载大文件到内存
$lines = file('large_file.txt'); // 内存溢出风险
```

---

## 十四、文档规范

### 14.1 Go 文档注释

```go
// CreateUser creates a new user with the given request data.
// It validates the input, hashes the password, and stores the user in the database.
// Returns the created user or an error if validation fails or database operation fails.
//
// Example:
//   user, err := service.CreateUser(ctx, &CreateUserRequest{
//       Name:     "John",
//       Email:    "john@example.com",
//       Password: "secure_password",
//   })
func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
    // 实现...
}
```

### 14.2 PHP 文档注释

```php
<?php
/**
 * Creates a new user with the given data.
 *
 * @param string $name The user's full name
 * @param string $email The user's email address
 * @param string $password The user's plain text password (will be hashed)
 *
 * @return User The created user entity
 *
 * @throws ValidationException If the input data is invalid
 * @throws DatabaseException If the database operation fails
 *
 * @example
 * ```php
 * $user = $service->createUser('John', 'john@example.com', 'secure_password');
 * ```
 */
public function createUser(string $name, string $email, string $password): User
{
    // 实现...
}
```

---

## 十五、持续改进清单

### 15.1 定期审查项

- [ ] 每月检查 Go/PHP 版本更新，评估升级必要性
- [ ] 每季度审查依赖包，移除未使用和已废弃的包
- [ ] 每半年进行一次代码质量审计（使用 golangci-lint、PHPStan）
- [ ] 每年更新技术栈，淘汰过时技术

### 15.2 技术债务管理

- 发现技术债务立即记录到 Issue Tracker
- 按优先级排序：安全性 > 性能 > 可维护性 > 功能
- 每个 Sprint 预留 20% 时间处理技术债务

---

**最后更新时间：** 2026-06-02  
**维护者：** Senior Go & PHP Developer (5+ years Go, 10+ years PHP)