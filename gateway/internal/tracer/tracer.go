package tracer

import (
	"context"

	"gateway/internal/config"
	"gateway/internal/logger"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	// TracerProvider 全局 Tracer Provider
	TracerProvider *sdktrace.TracerProvider
	// Tracer 全局 Tracer 实例
	Tracer trace.Tracer
)

// InitTracer 初始化链路追踪系统
func InitTracer(cfg config.TracingConfig) error {
	if !cfg.Enabled {
		logger.SugaredLogger.Info("Tracing is disabled")
		return nil
	}

	// 创建 Jaeger 导出器
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(cfg.Endpoint)))
	if err != nil {
		return err
	}

	// 创建资源
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
		),
	)
	if err != nil {
		return err
	}

	// 创建采样器
	var sampler sdktrace.Sampler
	switch cfg.SamplerType {
	case "const":
		if cfg.SamplerParam == 1 {
			sampler = sdktrace.AlwaysSample()
		} else {
			sampler = sdktrace.NeverSample()
		}
	case "probabilistic":
		sampler = sdktrace.TraceIDRatioBased(cfg.SamplerParam)
	case "ratelimiting":
		sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SamplerParam))
	default:
		sampler = sdktrace.AlwaysSample()
	}

	// 创建 Tracer Provider
	TracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// 设置全局 Tracer Provider
	otel.SetTracerProvider(TracerProvider)

	// 设置全局传播器（支持 W3C Trace Context）
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// 创建 Tracer
	Tracer = TracerProvider.Tracer(cfg.ServiceName)

	logger.SugaredLogger.Infof("Tracing initialized [endpoint=%s, sampler=%s]", cfg.Endpoint, cfg.SamplerType)
	return nil
}

// Shutdown 关闭 Tracer Provider，确保所有 Span 被导出
func Shutdown(ctx context.Context) error {
	if TracerProvider != nil {
		return TracerProvider.Shutdown(ctx)
	}
	return nil
}

// StartSpan 创建新的 Span
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return Tracer.Start(ctx, name, opts...)
}
