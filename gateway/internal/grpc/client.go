package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"gateway/internal/logger"
	"gateway/internal/middleware"
	"gateway/internal/registry"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	pbAdmin "github.com/Erain-byte/GrpcAi/proto/admin"
	pbAi "github.com/Erain-byte/GrpcAi/proto/ai"
	pbUser "github.com/Erain-byte/GrpcAi/proto/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// GrpcConfig gRPC 客户端配置
type GrpcConfig struct {
	UseTLS           bool   // 是否启用 TLS
	InsecureSkipVerify bool // 跳过证书验证（仅用于开发环境）
	CertFile         string // 客户端证书文件路径
	KeyFile          string // 客户端私钥文件路径
	CaFile           string // CA 证书文件路径
	ServerName       string // TLS Server Name
}

// ClientManager gRPC 客户端连接管理器
type ClientManager struct {
	registry *registry.ConsulRegistry
	config   *GrpcConfig
	mu       sync.RWMutex
	conns    map[string]*grpc.ClientConn
	rrCount  uint64
}

// NewClientManager 创建客户端管理器实例
func NewClientManager(reg *registry.ConsulRegistry, config *GrpcConfig) *ClientManager {
	if config == nil {
		// 默认配置：不使用 TLS
		config = &GrpcConfig{
			UseTLS: false,
		}
	}

	// ⭐ 证书自动推导机制：当证书路径未明确指定时，根据服务名自动推导
	if config.UseTLS && config.CertFile == "" && config.KeyFile == "" {
		logger.SugaredLogger.Warn("TLS enabled but certificate paths not specified, using default paths")
		// 默认证书路径：/etc/certs/{service-name}/
		defaultCertDir := "/etc/certs/gateway"
		config.CertFile = fmt.Sprintf("%s/client.crt", defaultCertDir)
		config.KeyFile = fmt.Sprintf("%s/client.key", defaultCertDir)
		if config.CaFile == "" {
			config.CaFile = fmt.Sprintf("%s/ca.crt", defaultCertDir)
		}
		if config.ServerName == "" {
			config.ServerName = "gateway-service"
		}
	}

	return &ClientManager{
		registry: reg,
		config:   config,
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

	// 根据配置创建传输凭证
	creds, err := m.createTransportCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to create transport credentials: %v", err)
	}

	conn, err = grpc.NewClient(addr,
		grpc.WithTransportCredentials(creds),
		grpc.WithStatsHandler(middleware.NewClientStatsHandler()), // ⭐ 添加客户端追踪 Stats Handler
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect gRPC service %s (%s): %v", serviceName, addr, err)
	}

	m.conns[key] = conn
	logger.SugaredLogger.Infof("gRPC connection established: %s -> %s (TLS: %v)", serviceName, addr, m.config.UseTLS)
	return conn, nil
}

// createTransportCredentials 创建传输凭证
func (m *ClientManager) createTransportCredentials() (credentials.TransportCredentials, error) {
	if !m.config.UseTLS {
		// 开发环境：使用不安全的连接
		log.Println("WARNING: Using insecure gRPC connection (no TLS)")
		return insecure.NewCredentials(), nil
	}

	// 生产环境：使用 TLS
	if m.config.CertFile != "" && m.config.KeyFile != "" {
		// 双向 TLS 认证（mTLS）
		cert, err := tls.LoadX509KeyPair(m.config.CertFile, m.config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %v", err)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			ServerName:   m.config.ServerName,
		}

		if m.config.InsecureSkipVerify {
			logger.SugaredLogger.Warn("TLS certificate verification is disabled")
			tlsConfig.InsecureSkipVerify = true
		}

		if m.config.CaFile != "" {
			// 加载自定义 CA 证书
			caCert, err := os.ReadFile(m.config.CaFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA certificate: %v", err)
			}

			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse CA certificate")
			}

			tlsConfig.RootCAs = caCertPool
			logger.SugaredLogger.Infof("Loaded custom CA certificate: %s", m.config.CaFile)
		}

		return credentials.NewTLS(tlsConfig), nil
	}

	// 单向 TLS（仅服务器认证）
	tlsConfig := &tls.Config{
		ServerName: m.config.ServerName,
	}

	if m.config.InsecureSkipVerify {
		logger.SugaredLogger.Warn("TLS certificate verification is disabled")
		tlsConfig.InsecureSkipVerify = true
	}

	if m.config.CaFile != "" {
		// 加载自定义 CA 证书
		caCert, err := os.ReadFile(m.config.CaFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %v", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		tlsConfig.RootCAs = caCertPool
		logger.SugaredLogger.Infof("Loaded custom CA certificate: %s", m.config.CaFile)
	}

	return credentials.NewTLS(tlsConfig), nil
}

// CheckCertificateExpiry 检查证书有效期（健康检查）
func (m *ClientManager) CheckCertificateExpiry() error {
	if !m.config.UseTLS {
		return nil // 未启用 TLS，无需检查
	}

	if m.config.CertFile == "" {
		return nil // 单向 TLS，无客户端证书
	}

	// 读取客户端证书
	cert, err := tls.LoadX509KeyPair(m.config.CertFile, m.config.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to load certificate: %v", err)
	}

	// 解析证书
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %v", err)
	}

	// 检查有效期
	now := time.Now()
	daysUntilExpiry := x509Cert.NotAfter.Sub(now).Hours() / 24

	if daysUntilExpiry < 7 {
		logger.SugaredLogger.Warnf("Certificate expiring soon: %.1f days remaining", daysUntilExpiry)
		return fmt.Errorf("certificate expiring in %.1f days", daysUntilExpiry)
	}

	logger.SugaredLogger.Debugf("Certificate valid for %.1f days", daysUntilExpiry)
	return nil
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
