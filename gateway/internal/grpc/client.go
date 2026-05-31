package grpc

import (
	"fmt"
	"gateway/internal/registry"
	"log"
	"sync"
	"sync/atomic"

	pbAdmin "github.com/Erain-byte/GrpcAi/proto/admin"
	pbAi "github.com/Erain-byte/GrpcAi/proto/ai"
	pbUser "github.com/Erain-byte/GrpcAi/proto/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ClientManager gRPC 客户端连接管理器
type ClientManager struct {
	registry *registry.ConsulRegistry
	mu       sync.RWMutex
	conns    map[string]*grpc.ClientConn
	rrCount  uint64
}

// NewClientManager 创建客户端管理器实例
func NewClientManager(reg *registry.ConsulRegistry) *ClientManager {
	return &ClientManager{
		registry: reg,
		conns:    make(map[string]*grpc.ClientConn),
	}
}

// GetConn 获取或创建 gRPC 连接（带连接池和负载均衡）
func (m *ClientManager) GetConn(serviceName string) (*grpc.ClientConn, error) {
	addr, err := m.resolveService(serviceName)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%s-%s", serviceName, addr)
	
	// 先尝试读锁获取连接
	m.mu.RLock()
	conn, ok := m.conns[key]
	m.mu.RUnlock()
	if ok {
		return conn, nil
	}

	// 写锁创建新连接
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 双重检查
	if conn, ok = m.conns[key]; ok {
		return conn, nil
	}

	conn, err = grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect gRPC service %s (%s): %v", serviceName, addr, err)
	}
	
	m.conns[key] = conn
	log.Printf("gRPC connection established: %s -> %s", serviceName, addr)
	return conn, nil
}

// resolveService 从 Consul 解析服务地址（Round-Robin 负载均衡）
func (m *ClientManager) resolveService(serviceName string) (string, error) {
	grpcServiceName := serviceName + "-grpc"
	entries, err := m.registry.DiscoverService(grpcServiceName)
	if err != nil {
		return "", fmt.Errorf("failed to discover gRPC service %s: %v", grpcServiceName, err)
	}

	// Round-Robin 选择实例
	idx := atomic.AddUint64(&m.rrCount, 1) % uint64(len(entries))
	entry := entries[idx]
	return fmt.Sprintf("%s:%d", entry.Service.Address, entry.Service.Port), nil
}

// ========== 类型安全的客户端工厂函数 ==========

// ClientFactory 通用客户端工厂函数类型
type ClientFactory[T any] func(grpc.ClientConnInterface) T

// CreateClient 通用客户端创建辅助函数（泛型函数）
func CreateClient[T any](manager *ClientManager, serviceName string, factory ClientFactory[T]) (T, error) {
	var zero T
	
	conn, err := manager.GetConn(serviceName)
	if err != nil {
		return zero, err
	}
	
	return factory(conn), nil
}

// UserClient 获取用户服务客户端
func (m *ClientManager) UserClient() (pbUser.UserServiceClient, error) {
	return CreateClient(m, "user-service", pbUser.NewUserServiceClient)
}

// AdminClient 获取管理服务客户端
func (m *ClientManager) AdminClient() (pbAdmin.AdminServiceClient, error) {
	return CreateClient(m, "admin-service", pbAdmin.NewAdminServiceClient)
}

// AiClient 获取 AI 服务客户端
func (m *ClientManager) AiClient() (pbAi.AiServiceClient, error) {
	return CreateClient(m, "ai-service", pbAi.NewAiServiceClient)
}

// Close 关闭所有 gRPC 连接
func (m *ClientManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for key, conn := range m.conns {
		if err := conn.Close(); err != nil {
			log.Printf("Failed to close gRPC connection %s: %v", key, err)
		}
	}
	m.conns = make(map[string]*grpc.ClientConn)
	log.Println("All gRPC connections closed")
}
