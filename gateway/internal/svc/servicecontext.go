package svc

import (
	"context"
	"fmt"
	"gateway/internal/config"
	"gateway/internal/grpc"
	"gateway/internal/registry"
	"time"

	"github.com/redis/go-redis/v9"
)

type ServiceContext struct {
	Config      config.Config
	Redis       redis.Cmdable
	Registry    *registry.ConsulRegistry
	GrpcClients *grpc.ClientManager
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化数据库连接
	/*db, err := gorm.Open(mysql.Open(GetDSN(c.Database)), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}*/

	// 初始化Redis连接（自动判断单节点/集群模式）
	var rdb redis.Cmdable
	poolSize := c.Redis.PoolSize
	if poolSize <= 0 {
		poolSize = 100
	}
	minIdleConns := c.Redis.MinIdleConns
	if minIdleConns < 0 {
		minIdleConns = 0
	}
	maxIdleConns := c.Redis.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = poolSize / 2
	}

	if c.Redis.IsCluster() {
		rdb = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:           c.Redis.ClusterAddresses,
			Password:        c.Redis.Password,
			PoolSize:        poolSize,
			MinIdleConns:    minIdleConns,
			MaxIdleConns:    maxIdleConns,
			ConnMaxIdleTime: parseDuration(c.Redis.ConnMaxIdleTime, 30*time.Minute),
			ConnMaxLifetime: parseDuration(c.Redis.ConnMaxLifetime, time.Hour),
			DialTimeout:     parseDuration(c.Redis.DialTimeout, 5*time.Second),
			ReadTimeout:     parseDuration(c.Redis.ReadTimeout, 3*time.Second),
			WriteTimeout:    parseDuration(c.Redis.WriteTimeout, 3*time.Second),
			PoolTimeout:     parseDuration(c.Redis.PoolTimeout, 4*time.Second),
		})
	} else {
		rdb = redis.NewClient(&redis.Options{
			Addr:            c.Redis.GetAddr(),
			Password:        c.Redis.Password,
			DB:              c.Redis.DB,
			PoolSize:        poolSize,
			MinIdleConns:    minIdleConns,
			MaxIdleConns:    maxIdleConns,
			ConnMaxIdleTime: parseDuration(c.Redis.ConnMaxIdleTime, 30*time.Minute),
			ConnMaxLifetime: parseDuration(c.Redis.ConnMaxLifetime, time.Hour),
			DialTimeout:     parseDuration(c.Redis.DialTimeout, 5*time.Second),
			ReadTimeout:     parseDuration(c.Redis.ReadTimeout, 3*time.Second),
			WriteTimeout:    parseDuration(c.Redis.WriteTimeout, 3*time.Second),
			PoolTimeout:     parseDuration(c.Redis.PoolTimeout, 4*time.Second),
		})
	}

	return &ServiceContext{
		Config: c,
		Redis:  rdb,
	}
}

func (s *ServiceContext) HealthCheck(ctx context.Context) error {
	if s == nil || s.Redis == nil {
		return fmt.Errorf("redis client is not initialized")
	}

	if err := s.Redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	return nil
}

func (s *ServiceContext) Close() error {
	if s == nil || s.Redis == nil {
		return nil
	}

	if closer, ok := s.Redis.(interface{ Close() error }); ok {
		return closer.Close()
	}

	return nil
}

func parseDuration(value string, fallback time.Duration) time.Duration {
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return fallback
	}
	return duration
}
