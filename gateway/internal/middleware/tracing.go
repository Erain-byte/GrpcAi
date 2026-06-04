package middleware

import (
	"gateway/internal/logger"
	"gateway/internal/tracer"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware 链路追踪中间件
func TracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头提取 Trace Context（如果有）
		ctx := c.Request.Context()

		// 创建 Span
		spanName := c.Request.Method + " " + c.FullPath()
		if spanName == "" {
			spanName = c.Request.Method + " " + c.Request.URL.Path
		}

		ctx, span := tracer.StartSpan(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.url", c.Request.URL.String()),
				attribute.String("http.path", c.Request.URL.Path),
				attribute.String("client.ip", c.ClientIP()),
				attribute.String("user_agent", c.Request.UserAgent()),
			),
		)
		defer span.End()

		// 将 context 和 span 注入到 Gin Context
		c.Request = c.Request.WithContext(ctx)
		c.Set("span", span)

		// 处理请求
		c.Next()

		// 记录响应状态码
		statusCode := c.Writer.Status()
		span.SetAttributes(
			attribute.Int("http.status_code", statusCode),
		)

		// 如果有错误，记录到 Span
		if statusCode >= 500 {
			span.RecordError(nil)
			span.SetAttributes(
				attribute.Bool("error", true),
			)
		}

		// 添加 Trace ID 到响应头（便于前端关联日志）
		traceID := span.SpanContext().TraceID().String()
		c.Header("X-Trace-ID", traceID)

		// 记录 Trace ID 到日志
		logger.SugaredLogger.Debugf("Request traced [trace_id=%s]", traceID)
	}
}
