package svc

import (
	"fmt"
	"user/internal/config"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config config.Config
	DB     *gorm.DB
	Redis  redis.Cmdable
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化数据库连接
	db, err := gorm.Open(mysql.Open(GetDSN(c.Database)), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// 初始化Redis连接（自动判断单节点/集群模式）
	var rdb redis.Cmdable
	if c.Redis.IsCluster() {
		rdb = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    c.Redis.ClusterAddresses,
			Password: c.Redis.Password,
			PoolSize: c.Redis.PoolSize,
		})
	} else {
		rdb = redis.NewClient(&redis.Options{
			Addr:     c.Redis.GetAddr(),
			Password: c.Redis.Password,
			DB:       c.Redis.DB,
			PoolSize: c.Redis.PoolSize,
		})
	}

	return &ServiceContext{
		Config: c,
		DB:     db,
		Redis:  rdb,
	}
}

func GetDSN(db config.DBConfig) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		db.Username,
		db.Password,
		db.Host,
		db.Port,
		db.DBName,
	)
}
