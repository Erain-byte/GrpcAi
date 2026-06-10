package registry

import (
	"fmt"
	"gateway/internal/config"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
)

type ConsulRegistry struct {
	client *api.Client
	config config.ConsulConfig

	// 服务发现缓存
	cacheMu         sync.RWMutex
	serviceCache    map[string]*serviceCacheEntry
	watchMu         sync.Mutex
	watchedServices map[string]struct{}
}

// serviceCacheEntry 服务缓存条目
type serviceCacheEntry struct {
	entries   []*api.ServiceEntry
	lastIndex uint64
	updatedAt time.Time
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
		client:          client,
		config:          cfg,
		serviceCache:    make(map[string]*serviceCacheEntry),
		watchedServices: make(map[string]struct{}),
	}, nil
}

// Register 注册Gateway服务到Consul
func (r *ConsulRegistry) Register(name string, host string, httpPort int, grpcPort int, metadata map[string]string, cfg *config.Config) error {
	// 注册HTTP服务
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
			TLSSkipVerify:                  true,
			DeregisterCriticalServiceAfter: r.config.DeregisterCriticalAfter,
		},
	}

	if err := r.client.Agent().ServiceRegister(httpRegistration); err != nil {
		return fmt.Errorf("failed to register HTTP service: %v", err)
	}
	log.Printf("Gateway HTTP service registered to Consul: %s (host: %s, port: %d)", name, host, httpPort)

	// 注册gRPC服务
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
			DeregisterCriticalServiceAfter: r.config.DeregisterCriticalAfter,
		},
	}

	if err := r.client.Agent().ServiceRegister(grpcRegistration); err != nil {
		return fmt.Errorf("failed to register gRPC service: %v", err)
	}
	log.Printf("Gateway gRPC service registered to Consul: %s-grpc (host: %s, port: %d)", name, host, grpcPort)

	return nil
}

// BuildServiceMetadata 构建Gateway服务元数据
func BuildServiceMetadata(cfg *config.Config) map[string]string {
	publicAPIs := strings.Join(cfg.Service.PublicAPIs, ",")
	authAPIs := strings.Join(cfg.Service.AuthAPIs, ",")

	return map[string]string{
		"public-apis":   publicAPIs,
		"auth-required": authAPIs,
		"service-type":  cfg.Name,
		"version":       cfg.Service.Version,
	}
}

// BuildServiceTags 构建服务标签
func BuildServiceTags(cfg *config.Config) []string {
	tags := make([]string, len(cfg.Service.CTags))
	copy(tags, cfg.Service.CTags)

	if len(tags) == 0 {
		tags = []string{"http", "gateway"}
	}

	return tags
}

// DiscoverService 从Consul发现服务实例（带缓存和Watch机制）
func (r *ConsulRegistry) DiscoverService(serviceName string) ([]*api.ServiceEntry, error) {
	// 先尝试从缓存读取
	if entries := r.getFromCache(serviceName); entries != nil {
		return entries, nil
	}

	// 缓存未命中，从 Consul 查询并启动 Watch
	entries, lastIndex, err := r.queryConsul(serviceName, 0)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	r.updateCache(serviceName, entries, lastIndex)

	r.startWatch(serviceName, lastIndex)

	return entries, nil
}

func (r *ConsulRegistry) startWatch(serviceName string, lastIndex uint64) {
	r.watchMu.Lock()
	defer r.watchMu.Unlock()

	if _, ok := r.watchedServices[serviceName]; ok {
		return
	}

	r.watchedServices[serviceName] = struct{}{}
	go r.watchService(serviceName, lastIndex)
}

// getFromCache 从缓存获取服务实例
func (r *ConsulRegistry) getFromCache(serviceName string) []*api.ServiceEntry {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	entry, ok := r.serviceCache[serviceName]
	if !ok {
		return nil
	}

	// 检查缓存是否过期（30秒）
	if time.Since(entry.updatedAt) > 30*time.Second {
		return nil
	}

	return entry.entries
}

// updateCache 更新服务缓存
func (r *ConsulRegistry) updateCache(serviceName string, entries []*api.ServiceEntry, lastIndex uint64) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	r.serviceCache[serviceName] = &serviceCacheEntry{
		entries:   entries,
		lastIndex: lastIndex,
		updatedAt: time.Now(),
	}
}

// queryConsul 查询 Consul 服务
func (r *ConsulRegistry) queryConsul(serviceName string, waitIndex uint64) ([]*api.ServiceEntry, uint64, error) {
	opts := &api.QueryOptions{
		WaitIndex: waitIndex,
		WaitTime:  30 * time.Second, // Blocking Query 超时时间
	}

	services, meta, err := r.client.Health().Service(serviceName, "", true, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to discover service %s: %v", serviceName, err)
	}

	if len(services) == 0 {
		return nil, 0, fmt.Errorf("no healthy instances found for service: %s", serviceName)
	}

	return services, meta.LastIndex, nil
}

// watchService 后台监听服务变化
func (r *ConsulRegistry) watchService(serviceName string, lastIndex uint64) {
	for {
		entries, newIndex, err := r.queryConsul(serviceName, lastIndex)
		if err != nil {
			log.Printf("Watch service %s failed: %v, retry in 5s", serviceName, err)
			time.Sleep(5 * time.Second)
			continue
		}

		// 只有索引变化时才更新缓存
		if newIndex > lastIndex {
			r.updateCache(serviceName, entries, newIndex)
			lastIndex = newIndex
			log.Printf("Service %s updated: %d instances", serviceName, len(entries))
		}

		// 如果索引没变，queryConsul 会阻塞等待变化或超时
		// 超时后继续下一轮循环
	}
}

// GetServiceMetadata 获取服务的元数据（公开接口、CORS配置等）
func (r *ConsulRegistry) GetServiceMetadata(serviceName string) (map[string]string, error) {
	services, err := r.DiscoverService(serviceName)
	if err != nil {
		return nil, err
	}

	// 返回第一个健康实例的元数据
	if len(services) > 0 && services[0].Service.Meta != nil {
		return services[0].Service.Meta, nil
	}

	return nil, fmt.Errorf("no metadata found for service: %s", serviceName)
}

// GetPublicEndpoints 获取服务的公开接口列表
func (r *ConsulRegistry) GetPublicEndpoints(serviceName string) ([]string, error) {
	metadata, err := r.GetServiceMetadata(serviceName)
	if err != nil {
		return nil, err
	}

	if publicAPIs, ok := metadata["public-apis"]; ok {
		apiList := strings.Split(publicAPIs, ",")
		var endpoints []string
		for _, api := range apiList {
			if trimmed := strings.TrimSpace(api); trimmed != "" {
				endpoints = append(endpoints, trimmed)
			}
		}
		return endpoints, nil
	}

	return []string{}, nil
}

// KeepAlive TTL心跳保活
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
			if err := r.client.Agent().UpdateTTL(httpCheckID, "ok", "pass"); err != nil {
				log.Printf("Failed to update HTTP TTL: %v", err)
			}
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
	if err := r.client.Agent().ServiceDeregister(fmt.Sprintf("%s-http", name)); err != nil {
		log.Printf("Failed to deregister HTTP service: %v", err)
	} else {
		log.Printf("HTTP service deregistered from Consul: %s", name)
	}

	if err := r.client.Agent().ServiceDeregister(fmt.Sprintf("%s-grpc", name)); err != nil {
		log.Printf("Failed to deregister gRPC service: %v", err)
	} else {
		log.Printf("gRPC service deregistered from Consul: %s-grpc", name)
	}

	return nil
}
