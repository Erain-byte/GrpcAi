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

	"google.golang.org/grpc"

	"gateway/internal/config"
	grpcClient "gateway/internal/grpc"
	"gateway/internal/registry"
	"gateway/internal/server"
	"gateway/internal/svc"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "f", "etc/gateway.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Init(configFile)
	if err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	serviceContext := svc.NewServiceContext(*cfg)

	consulRegistry, err := registry.NewConsulRegistry(cfg.Consul)
	if err != nil {
		log.Fatalf("Failed to create consul registry: %v", err)
	}
	serviceContext.Registry = consulRegistry

	// ========== HTTP服务 ==========
	engine := server.NewServer(serviceContext)
	httpAddr := fmt.Sprintf(":%d", cfg.Port)
	httpSrv := &http.Server{
		Addr:    httpAddr,
		Handler: engine.GetEngine(),
	}

	go func() {
		log.Printf("Gateway HTTP Server starting at %s...", httpAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// ========== gRPC服务 ==========
	grpcAddr := fmt.Sprintf(":%d", cfg.GRPCPort)
	grpcSrv := grpc.NewServer()
	
	// 创建客户端管理器
	grpcClients := grpcClient.NewClientManager(consulRegistry)
	serviceContext.GrpcClients = grpcClients
	
	// 注册所有 gRPC 服务
	server.RegisterAllGRPCServices(grpcSrv, serviceContext, grpcClients)

	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("Failed to listen gRPC: %v", err)
	}

	go func() {
		log.Printf("Gateway gRPC Server starting at %s...", grpcAddr)
		if err := grpcSrv.Serve(grpcListener); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	// ========== Consul注册 ==========
	metadata := registry.BuildServiceMetadata(cfg)
	if err := consulRegistry.Register(cfg.Name, cfg.Host, cfg.Port, cfg.GRPCPort, metadata, cfg); err != nil {
		log.Fatalf("Failed to register service to consul: %v", err)
	}

	log.Printf("Routes configured: %d", len(cfg.Routes))
	for _, route := range cfg.Routes {
		log.Printf("  - %s -> %s (strip_path: %v, timeout: %s)", route.Path, route.Service, route.StripPath, route.Timeout)
	}

	stopKeepAlive := make(chan struct{})
	go consulRegistry.KeepAlive(cfg.Name, stopKeepAlive)

	// ========== 优雅关闭 ==========
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down gateway...")

	close(stopKeepAlive)

	if err := consulRegistry.Deregister(cfg.Name); err != nil {
		log.Printf("Failed to deregister service from consul: %v", err)
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
		log.Printf("HTTP Server forced to shutdown: %v", err)
	}

	log.Println("Gateway exited")
}
