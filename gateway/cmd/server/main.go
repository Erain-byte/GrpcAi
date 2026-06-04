package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gateway/internal/config"
	grpcClient "gateway/internal/grpc"
	"gateway/internal/logger"
	"gateway/internal/middleware"
	"gateway/internal/registry"
	"gateway/internal/server"
	"gateway/internal/svc"
	"gateway/internal/tracer"
	_ "net/http/pprof" // 引入 pprof 支持

	"google.golang.org/grpc"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "f", "etc/gateway.yaml", "config file path") // 默认配置文件路径
	flag.Parse()                                                             // 解析命令行参数

	cfg, err := config.Init(configFile)
	if err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	// ⭐ 初始化日志系统
	if err := logger.InitLogger(cfg.Logger); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync() // 确保日志缓冲区写入磁盘

	// ⭐ 初始化链路追踪系统
	if err := tracer.InitTracer(cfg.Tracing); err != nil {
		logger.SugaredLogger.Warnf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := tracer.Shutdown(ctx); err != nil {
			logger.SugaredLogger.Warnf("Failed to shutdown tracer: %v", err)
		}
		cancel() // 立即释放 context 资源
	}()

	logger.SugaredLogger.Infof("Gateway Service starting... [version=%s]", cfg.Service.Version)

	serviceContext := svc.NewServiceContext(*cfg)                 // 创建 redis 和数据库连接等资源
	consulRegistry, err := registry.NewConsulRegistry(cfg.Consul) // 创建 Consul 注册中心实例
	if err != nil {
		logger.SugaredLogger.Fatalf("Failed to create consul registry: %v", err)
	}
	serviceContext.Registry = consulRegistry // 将 Consul 注册中心添加到服务上下文中，供后续使用

	// ========== HTTP服务 ==========
	engine := server.NewServer(serviceContext)
	httpAddr := fmt.Sprintf(":%d", cfg.Port)
	httpSrv := &http.Server{
		Addr:    httpAddr,
		Handler: engine.GetEngine(),
	}

	go startHTTPServer(httpSrv, httpAddr)

	// ========== gRPC服务 ==========
	grpcAddr := fmt.Sprintf(":%d", cfg.GRPCPort)
	grpcSrv := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.UnaryServerInterceptor()), // ⭐ 添加服务端追踪拦截器
	)

	// 创建客户端管理器（带 TLS 配置）
	grpcConfig := &grpcClient.GrpcConfig{
		UseTLS:             cfg.Grpc.UseTLS,
		InsecureSkipVerify: cfg.Grpc.InsecureSkipVerify,
		CertFile:           cfg.Grpc.CertFile,
		KeyFile:            cfg.Grpc.KeyFile,
		CaFile:             cfg.Grpc.CaFile,
		ServerName:         cfg.Grpc.ServerName,
	}
	grpcClients := grpcClient.NewClientManager(consulRegistry, grpcConfig)
	serviceContext.GrpcClients = grpcClients

	// 注册所有 gRPC 服务
	server.RegisterAllGRPCServices(grpcSrv, serviceContext, grpcClients)

	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logger.SugaredLogger.Fatalf("Failed to listen gRPC: %v", err)
	}

	go startGRPCServer(grpcSrv, grpcListener, grpcAddr)

	// ========== Consul注册 ==========
	metadata := registry.BuildServiceMetadata(cfg)
	if err := consulRegistry.Register(cfg.Name, cfg.Host, cfg.Port, cfg.GRPCPort, metadata, cfg); err != nil {
		logger.SugaredLogger.Fatalf("Failed to register service to consul: %v", err)
	}

	logger.SugaredLogger.Infof("Routes configured: %d", len(cfg.Routes))
	for _, route := range cfg.Routes {
		logger.SugaredLogger.Infof("  - %s -> %s (strip_path: %v, timeout: %s)", route.Path, route.Service, route.StripPath, route.Timeout)
	}

	stopKeepAlive := make(chan struct{})
	go consulRegistry.KeepAlive(cfg.Name, stopKeepAlive)

	// ⭐ 启动证书有效期定期检查（每24小时）
	if cfg.Grpc.UseTLS {
		quitCheck := make(chan struct{})
		go func() {
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := grpcClients.CheckCertificateExpiry(); err != nil {
						logger.SugaredLogger.Warnf("Certificate health check failed: %v", err)
					}
				case <-quitCheck:
					return
				}
			}
		}()
		logger.SugaredLogger.Info("Certificate expiry health check started (every 24 hours)")
		defer close(quitCheck)
	}

	// ========== 优雅关闭 ==========
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.SugaredLogger.Info("Shutting down gateway...")

	close(stopKeepAlive)

	if err := consulRegistry.Deregister(cfg.Name); err != nil {
		logger.SugaredLogger.Warnf("Failed to deregister service from consul: %v", err)
	}

	grpcSrv.GracefulStop()
	grpcClients.Close()

	shutdownTimeout, err := time.ParseDuration(cfg.Shutdown.Timeout)
	if err != nil {
		shutdownTimeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		logger.SugaredLogger.Warnf("HTTP Server forced to shutdown: %v", err)
	}

	logger.SugaredLogger.Info("Gateway exited")
}

// startHTTPServer 启动 HTTP 服务器（在 goroutine 中运行）
func startHTTPServer(srv *http.Server, addr string) {
	logger.SugaredLogger.Infof("Gateway HTTP Server starting at %s...", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.SugaredLogger.Fatalf("Failed to start HTTP server: %v", err)
	}
}

// startGRPCServer 启动 gRPC 服务器（在 goroutine 中运行）
func startGRPCServer(srv *grpc.Server, listener net.Listener, addr string) {
	logger.SugaredLogger.Infof("Gateway gRPC Server starting at %s...", addr)
	if err := srv.Serve(listener); err != nil {
		logger.SugaredLogger.Fatalf("Failed to start gRPC server: %v", err)
	}
}
