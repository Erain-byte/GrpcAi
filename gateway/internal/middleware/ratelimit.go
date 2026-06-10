package middleware

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"gateway/internal/config"
	"gateway/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

const redisRateLimitScript = `
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])

local bucket = redis.call("HMGET", key, "tokens", "updated_at")
local tokens = tonumber(bucket[1])
local updated_at = tonumber(bucket[2])

if tokens == nil then
  tokens = burst
end
if updated_at == nil then
  updated_at = now
end

local elapsed = math.max(0, now - updated_at) / 1000
tokens = math.min(burst, tokens + elapsed * rate)

local allowed = 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
end

redis.call("HSET", key, "tokens", tokens, "updated_at", now)
redis.call("PEXPIRE", key, ttl)

return allowed
`

type RateLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	limiters map[string]*RateLimiter
	mu       sync.RWMutex
	cfg      config.RateLimitConfig
}

func NewIPRateLimiter(cfg config.RateLimitConfig) *IPRateLimiter {
	cfg = normalizeRateLimitConfig(cfg)
	return &IPRateLimiter{
		limiters: make(map[string]*RateLimiter),
		cfg:      cfg,
	}
}

func (irl *IPRateLimiter) GetLimiter(key string) *rate.Limiter {
	now := time.Now()

	irl.mu.RLock()
	if limiter, exists := irl.limiters[key]; exists {
		irl.mu.RUnlock()

		irl.mu.Lock()
		limiter.lastSeen = now
		irl.mu.Unlock()

		return limiter.limiter
	}
	irl.mu.RUnlock()

	irl.mu.Lock()
	defer irl.mu.Unlock()

	if limiter, exists := irl.limiters[key]; exists {
		limiter.lastSeen = now
		return limiter.limiter
	}

	limiter := rate.NewLimiter(rate.Limit(irl.cfg.RequestsPerSecond), irl.cfg.BurstSize)
	irl.limiters[key] = &RateLimiter{limiter: limiter, lastSeen: now}

	return limiter
}

func (irl *IPRateLimiter) Cleanup() {
	irl.mu.Lock()
	defer irl.mu.Unlock()

	expireBefore := time.Now().Add(-10 * time.Minute)
	for key, limiter := range irl.limiters {
		if limiter.lastSeen.Before(expireBefore) {
			delete(irl.limiters, key)
		}
	}

	if len(irl.limiters) > 10000 {
		for key := range irl.limiters {
			delete(irl.limiters, key)
			if len(irl.limiters) <= 8000 {
				break
			}
		}
	}
}

func RateLimitMiddleware(cfg config.RateLimitConfig, redisClients ...redis.Cmdable) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	cfg = normalizeRateLimitConfig(cfg)
	localLimiter := NewIPRateLimiter(cfg)
	var redisClient redis.Cmdable
	if len(redisClients) > 0 {
		redisClient = redisClients[0]
	}

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			localLimiter.Cleanup()
		}
	}()

	return func(c *gin.Context) {
		key := rateLimitKey(c, cfg)
		allowed := false

		if redisClient != nil {
			var err error
			allowed, err = allowByRedis(c.Request.Context(), redisClient, key, cfg)
			if err != nil {
				if !cfg.FallbackToLocal {
					AuditReject(c, "rate_limit", "store_unavailable", http.StatusServiceUnavailable)
					c.JSON(http.StatusServiceUnavailable, gin.H{
						"code":    http.StatusServiceUnavailable,
						"message": "Rate limit store unavailable",
					})
					c.Abort()
					return
				}
				if logger.SugaredLogger != nil {
					logger.SugaredLogger.Warnf("Redis rate limit failed, falling back to local limiter: %v", err)
				}
				allowed = localLimiter.GetLimiter(key).Allow()
			}
		} else {
			allowed = localLimiter.GetLimiter(key).Allow()
		}

		if !allowed {
			AuditReject(c, "rate_limit", "limit_exceeded", http.StatusTooManyRequests)
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

func GlobalRateLimitMiddleware(cfg config.RateLimitConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	cfg = normalizeRateLimitConfig(cfg)
	limiter := rate.NewLimiter(rate.Limit(cfg.RequestsPerSecond), cfg.BurstSize)

	return func(c *gin.Context) {
		if !limiter.Allow() {
			AuditReject(c, "rate_limit", "global_limit_exceeded", http.StatusTooManyRequests)
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

func normalizeRateLimitConfig(cfg config.RateLimitConfig) config.RateLimitConfig {
	if cfg.RequestsPerSecond <= 0 {
		cfg.RequestsPerSecond = 100
	}
	if cfg.BurstSize <= 0 {
		cfg.BurstSize = 20
	}
	return cfg
}

func rateLimitKey(c *gin.Context, cfg config.RateLimitConfig) string {
	key := c.ClientIP()
	if cfg.ByIP && cfg.ByAPI {
		key = c.ClientIP() + ":" + c.Request.URL.Path
	} else if cfg.ByAPI {
		key = c.Request.URL.Path
	}
	return key
}

func allowByRedis(ctx context.Context, redisClient redis.Cmdable, key string, cfg config.RateLimitConfig) (bool, error) {
	burst := cfg.BurstSize
	ttl := redisBucketTTL(cfg)
	result, err := redisClient.Eval(ctx, redisRateLimitScript,
		[]string{"gateway:rate_limit:" + key},
		cfg.RequestsPerSecond,
		burst,
		time.Now().UnixMilli(),
		ttl.Milliseconds(),
	).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func redisBucketTTL(cfg config.RateLimitConfig) time.Duration {
	seconds := float64(cfg.BurstSize) / cfg.RequestsPerSecond * 2
	if seconds < 1 {
		seconds = 1
	}
	return time.Duration(math.Ceil(seconds)) * time.Second
}
