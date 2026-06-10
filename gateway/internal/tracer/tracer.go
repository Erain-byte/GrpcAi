package tracer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"

	"gateway/internal/config"
	"gateway/internal/logger"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/credentials"
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
		Tracer = otel.Tracer(cfg.ServiceName)
		return nil
	}

	// 创建 OTLP gRPC 导出器选项
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
	}

	// 配置 TLS
	if cfg.UseTLS {
		tlsConfig, err := createTLSConfig(cfg)
		if err != nil {
			return err
		}
		opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig)))
		logger.SugaredLogger.Info("Tracing TLS enabled")
	} else {
		opts = append(opts, otlptracegrpc.WithInsecure())
		logger.SugaredLogger.Warn("Tracing using insecure connection (development mode only)")
	}

	// 创建 OTLP gRPC 导出器
	ctx := context.Background()
	exporter, err := otlptracegrpc.New(ctx, opts...)
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
	if Tracer == nil {
		Tracer = otel.Tracer("gateway")
	}
	return Tracer.Start(ctx, name, opts...)
}

// createTLSConfig 创建 TLS 配置
func createTLSConfig(cfg config.TracingConfig) (*tls.Config, error) {
	// 加载 CA 证书
	caCertPool := x509.NewCertPool()
	
	// 如果指定了 CA 文件，使用指定的；否则使用默认路径
	caFile := cfg.CaFile
	if caFile == "" {
		caFile = "/etc/certs/gateway/ca.crt"
	}
	
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, err
	}

	tlsConfig := &tls.Config{
		RootCAs:    caCertPool,
		ServerName: cfg.ServerName,
	}

	// 如果配置了客户端证书（双向认证）
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}
