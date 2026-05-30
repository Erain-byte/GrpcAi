package config

import (
	"fmt"
	"github.com/creasty/defaults"
	"github.com/spf13/viper"
)

type Config struct {
	Name         string       `yaml:"name" default:"user-service"`
	Host         string       `yaml:"host" default:"localhost"`
	Port         int          `yaml:"port" default:"8001"`
	GRPCPort     int          `yaml:"grpc_port" default:"9001"`
	Consul       ConsulConfig  `yaml:"consul"`
	Database     DBConfig     `yaml:"database"`
	Redis        RedisConfig  `yaml:"redis"`
	JWT          JWTConfig    `yaml:"jwt"`
	Logger       LoggerConfig `yaml:"logger"`
	Shutdown     ShutdownConfig `yaml:"shutdown"`
	Service      ServiceConfig `yaml:"service"`
}

type ConsulConfig struct {
	Addresses                   []string `yaml:"addresses"`  // 集群地址列表，如 ["192.168.1.1:8500", "192.168.1.2:8500"]
	Host                        string   `yaml:"host" default:"localhost"`   // 单节点地址（addresses为空时使用）
	Port                        int      `yaml:"port" default:"8500"`        // 单节点端口（addresses为空时使用）
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
	DBName          string `yaml:"dbname" default:"user_db"`
	MaxIdleConns    int    `yaml:"max_idle_conns" default:"10"`
	MaxOpenConns    int    `yaml:"max_open_conns" default:"100"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime" default:"3600"`
}

type RedisConfig struct {
	Host              string   `yaml:"host" default:"localhost"`     // 单节点地址（cluster_addresses为空时使用）
	Port              int      `yaml:"port" default:"6379"`          // 单节点端口（cluster_addresses为空时使用）
	ClusterAddresses  []string `yaml:"cluster_addresses"`            // 集群地址列表，如 ["192.168.1.1:6379", "192.168.1.2:6379"]
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

func Init(configPath string) (*Config, error) {
	v := viper.New()

	// 设置配置文件路径
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("user")
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
