package middleware

import (
	"fmt"
	"net/http"
	"time"

	"gateway/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/sony/gobreaker"
)

// CircuitBreakerMiddleware 熔断器中间件
func CircuitBreakerMiddleware(cfg config.CircuitBreakerConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// 解析超时时间
	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		timeout = 30 * time.Second // 默认 30 秒
	}

	interval, err := time.ParseDuration(cfg.Interval)
	if err != nil {
		interval = 60 * time.Second // 默认 60 秒
	}

	// 创建熔断器
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "gateway-circuit-breaker",
		MaxRequests: cfg.MinRequests,      // 半开状态允许的最大请求数
		Interval:    interval,             // 统计窗口时间
		Timeout:     timeout,              // 熔断器打开后的等待时间
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// 当失败次数超过阈值时，触发熔断
			return counts.ConsecutiveFailures > cfg.MaxFailures
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			// 状态变化时的回调（可用于监控告警）
			switch to {
			case gobreaker.StateOpen:
				// 熔断器打开，拒绝所有请求
			case gobreaker.StateHalfOpen:
				// 进入半开状态，允许少量请求探测
			case gobreaker.StateClosed:
				// 熔断器关闭，恢复正常
			}
		},
	})

	return func(c *gin.Context) {
		// 执行受保护的请求
		result, err := cb.Execute(func() (interface{}, error) {
			c.Next()
			
			// 检查是否有错误（状态码 >= 500）
			if c.Writer.Status() >= http.StatusInternalServerError {
				return nil, fmt.Errorf("backend service error: %d", c.Writer.Status())
			}
			
			return nil, nil
		})

		// 如果熔断器处于 Open 状态，返回 503
		if err != nil {
			if err == gobreaker.ErrOpenState {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"code":    http.StatusServiceUnavailable,
					"message": "Service temporarily unavailable, circuit breaker is open",
				})
				c.Abort()
				return
			}
			
			// 其他错误，记录但不中断（由后续中间件处理）
		}
		
		_ = result // 使用结果避免未使用警告
	}
}

// PerServiceCircuitBreaker 按服务划分的熔断器（用于 gRPC 转发）
type PerServiceCircuitBreaker struct {
	breakers map[string]*gobreaker.CircuitBreaker
	cfg      config.CircuitBreakerConfig
}

// NewPerServiceCircuitBreaker 创建按服务的熔断器
func NewPerServiceCircuitBreaker(cfg config.CircuitBreakerConfig) *PerServiceCircuitBreaker {
	return &PerServiceCircuitBreaker{
		breakers: make(map[string]*gobreaker.CircuitBreaker),
		cfg:      cfg,
	}
}

// GetBreaker 获取或创建指定服务的熔断器
func (pscb *PerServiceCircuitBreaker) GetBreaker(serviceName string) *gobreaker.CircuitBreaker {
	if breaker, exists := pscb.breakers[serviceName]; exists {
		return breaker
	}

	timeout, _ := time.ParseDuration(pscb.cfg.Timeout)
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	interval, _ := time.ParseDuration(pscb.cfg.Interval)
	if interval == 0 {
		interval = 60 * time.Second
	}

	breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        serviceName,
		MaxRequests: pscb.cfg.MinRequests,
		Interval:    interval,
		Timeout:     timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > pscb.cfg.MaxFailures
		},
	})

	pscb.breakers[serviceName] = breaker
	return breaker
}
