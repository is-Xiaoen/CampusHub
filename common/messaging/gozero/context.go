package gozero

import (
	"context"

	"CampusHub/common/messaging"
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

// InjectTraceContext 将追踪上下文注入到消息元数据
// 从 context 中提取 trace_id、span_id 等信息，并添加到消息元数据中
func InjectTraceContext(ctx context.Context, msg *messaging.Message) {
	// 提取 trace_id
	if traceID := GetTraceID(ctx); traceID != "" {
		msg.Metadata.Set(messaging.MetadataKeyTraceID, traceID)
	}

	// 提取 span_id
	if spanID := GetSpanID(ctx); spanID != "" {
		msg.Metadata.Set("span_id", spanID)
	}

	// 提取 service_name
	if serviceName := GetServiceName(ctx); serviceName != "" {
		msg.Metadata.Set(messaging.MetadataKeySourceService, serviceName)
	}
}

// ExtractTraceContext 从消息元数据中提取追踪上下文
// 将消息元数据中的 trace_id、span_id 等信息注入到 context 中
func ExtractTraceContext(ctx context.Context, msg *messaging.Message) context.Context {
	// 注入 trace_id
	if traceID := msg.Metadata.Get(messaging.MetadataKeyTraceID); traceID != "" {
		ctx = WithTraceID(ctx, traceID)
	}

	// 注入 span_id
	if spanID := msg.Metadata.Get("span_id"); spanID != "" {
		ctx = WithSpanID(ctx, spanID)
	}

	// 注入 source_service
	if sourceService := msg.Metadata.Get(messaging.MetadataKeySourceService); sourceService != "" {
		ctx = context.WithValue(ctx, "source_service", sourceService)
	}

	return ctx
}

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
