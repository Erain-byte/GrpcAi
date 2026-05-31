package server

import (
	"context"
	"fmt"
	"gateway/internal/svc"
	"log"
	"sync"

	grpcclient "gateway/internal/grpc"
	"google.golang.org/grpc"
)

// BaseForwarder 通用转发器基类（只负责转发逻辑）
type BaseForwarder[T any] struct {
	mu          sync.Mutex
	clients     map[string]T                    // 保存客户端实例（key: 服务地址）
	nodes       []string                        // 可用节点列表
	factory     grpcclient.ClientFactory[T]     // 客户端工厂函数
	connections []*grpc.ClientConn              // 保存所有连接，用于关闭
	svcCtx      *svc.ServiceContext             // 服务上下文，用于访问配置和注册中心
	serviceName string                          // 服务名称，用于日志和发现
	clientMgr   *grpcclient.ClientManager       // gRPC 客户端管理器
}

// NewBaseForwarder 创建通用转发器实例
func NewBaseForwarder[T any](
	svcCtx *svc.ServiceContext,
	clientMgr *grpcclient.ClientManager,
	serviceName string,
	factory grpcclient.ClientFactory[T],
) *BaseForwarder[T] {
	return &BaseForwarder[T]{
		clients:     make(map[string]T),
		svcCtx:      svcCtx,
		serviceName: serviceName,
		factory:     factory,
		clientMgr:   clientMgr,
	}
}

// GetClient 获取或创建 gRPC 客户端（带负载均衡）
func (f *BaseForwarder[T]) GetClient(ctx context.Context) (T, error) {
	var zero T

	// 从 ClientManager 获取客户端（已包含连接池和负载均衡）
	client, err := f.getClientFromManager()
	if err != nil {
		return zero, fmt.Errorf("failed to get client for service %s: %w", f.serviceName, err)
	}

	return client, nil
}

// getClientFromManager 从 ClientManager 获取客户端
func (f *BaseForwarder[T]) getClientFromManager() (T, error) {
	var zero T

	// 使用泛型工厂函数创建客户端
	client, err := grpcclient.CreateClient(f.clientMgr, f.serviceName, f.factory)
	if err != nil {
		return zero, err
	}

	return client, nil
}

// HealthCheck 检查服务健康状态
func (f *BaseForwarder[T]) HealthCheck(ctx context.Context) bool {
	if f.svcCtx.Registry == nil {
		log.Printf("[%s] Registry not available", f.serviceName)
		return false
	}

	entries, err := f.svcCtx.Registry.DiscoverService(f.serviceName)
	if err != nil || len(entries) == 0 {
		log.Printf("[%s] No healthy instances found: %v", f.serviceName, err)
		return false
	}

	log.Printf("[%s] Found %d healthy instances", f.serviceName, len(entries))
	return true
}

// Close 关闭所有连接
func (f *BaseForwarder[T]) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, conn := range f.connections {
		if err := conn.Close(); err != nil {
			log.Printf("[%s] Failed to close connection: %v", f.serviceName, err)
		}
	}

	f.connections = nil
	f.clients = make(map[string]T)
	log.Printf("[%s] All connections closed", f.serviceName)
}
