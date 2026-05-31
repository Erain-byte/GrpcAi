package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"gateway/internal/config"

	"github.com/gin-gonic/gin"
)

// CORS 跨域中间件 - 从配置读取CORS规则
func CORS(cfg config.CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置 Allow-Origin
		if len(cfg.AllowOrigins) > 0 {
			origin := c.Request.Header.Get("Origin")
			if contains(cfg.AllowOrigins, "*") {
				c.Header("Access-Control-Allow-Origin", "*")
			} else if contains(cfg.AllowOrigins, origin) {
				c.Header("Access-Control-Allow-Origin", origin)
			}
		}

		// 设置 Allow-Methods
		if len(cfg.AllowMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowMethods, ", "))
		}

		// 设置 Allow-Headers
		if len(cfg.AllowHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowHeaders, ", "))
		}

		// 设置 Expose-Headers
		if len(cfg.ExposeHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(cfg.ExposeHeaders, ", "))
		}

		// 设置 Allow-Credentials
		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// 设置 Max-Age
		if cfg.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.MaxAge))
		}

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// contains 检查字符串切片是否包含指定值
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Recovery 异常恢复中间件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    http.StatusInternalServerError,
					"message": "Internal Server Error",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 请求前
		path := c.Request.URL.Path
		method := c.Request.Method

		// 处理请求
		c.Next()

		// 请求后记录日志
		statusCode := c.Writer.Status()
		log.Printf("[%s] %s %d %s", method, path, statusCode, c.ClientIP())
	}
}
