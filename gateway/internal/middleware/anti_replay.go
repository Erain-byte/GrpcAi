package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"gateway/internal/config"

	"github.com/gin-gonic/gin"
)

// NonceCache Nonce 缓存（线程安全）
type NonceCache struct {
	cache      map[string]int64 // nonce -> timestamp
	mu         sync.RWMutex
	maxSize    int
	expireTime int64 // 过期时间（秒）
}

// NewNonceCache 创建 Nonce 缓存
func NewNonceCache(maxSize int, expireTime int) *NonceCache {
	return &NonceCache{
		cache:      make(map[string]int64),
		maxSize:    maxSize,
		expireTime: int64(expireTime),
	}
}

// Add 添加 nonce，如果已存在返回 false
func (nc *NonceCache) Add(nonce string, timestamp int64) bool {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	// 检查是否已存在
	if _, exists := nc.cache[nonce]; exists {
		return false
	}

	// 如果缓存已满，清理过期的 nonce
	if len(nc.cache) >= nc.maxSize {
		nc.cleanup(time.Now().Unix())
	}

	// 添加新的 nonce
	nc.cache[nonce] = timestamp
	return true
}

// Exists 检查 nonce 是否存在
func (nc *NonceCache) Exists(nonce string) bool {
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	_, exists := nc.cache[nonce]
	return exists
}

// cleanup 清理过期的 nonce
func (nc *NonceCache) cleanup(currentTime int64) {
	for nonce, timestamp := range nc.cache {
		if currentTime-timestamp > nc.expireTime {
			delete(nc.cache, nonce)
		}
	}
}

// AntiReplayMiddleware 防重放中间件
func AntiReplayMiddleware(cfg config.AntiReplayConfig) gin.HandlerFunc {
	nonceCache := NewNonceCache(cfg.NonceCacheSize, cfg.NonceExpireTime)

	return func(c *gin.Context) {
		// 如果未启用，直接跳过
		if !cfg.Enabled {
			c.Next()
			return
		}

		// 获取请求头中的签名信息
		timestampStr := c.GetHeader("X-Timestamp")
		nonce := c.GetHeader("X-Nonce")
		signature := c.GetHeader("X-Signature")

		// 检查必需的请求头是否存在
		if timestampStr == "" || nonce == "" || signature == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "Missing required headers: X-Timestamp, X-Nonce, X-Signature",
			})
			c.Abort()
			return
		}

		// 解析时间戳
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "Invalid timestamp format",
			})
			c.Abort()
			return
		}

		// 验证时间戳是否在允许的时间窗口内
		currentTime := time.Now().Unix()
		timeDiff := currentTime - timestamp

		// 检查时间戳是否过期（允许前后浮动）
		if timeDiff < 0 {
			timeDiff = -timeDiff
		}

		if timeDiff > int64(cfg.TimestampTolerance) {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": fmt.Sprintf("Request timestamp expired or too far in the future (tolerance: %d seconds)", cfg.TimestampTolerance),
			})
			c.Abort()
			return
		}

		// 检查 nonce 是否已被使用（防重放）
		if !nonceCache.Add(nonce, timestamp) {
			c.JSON(http.StatusConflict, gin.H{
				"code":    409,
				"message": "Duplicate request detected (nonce already used)",
			})
			c.Abort()
			return
		}

		// 验证签名（可选，如果配置了密钥）
		// TODO: 如果需要签名验证，在这里实现
		// expectedSignature := generateSignature(timestampStr, nonce, secretKey)
		// if signature != expectedSignature {
		//     c.JSON(http.StatusUnauthorized, gin.H{
		//         "code":    401,
		//         "message": "Invalid signature",
		//     })
		//     c.Abort()
		//     return
		// }

		c.Next()
	}
}

// GenerateSignature 生成 HMAC-SHA256 签名（供客户端使用）
func GenerateSignature(timestamp string, nonce string, secretKey string) string {
	// 构造签名字符串
	message := fmt.Sprintf("%s:%s", timestamp, nonce)

	// 创建 HMAC-SHA256
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(message))

	// 返回十六进制编码的签名
	return hex.EncodeToString(h.Sum(nil))
}
