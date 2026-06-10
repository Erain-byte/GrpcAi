package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"gateway/internal/config"
	"gateway/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type NonceCache struct {
	cache      map[string]int64
	mu         sync.RWMutex
	maxSize    int
	expireTime int64
}

func NewNonceCache(maxSize int, expireTime int) *NonceCache {
	if maxSize <= 0 {
		maxSize = 10000
	}
	if expireTime <= 0 {
		expireTime = 600
	}

	return &NonceCache{
		cache:      make(map[string]int64),
		maxSize:    maxSize,
		expireTime: int64(expireTime),
	}
}

func (nc *NonceCache) Add(nonce string, timestamp int64) bool {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	if _, exists := nc.cache[nonce]; exists {
		return false
	}

	now := time.Now().Unix()
	if len(nc.cache) >= nc.maxSize {
		nc.cleanup(now)
	}
	if len(nc.cache) >= nc.maxSize {
		nc.dropOldest()
	}

	nc.cache[nonce] = timestamp
	return true
}

func (nc *NonceCache) Exists(nonce string) bool {
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	_, exists := nc.cache[nonce]
	return exists
}

func (nc *NonceCache) cleanup(currentTime int64) {
	for nonce, timestamp := range nc.cache {
		if currentTime-timestamp > nc.expireTime {
			delete(nc.cache, nonce)
		}
	}
}

func (nc *NonceCache) dropOldest() {
	var oldestNonce string
	var oldestTime int64
	for nonce, timestamp := range nc.cache {
		if oldestNonce == "" || timestamp < oldestTime {
			oldestNonce = nonce
			oldestTime = timestamp
		}
	}
	if oldestNonce != "" {
		delete(nc.cache, oldestNonce)
	}
}

func AntiReplayMiddleware(cfg config.AntiReplayConfig, redisClients ...redis.Cmdable) gin.HandlerFunc {
	nonceCache := NewNonceCache(cfg.NonceCacheSize, cfg.NonceExpireTime)
	secret := cfg.Secret
	var redisClient redis.Cmdable
	if len(redisClients) > 0 {
		redisClient = redisClients[0]
	}

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		timestampStr := c.GetHeader("X-Timestamp")
		nonce := c.GetHeader("X-Nonce")
		signature := c.GetHeader("X-Signature")

		if timestampStr == "" || nonce == "" || signature == "" {
			AuditReject(c, "anti_replay", "missing_required_headers", http.StatusBadRequest)
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    http.StatusBadRequest,
				"message": "Missing required headers: X-Timestamp, X-Nonce, X-Signature",
			})
			c.Abort()
			return
		}

		if secret == "" {
			logger.SugaredLogger.Warn("Anti-replay is enabled but signature secret is empty")
			AuditReject(c, "anti_replay", "secret_not_configured", http.StatusInternalServerError)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": "Anti-replay signature secret is not configured",
			})
			c.Abort()
			return
		}

		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			AuditReject(c, "anti_replay", "invalid_timestamp", http.StatusBadRequest)
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    http.StatusBadRequest,
				"message": "Invalid timestamp format",
			})
			c.Abort()
			return
		}

		timeDiff := time.Now().Unix() - timestamp
		if timeDiff < 0 {
			timeDiff = -timeDiff
		}
		if timeDiff > int64(cfg.TimestampTolerance) {
			AuditReject(c, "anti_replay", "timestamp_out_of_window", http.StatusBadRequest)
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    http.StatusBadRequest,
				"message": fmt.Sprintf("Request timestamp expired or too far in the future (tolerance: %d seconds)", cfg.TimestampTolerance),
			})
			c.Abort()
			return
		}

		expectedSignature := GenerateSignature(timestampStr, nonce, secret)
		if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
			AuditReject(c, "anti_replay", "invalid_signature", http.StatusUnauthorized)
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "Invalid signature",
			})
			c.Abort()
			return
		}

		ok, err := addNonce(c.Request.Context(), redisClient, nonceCache, nonce, timestamp, cfg.NonceExpireTime, cfg.FallbackToLocal)
		if err != nil {
			AuditReject(c, "anti_replay", "nonce_store_unavailable", http.StatusServiceUnavailable)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"code":    http.StatusServiceUnavailable,
				"message": "Anti-replay nonce store unavailable",
			})
			c.Abort()
			return
		}

		if !ok {
			AuditReject(c, "anti_replay", "duplicate_nonce", http.StatusConflict)
			c.JSON(http.StatusConflict, gin.H{
				"code":    http.StatusConflict,
				"message": "Duplicate request detected (nonce already used)",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func addNonce(ctx context.Context, redisClient redis.Cmdable, fallback *NonceCache, nonce string, timestamp int64, expireSeconds int, fallbackToLocal bool) (bool, error) {
	if redisClient == nil {
		return fallback.Add(nonce, timestamp), nil
	}

	if expireSeconds <= 0 {
		expireSeconds = 600
	}

	key := "gateway:anti_replay:nonce:" + nonce
	ok, err := redisClient.SetNX(ctx, key, timestamp, time.Duration(expireSeconds)*time.Second).Result()
	if err == nil {
		return ok, nil
	}

	if !fallbackToLocal {
		return false, err
	}

	logger.SugaredLogger.Warnf("Redis anti-replay nonce write failed, falling back to local cache: %v", err)
	return fallback.Add(nonce, timestamp), nil
}

func GenerateSignature(timestamp string, nonce string, secretKey string) string {
	message := fmt.Sprintf("%s:%s", timestamp, nonce)
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
