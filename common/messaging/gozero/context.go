package gozero

import (
	"context"
)

// Context keys for go-zero integration
const (
	// TraceIDKey 追踪 ID 的上下文键
	TraceIDKey = "trace_id"

	// SpanIDKey Span ID 的上下文键
	SpanIDKey = "span_id"

	// ServiceNameKey 服务名称的上下文键
	ServiceNameKey = "service_name"
)

// GetTraceID 从上下文中获取 trace_id
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	// 尝试从 context.Value 获取
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}

	// 尝试从 "x-trace-id" 获取（兼容不同的命名）
	if traceID, ok := ctx.Value("x-trace-id").(string); ok {
		return traceID
	}

	return ""
}

// GetSpanID 从上下文中获取 span_id
func GetSpanID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if spanID, ok := ctx.Value(SpanIDKey).(string); ok {
		return spanID
	}

	return ""
}

// GetServiceName 从上下文中获取服务名称
func GetServiceName(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if serviceName, ok := ctx.Value(ServiceNameKey).(string); ok {
		return serviceName
	}

	return ""
}

// WithTraceID 将 trace_id 注入到上下文
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// WithSpanID 将 span_id 注入到上下文
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, SpanIDKey, spanID)
}

// WithServiceName 将服务名称注入到上下文
func WithServiceName(ctx context.Context, serviceName string) context.Context {
	return context.WithValue(ctx, ServiceNameKey, serviceName)
}
