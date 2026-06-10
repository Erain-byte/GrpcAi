package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"gateway/internal/config"
	"gateway/internal/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func CORS(cfg config.CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Header("Vary", "Origin")
		}

		if allowedOrigin := resolveAllowedOrigin(origin, cfg); allowedOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
		}

		if len(cfg.AllowMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowMethods, ", "))
		}

		if len(cfg.AllowHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowHeaders, ", "))
		}

		if len(cfg.ExposeHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(cfg.ExposeHeaders, ", "))
		}

		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if cfg.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.MaxAge))
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func resolveAllowedOrigin(origin string, cfg config.CORSConfig) string {
	if len(cfg.AllowOrigins) == 0 {
		return ""
	}

	if contains(cfg.AllowOrigins, "*") {
		if cfg.AllowCredentials {
			return origin
		}
		return "*"
	}

	if contains(cfg.AllowOrigins, origin) {
		return origin
	}

	return ""
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				if logger.Logger != nil {
					logger.Logger.Error("panic recovered",
						zap.Any("error", err),
						zap.String("method", c.Request.Method),
						zap.String("path", c.Request.URL.Path),
						zap.String("client_ip", c.ClientIP()),
						zap.ByteString("stack", debug.Stack()),
						logger.WithTraceID(c.Request.Context()),
					)
				}

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    http.StatusInternalServerError,
					"message": "Internal Server Error",
				})
			}
		}()

		c.Next()
	}
}
