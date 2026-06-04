package middleware

import (
	"fmt"
	"net/http"
	"sync"

	"gateway/internal/config"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter 限流器结构
type RateLimiter struct {
	limiter *rate.Limiter
}

// IPRateLimiter 按 IP 限流的映射表
type IPRateLimiter struct {
	limiters map[string]*RateLimiter
	mu       sync.RWMutex
	cfg      config.RateLimitConfig
}

// NewIPRateLimiter 创建基于 IP 的限流器
func NewIPRateLimiter(cfg config.RateLimitConfig) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*RateLimiter),
		cfg:      cfg,
	}
}

// GetLimiter 获取或创建指定 key 的限流器
func (irl *IPRateLimiter) GetLimiter(key string) *rate.Limiter {
	irl.mu.RLock()
	if limiter, exists := irl.limiters[key]; exists {
		irl.mu.RUnlock()
		return limiter.limiter
	}
	irl.mu.RUnlock()

	irl.mu.Lock()
	defer irl.mu.Unlock()

	// 双重检查
	if limiter, exists := irl.limiters[key]; exists {
		return limiter.limiter
	}

	// 创建新的限流器
	limiter := rate.NewLimiter(rate.Limit(irl.cfg.RequestsPerSecond), irl.cfg.BurstSize)
	irl.limiters[key] = &RateLimiter{limiter: limiter}

	return limiter
}

// Cleanup 清理过期的限流器（防止内存泄漏）
func (irl *IPRateLimiter) Cleanup() {
	irl.mu.Lock()
	defer irl.mu.Unlock()

	// 简单实现：清空所有限流器
	// 生产环境可以使用 LRU 缓存或定时清理策略
	if len(irl.limiters) > 10000 {
		irl.limiters = make(map[string]*RateLimiter)
	}
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware(cfg config.RateLimitConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	ipLimiter := NewIPRateLimiter(cfg)

	return func(c *gin.Context) {
		var key string

		// 根据配置选择限流维度
		if cfg.ByIP {
			key = c.ClientIP()
		} else if cfg.ByAPI {
			key = c.Request.URL.Path
		} else {
			// 默认按 IP 限流
			key = c.ClientIP()
		}

		limiter := ipLimiter.GetLimiter(key)

		// 尝试获取令牌
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    http.StatusTooManyRequests,
				"message": "Too many requests, please try again later",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GlobalRateLimitMiddleware 全局限流中间件（所有请求共享一个令牌桶）
func GlobalRateLimitMiddleware(cfg config.RateLimitConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// 创建全局限流器
	limiter := rate.NewLimiter(rate.Limit(cfg.RequestsPerSecond), cfg.BurstSize)

	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    http.StatusTooManyRequests,
				"message": fmt.Sprintf("Global rate limit exceeded (%.0f req/s)", cfg.RequestsPerSecond),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
