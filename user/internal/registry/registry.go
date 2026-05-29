package registry

import (
	"fmt"
	"log"
	"time"
	"user/internal/config"

	"github.com/hashicorp/consul/api"
)

type ConsulRegistry struct {
	client *api.Client
	config config.ConsulConfig
}

func NewConsulRegistry(cfg config.ConsulConfig) (*ConsulRegistry, error) {
	consulConfig := api.DefaultConfig()
	// 优先使用集群地址，否则使用单节点地址
	addresses := cfg.GetAddresses()
	consulConfig.Address = addresses[0]
	if cfg.Token != "" {
		consulConfig.Token = cfg.Token
	}

	client, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %v", err)
	}

	return &ConsulRegistry{
		client: client,
		config: cfg,
	}, nil
}

// Register 注册服务到Consul，同时支持HTTP/gRPC健康检查和TTL心跳
func (r *ConsulRegistry) Register(name string, host string, httpPort int, grpcPort int) error {
	// 注册HTTP服务，同时启用HTTP健康检查和TTL检查
	httpRegistration := &api.AgentServiceRegistration{
		ID:      fmt.Sprintf("%s-http", name),
		Name:    name,
		Port:    httpPort,
		Address: host,
		Check: &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("%s://%s:%d/health", r.config.Scheme, host, httpPort),
			Interval:                       r.config.CheckInterval,
			Timeout:                        r.config.CheckTimeout,
			TTL:                            r.config.TTL,
			TLSSkipVerify:                  true,
			DeregisterCriticalServiceAfter: r.config.DeregisterCriticalAfter,
		},
	}

	if err := r.client.Agent().ServiceRegister(httpRegistration); err != nil {
		return fmt.Errorf("failed to register HTTP service: %v", err)
	}
	log.Printf("HTTP service registered to Consul: %s (host: %s, port: %d)", name, host, httpPort)

	// 注册gRPC服务，同时启用gRPC健康检查和TTL检查
	grpcRegistration := &api.AgentServiceRegistration{
		ID:      fmt.Sprintf("%s-grpc", name),
		Name:    fmt.Sprintf("%s-grpc", name),
		Port:    grpcPort,
		Address: host,
		Check: &api.AgentServiceCheck{
			GRPC:                           fmt.Sprintf("%s:%d", host, grpcPort),
			Interval:                       r.config.CheckInterval,
			Timeout:                        r.config.CheckTimeout,
			TTL:                            r.config.TTL,
			DeregisterCriticalServiceAfter: r.config.DeregisterCriticalAfter,
		},
	}

	if err := r.client.Agent().ServiceRegister(grpcRegistration); err != nil {
		return fmt.Errorf("failed to register gRPC service: %v", err)
	}
	log.Printf("gRPC service registered to Consul: %s-grpc (host: %s, port: %d)", name, host, grpcPort)

	return nil
}

// KeepAlive TTL心跳保活，定期向Consul报告服务健康状态
func (r *ConsulRegistry) KeepAlive(name string, stopCh <-chan struct{}) {
	httpCheckID := fmt.Sprintf("service:%s-http", name)
	grpcCheckID := fmt.Sprintf("service:%s-grpc", name)

	interval, err := time.ParseDuration(r.config.KeepAliveInterval)
	if err != nil {
		log.Printf("Invalid keepalive interval %q, using default 10s: %v", r.config.KeepAliveInterval, err)
		interval = 10 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 更新HTTP服务TTL
			if err := r.client.Agent().UpdateTTL(httpCheckID, "ok", "pass"); err != nil {
				log.Printf("Failed to update HTTP TTL: %v", err)
			}
			// 更新gRPC服务TTL
			if err := r.client.Agent().UpdateTTL(grpcCheckID, "ok", "pass"); err != nil {
				log.Printf("Failed to update gRPC TTL: %v", err)
			}
		case <-stopCh:
			log.Println("TTL keepalive stopped")
			return
		}
	}
}

// Deregister 从Consul注销服务
func (r *ConsulRegistry) Deregister(name string) error {
	// 注销HTTP服务
	if err := r.client.Agent().ServiceDeregister(fmt.Sprintf("%s-http", name)); err != nil {
		log.Printf("Failed to deregister HTTP service: %v", err)
	} else {
		log.Printf("HTTP service deregistered from Consul: %s", name)
	}

	// 注销gRPC服务
	if err := r.client.Agent().ServiceDeregister(fmt.Sprintf("%s-grpc", name)); err != nil {
		log.Printf("Failed to deregister gRPC service: %v", err)
	} else {
		log.Printf("gRPC service deregistered from Consul: %s-grpc", name)
	}

	return nil
}

