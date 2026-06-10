package middleware

import (
	"gateway/internal/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func AuditReject(c *gin.Context, component string, reason string, status int, fields ...zap.Field) {
	if logger.Logger == nil {
		return
	}

	baseFields := []zap.Field{
		zap.String("event", "request_rejected"),
		zap.String("component", component),
		zap.String("reason", reason),
		zap.Int("status", status),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("query", c.Request.URL.RawQuery),
		zap.String("ip", c.ClientIP()),
		zap.String("user_agent", c.Request.UserAgent()),
		logger.WithTraceID(c.Request.Context()),
	}

	logger.Logger.Warn("request rejected", append(baseFields, fields...)...)
}
