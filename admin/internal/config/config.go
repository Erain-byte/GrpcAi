package config

import (
	"fmt"
	"github.com/creasty/defaults"
	"github.com/spf13/viper"
)

type Config struct {
	Name         string       `yaml:"name" default:"admin-service"`
	Host         string       `yaml:"host" default:"localhost"`
	Port         int          `yaml:"port" default:"8002"`
	GRPCPort     int          `yaml:"grpc_port" default:"9002"`
	Consul       ConsulConfig  `yaml:"consul"`
	Database     DBConfig     `yaml:"database"`
	Redis        RedisConfig  `yaml:"redis"`
	JWT          JWTConfig    `yaml:"jwt"`
	Logger       LoggerConfig `yaml:"logger"`
	Shutdown     ShutdownConfig `yaml:"shutdown"`
	Service      ServiceConfig `yaml:"service"`
	Grpc         GrpcConfig   `yaml:"grpc"`               // gRPC TLS/mTLS 配置
}

type ConsulConfig struct {
	Addresses                   []string `yaml:"addresses"`  // 集群地址列表
	Host                        string   `yaml:"host" default:"localhost"`   // 单节点地址
	Port                        int      `yaml:"port" default:"8500"`        // 单节点端口
	Token                       string   `yaml:"token" default:""`
	Scheme                      string   `yaml:"scheme" default:"http"`
	CheckInterval               string   `yaml:"check_interval" default:"10s"`
	CheckTimeout                string   `yaml:"check_timeout" default:"5s"`
	TTL                         string   `yaml:"ttl" default:"30s"`
	DeregisterCriticalAfter     string   `yaml:"deregister_critical_after" default:"90s"`
	KeepAliveInterval           string   `yaml:"keepalive_interval" default:"10s"`
}

// GetAddresses 获取Consul地址列表，优先使用集群地址
func (c *ConsulConfig) GetAddresses() []string {
	if len(c.Addresses) > 0 {
		return c.Addresses
	}
	return []string{fmt.Sprintf("%s:%d", c.Host, c.Port)}
}

type DBConfig struct {
	Driver          string `yaml:"driver" default:"mysql"`
	Host            string `yaml:"host" default:"localhost"`
	Port            int    `yaml:"port" default:"3306"`
	Username        string `yaml:"username" default:"root"`
	Password        string `yaml:"password" default:"123456"`
	DBName          string `yaml:"dbname" default:"admin_db"`
	MaxIdleConns    int    `yaml:"max_idle_conns" default:"10"`
	MaxOpenConns    int    `yaml:"max_open_conns" default:"100"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime" default:"3600"`
}

type RedisConfig struct {
	Host              string   `yaml:"host" default:"localhost"`     // 单节点地址
	Port              int      `yaml:"port" default:"6379"`          // 单节点端口
	ClusterAddresses  []string `yaml:"cluster_addresses"`            // 集群地址列表
	Password          string   `yaml:"password" default:""`
	DB                int      `yaml:"db" default:"0"`
	PoolSize          int      `yaml:"pool_size" default:"100"`
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
}

type LoggerConfig struct {
	Level  string `yaml:"level" default:"info"`
	Format string `yaml:"format" default:"json"`
}

type ShutdownConfig struct {
	Timeout string `yaml:"timeout" default:"5s"`
}

type ServiceConfig struct {
	Version     string   `yaml:"version"`
	CTags       []string `yaml:"tags"`
	PublicAPIs  []string `yaml:"public_apis"`
	AuthAPIs    []string `yaml:"auth_apis"`
	CorsEnabled bool     `yaml:"cors_enabled"`
	CORS        CORSConfig `yaml:"cors"`
}

type CORSConfig struct {
	AllowOrigins     []string `yaml:"allow_origins"`
	AllowMethods     []string `yaml:"allow_methods"`
	AllowHeaders     []string `yaml:"allow_headers"`
	ExposeHeaders    []string `yaml:"expose_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
	MaxAge          int      `yaml:"max_age"`
}

// GrpcConfig gRPC TLS/mTLS 配置（仅用于未来扩展）
// 注意：Admin 服务当前仅作为 gRPC 服务端，此配置暂未使用
// 如需在服务端启用 TLS，请在 grpc.go 中加载证书并创建 grpc.Credentials
type GrpcConfig struct {
	UseTLS             bool   `yaml:"use_tls" default:"false"`                    // 是否启用 TLS
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify" default:"false"`       // 跳过证书验证
	CertFile           string `yaml:"cert_file" default:""`                       // 服务端证书路径
	KeyFile            string `yaml:"key_file" default:""`                        // 服务端私钥路径
	CaFile             string `yaml:"ca_file" default:""`                         // CA 证书路径
	ServerName         string `yaml:"server_name" default:"admin-service"`        // TLS Server Name
}

func Init(configPath string) (*Config, error) {
	v := viper.New()

	// 设置配置文件路径
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("admin")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./etc")
	}

	// 设置默认值
	var cfg Config
	if err := defaults.Set(&cfg); err != nil {
		return nil, fmt.Errorf("failed to set default values: %v", err)
	}

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// 解析配置文件
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return &cfg, nil
}
