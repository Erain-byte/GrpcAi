package server

import (
	"context"
	"fmt"
	"gateway/internal/config"
	"gateway/internal/middleware"
	"gateway/internal/svc"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// Server HTTP 服务器结构
type Server struct {
	engine *gin.Engine
	svcCtx *svc.ServiceContext
}

// NewServer 创建 HTTP 服务器实例
func NewServer(svcCtx *svc.ServiceContext) *Server {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	// 注册中间件（顺序很重要！）
	engine.Use(middleware.Recovery())                              // 1. 异常恢复
	engine.Use(middleware.TracingMiddleware())                     // 2. ⭐ 链路追踪（在日志之前，记录完整请求链）
	engine.Use(middleware.LoggerMiddleware())                      // 3. 结构化日志记录
	engine.Use(middleware.CORS(svcCtx.Config.Service.CORS))       // 4. CORS 跨域
	engine.Use(middleware.AntiReplayMiddleware(svcCtx.Config.AntiReplay)) // 5. 防重放（在限流之前）
	engine.Use(middleware.RateLimitMiddleware(svcCtx.Config.RateLimit)) // 6. 限流（在 JWT 之前，防止未认证请求消耗资源）
	engine.Use(middleware.JWTAuth(svcCtx.Config.JWT, svcCtx.Config.Service.PublicAPIs)) // 7. JWT 认证
	engine.Use(middleware.CircuitBreakerMiddleware(svcCtx.Config.CircuitBreaker)) // 8. 熔断（在 JWT 之后，保护后端服务）

	server := &Server{
		engine: engine,
		svcCtx: svcCtx,
	}

	// 注册路由
	server.registerRoutes()

	return server
}

// registerRoutes 注册所有路由
func (s *Server) registerRoutes() {
	// 健康检查
	s.engine.GET("/health", s.healthCheck)

	// 动态路由注册
	for _, route := range s.svcCtx.Config.Routes {
		s.engine.Any(route.Path+"/*path", s.createProxyHandler(route))
	}
}

// createProxyHandler 创建反向代理处理器
func (s *Server) createProxyHandler(route config.RouteConfig) gin.HandlerFunc {
	timeout, _ := time.ParseDuration(route.Timeout)
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	return func(c *gin.Context) {
		serviceName := route.Service

		// 获取目标服务地址
		targetURL := s.getServiceURL(serviceName)
		if targetURL == "" {
			c.JSON(http.StatusBadGateway, gin.H{
				"code":    http.StatusBadGateway,
				"message": fmt.Sprintf("Service %s not available", serviceName),
			})
			return
		}

		// 解析目标 URL
		proxyURL, err := url.Parse(targetURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": "Invalid proxy configuration",
			})
			return
		}

		// 处理路径
		path := c.Param("path")
		if route.StripPath {
			path = strings.TrimPrefix(c.Request.URL.Path, route.Path)
			if path == "" {
				path = "/"
			}
		} else {
			path = c.Request.URL.Path
		}

		// 创建带超时的上下文
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// 创建反向代理
		proxy := &httputil.ReverseProxy{
			Rewrite: func(r *httputil.ProxyRequest) {
				r.SetURL(proxyURL)
				r.SetXForwarded()
				r.Out.URL.Path = path
				r.Out.URL.RawQuery = c.Request.URL.RawQuery
				r.Out = r.Out.WithContext(ctx)
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				if ctx.Err() == context.DeadlineExceeded {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusGatewayTimeout)
					w.Write([]byte(fmt.Sprintf(`{"code":504,"message":"Gateway timeout after %s"}`, timeout)))
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadGateway)
				w.Write([]byte(fmt.Sprintf(`{"code":502,"message":"Gateway error: %v"}`, err)))
			},
			Transport: transport,
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// rrCounter Round-Robin 计数器
var rrCounter uint64

// getServiceURL 从 Consul 获取服务地址（负载均衡）
func (s *Server) getServiceURL(serviceName string) string {
	if s.svcCtx.Registry == nil {
		log.Printf("Consul registry not available, cannot discover service: %s", serviceName)
		return ""
	}

	entries, err := s.svcCtx.Registry.DiscoverService(serviceName)
	if err != nil {
		log.Printf("Failed to discover service %s from Consul: %v", serviceName, err)
		return ""
	}

	// Round-Robin 负载均衡
	idx := atomic.AddUint64(&rrCounter, 1) % uint64(len(entries))
	entry := entries[idx]

	// 获取协议（默认 http）
	scheme := "http"
	if meta := entry.Service.Meta; meta != nil {
		if v, ok := meta["scheme"]; ok && v != "" {
			scheme = v
		}
	}
	
	return fmt.Sprintf("%s://%s:%d", scheme, entry.Service.Address, entry.Service.Port)
}

// healthCheck 健康检查处理器
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": s.svcCtx.Config.Name,
		"version": s.svcCtx.Config.Service.Version,
	})
}

// Run 启动 HTTP 服务器
func (s *Server) Run(addr string) error {
	fmt.Printf("Gateway HTTP Server starting at %s...\n", addr)
	return s.engine.Run(addr)
}

// GetEngine 获取 Gin 引擎实例
func (s *Server) GetEngine() *gin.Engine {
	return s.engine
}
