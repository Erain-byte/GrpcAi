package middleware

import (
	"context"

	"gateway/internal/tracer"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

// GRPCStatsHandler gRPC Stats Handler，用于链路追踪
type GRPCStatsHandler struct {
	serviceName string
}

// NewGRPCStatsHandler 创建 gRPC Stats Handler
func NewGRPCStatsHandler(serviceName string) *GRPCStatsHandler {
	return &GRPCStatsHandler{
		serviceName: serviceName,
	}
}

// TagConn 标记连接（不需要实现）
func (h *GRPCStatsHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	return ctx
}

// HandleConn 处理连接事件（不需要实现）
func (h *GRPCStatsHandler) HandleConn(ctx context.Context, info stats.ConnStats) {
}

// TagRPC 标记 RPC 调用
func (h *GRPCStatsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	// 使用 OpenTelemetry 的传播器提取 context
	ctx, _ = tracer.StartSpan(ctx, info.FullMethodName, trace.WithSpanKind(trace.SpanKindServer))
	return ctx
}

// HandleRPC 处理 RPC 事件
func (h *GRPCStatsHandler) HandleRPC(ctx context.Context, info stats.RPCStats) {
	switch s := info.(type) {
	case *stats.End:
		// RPC 结束时记录 Span
		span := trace.SpanFromContext(ctx)
		if s.Error != nil {
			span.RecordError(s.Error)
			span.SetAttributes(attribute.Bool("error", true))
		}
		span.End()
	}
}

// UnaryServerInterceptor gRPC 一元服务器拦截器（用于追踪）
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// 创建 Span
		ctx, span := tracer.StartSpan(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("grpc.method", info.FullMethod),
			),
		)
		defer span.End()

		// 调用实际处理函数
		resp, err := handler(ctx, req)

		// 记录错误
		if err != nil {
			span.RecordError(err)
			span.SetAttributes(attribute.Bool("error", true))
		}

		return resp, err
	}
}

// UnaryClientInterceptor gRPC 一元客户端拦截器（用于传递 Trace Context）
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 创建客户端 Span
		ctx, span := tracer.StartSpan(ctx, method,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				attribute.String("grpc.method", method),
			),
		)
		defer span.End()

		// 调用实际方法
		err := invoker(ctx, method, req, reply, cc, opts...)

		// 记录错误
		if err != nil {
			span.RecordError(err)
			span.SetAttributes(attribute.Bool("error", true))
		}

		return err
	}
}
