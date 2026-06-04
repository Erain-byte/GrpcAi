package middleware

import (
	"time"

	"gateway/internal/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoggerMiddleware 结构化日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录请求开始时间
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method
		ip := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// 处理请求
		c.Next()

		// 计算耗时
		duration := time.Since(start)

		// 获取响应状态码
		statusCode := c.Writer.Status()

		// 从 context 中获取用户信息（如果已认证）
		var userID interface{}
		if uid, exists := c.Get("user_id"); exists {
			userID = uid
		}

		// 记录结构化日志
		logger.Logger.Info("HTTP Request",
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", statusCode),
			zap.Duration("duration", duration),
			zap.String("ip", ip),
			zap.String("user_agent", userAgent),
			zap.Any("user_id", userID),
		)

		// 如果有错误，记录错误日志
		if statusCode >= 500 {
			logger.Logger.Error("HTTP Server Error",
				zap.String("method", method),
				zap.String("path", path),
				zap.Int("status", statusCode),
				zap.Duration("duration", duration),
				zap.String("ip", ip),
			)
		} else if statusCode >= 400 {
			logger.Logger.Warn("HTTP Client Error",
				zap.String("method", method),
				zap.String("path", path),
				zap.Int("status", statusCode),
				zap.Duration("duration", duration),
				zap.String("ip", ip),
			)
		}
	}
}
