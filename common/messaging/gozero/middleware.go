package gozero

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

// NewGoZeroMiddleware 创建 Go-Zero trace_id 传播中间件
// 该中间件从 Watermill 消息的 Metadata 中提取 trace_id 并注入到 context
func NewGoZeroMiddleware(serviceName string) message.HandlerMiddleware {
	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			// 从消息中提取 trace_id 并注入到 context
			ctx := msg.Context()

			// 提取 trace_id
			if traceID := msg.Metadata.Get("trace_id"); traceID != "" {
				ctx = WithTraceID(ctx, traceID)
			}

			// 提取 span_id
			if spanID := msg.Metadata.Get("span_id"); spanID != "" {
				ctx = WithSpanID(ctx, spanID)
			}

			// 注入服务名称
			if serviceName != "" {
				ctx = WithServiceName(ctx, serviceName)
			}

			// 提取 source_service
			if sourceService := msg.Metadata.Get("source_service"); sourceService != "" {
				ctx = context.WithValue(ctx, "source_service", sourceService)
			}

			// 更新消息的 context
			msg.SetContext(ctx)

			// 调用下一个处理器
			return h(msg)
		}
	}
}

// InjectTraceID 发布消息时注入 trace_id 到 Watermill 消息
// 从 context 中提取 trace_id、span_id 等信息并添加到消息 Metadata
func InjectTraceID(ctx context.Context, msg *message.Message) {
	// 注入 trace_id
	if traceID := GetTraceID(ctx); traceID != "" {
		msg.Metadata.Set("trace_id", traceID)
	}

	// 注入 span_id
	if spanID := GetSpanID(ctx); spanID != "" {
		msg.Metadata.Set("span_id", spanID)
	}

	// 注入 service_name
	if serviceName := GetServiceName(ctx); serviceName != "" {
		msg.Metadata.Set("source_service", serviceName)
	}
}
