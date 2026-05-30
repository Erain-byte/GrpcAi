package registry

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
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
func (r *ConsulRegistry) Register(name string, host string, httpPort int, grpcPort int, metadata map[string]string, cfg *config.Config) error {
	// 注册HTTP服务，同时启用HTTP健康检查和TTL检查
	httpRegistration := &api.AgentServiceRegistration{
		ID:      fmt.Sprintf("%s-http", name),
		Name:    name,
		Port:    httpPort,
		Address: host,
		Tags:    BuildServiceTags(cfg),
		Meta:    metadata,
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
		Tags:    []string{"grpc"},
		Meta:    metadata,
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

// BuildServiceMetadata 构建服务元数据
func BuildServiceMetadata(cfg *config.Config) map[string]string {
	// 将API列表转换为逗号分隔的字符串
	publicAPIs := strings.Join(cfg.Service.PublicAPIs, ",")
	authAPIs := strings.Join(cfg.Service.AuthAPIs, ",")

	// 将CORS配置转换为JSON字符串
	corsConfig, _ := json.Marshal(cfg.Service.CORS)

	return map[string]string{
		"cors-enabled":   strconv.FormatBool(cfg.Service.CorsEnabled),
		"public-apis":   publicAPIs,
		"auth-required": authAPIs,
		"service-type":  cfg.Name,
		"version":       cfg.Service.Version,
		"cors-config":   string(corsConfig),
	}
}

// BuildServiceTags 构建服务标签
func BuildServiceTags(cfg *config.Config) []string {
	// 为HTTP服务添加标签
	tags := make([]string, len(cfg.Service.CTags))
	copy(tags, cfg.Service.CTags)

	// 确保至少有基础标签
	if len(tags) == 0 {
		tags = []string{"http", "public-api"}
	}

	return tags
}

// GetPublicEndpoints 获取服务的公开接口信息
func (r *ConsulRegistry) GetPublicEndpoints(serviceName string) ([]string, error) {
	services, _, err := r.client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return nil, err
	}

	var endpoints []string
	for _, service := range services {
		if service.Service.Meta != nil {
			if publicAPIs, ok := service.Service.Meta["public-apis"]; ok {
				// 分割接口字符串
				apiList := strings.Split(publicAPIs, ",")
				for _, api := range apiList {
					endpoints = append(endpoints, strings.TrimSpace(api))
				}
			}
		}
	}

	return endpoints, nil
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

