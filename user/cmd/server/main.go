package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"flag"
	"fmt"
	pb "github.com/Erain-byte/GrpcAi/proto/user"
	"user/internal/config"
	"user/internal/registry"
	"user/internal/server"
	"user/internal/svc"
	"user/pkg"
	"google.golang.org/grpc"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "f", "etc/user.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Init(configFile)
	if err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	// 初始化JWT配置
	pkg.InitJWT(cfg.JWT.Secret, cfg.JWT.Expire, cfg.Name)

	serviceContext := svc.NewServiceContext(*cfg)
	engine := server.NewServer(serviceContext)

	// ========== HTTP服务 ==========
	httpAddr := fmt.Sprintf(":%d", cfg.Port)
	httpSrv := &http.Server{
		Addr:    httpAddr,
		Handler: engine,
	}

	go func() {
		log.Printf("HTTP Server starting at %s...", httpAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// ========== gRPC服务 ==========
	grpcAddr := fmt.Sprintf(":%d", cfg.GRPCPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("Failed to listen for gRPC: %v", err)
	}

	grpcSrv := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcSrv, server.NewUserGrpcServer(serviceContext))

	go func() {
		log.Printf("gRPC Server starting at %s...", grpcAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	// ========== 服务注册到Consul ==========
	consulRegistry, err := registry.NewConsulRegistry(cfg.Consul)
	if err != nil {
		log.Fatalf("Failed to create consul registry: %v", err)
	}

	// 构建服务元数据
	metadata := registry.BuildServiceMetadata(cfg)
	if err := consulRegistry.Register(cfg.Name, cfg.Host, cfg.Port, cfg.GRPCPort, metadata, cfg); err != nil {
		log.Fatalf("Failed to register service to consul: %v", err)
	}

	// 打印公开接口信息
	publicEndpoints, err := consulRegistry.GetPublicEndpoints(cfg.Name)
	if err != nil {
		log.Printf("Failed to get public endpoints: %v", err)
	} else {
		log.Printf("Public endpoints: %v", publicEndpoints)
	}

	// 打印CORS配置信息
	if cfg.Service.CorsEnabled {
		log.Printf("CORS is enabled with configuration: %+v", cfg.Service.CORS)
	} else {
		log.Printf("CORS is disabled")
	}

	// 启动TTL心跳保活
	stopKeepAlive := make(chan struct{})
	go consulRegistry.KeepAlive(cfg.Name, stopKeepAlive)

	// ========== 优雅关闭 ==========
	quit := make(chan os.Signal, 1)
	// kill 默认发送 syscall.SIGTERM
	// kill -2 发送 syscall.SIGINT
	// kill -9 发送 syscall.SIGKILL，但无法被捕获，所以不需要添加
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// 停止TTL心跳
	close(stopKeepAlive)

	// 从Consul注销服务
	if err := consulRegistry.Deregister(cfg.Name); err != nil {
		log.Printf("Failed to deregister service from consul: %v", err)
	}

	// 优雅关闭HTTP服务器
	shutdownTimeout, err := time.ParseDuration(cfg.Shutdown.Timeout)
	if err != nil {
		shutdownTimeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Printf("HTTP Server forced to shutdown: %v", err)
	}

	// 优雅关闭gRPC服务器
	grpcSrv.GracefulStop()

	log.Println("Server exited")
}
