package middleware

import (
	"net/http"

	"gateway/internal/logger"
	"gateway/internal/tracer"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := otel.GetTextMapPropagator().Extract(
			c.Request.Context(),
			propagation.HeaderCarrier(c.Request.Header),
		)

		spanName := c.Request.Method + " " + c.FullPath()
		if c.FullPath() == "" {
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

		c.Request = c.Request.WithContext(ctx)
		c.Set("span", span)

		c.Next()

		statusCode := c.Writer.Status()
		span.SetAttributes(attribute.Int("http.status_code", statusCode))
		if statusCode >= http.StatusInternalServerError {
			span.SetStatus(codes.Error, http.StatusText(statusCode))
		}

		traceID := span.SpanContext().TraceID().String()
		if span.SpanContext().IsValid() {
			c.Header("X-Trace-ID", traceID)
			if logger.SugaredLogger != nil {
				logger.SugaredLogger.Debugf("Request traced [trace_id=%s]", traceID)
			}
		}
	}
}
