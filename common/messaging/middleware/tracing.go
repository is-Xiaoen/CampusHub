package middleware

import (
	"context"
	"fmt"

	"activity-platform/common/messaging"
)

// TracingConfig 追踪配置
type TracingConfig struct {
	// ServiceName 服务名称
	ServiceName string

	// Enabled 是否启用追踪
	Enabled bool
}

// TracingMiddleware 创建追踪中间件
// 为消息处理创建 span，记录追踪信息
func TracingMiddleware(config TracingConfig) messaging.Middleware {
	return func(next messaging.HandlerFunc) messaging.HandlerFunc {
		return func(ctx context.Context, msg *messaging.Message) error {
			// 如果追踪未启用，直接调用处理器
			if !config.Enabled {
				return next(ctx, msg)
			}

			// 创建 span（简化实现，实际应使用 OpenTelemetry SDK）
			spanCtx := createSpan(ctx, msg, config.ServiceName)

			// 调用处理器
			err := next(spanCtx, msg)

			// 记录错误（如果有）
			if err != nil {
				recordSpanError(spanCtx, err)
			}

			// 结束 span
			endSpan(spanCtx)

			return err
		}
	}
}

// createSpan 创建 span（简化实现）
// 实际项目中应使用 OpenTelemetry SDK
func createSpan(ctx context.Context, msg *messaging.Message, serviceName string) context.Context {
	// 从消息元数据中提取 trace_id
	traceID := msg.Metadata.Get(messaging.MetadataKeyTraceID)

	// 创建 span 上下文（简化实现）
	spanCtx := context.WithValue(ctx, "span_name", fmt.Sprintf("process_message_%s", msg.Topic))
	spanCtx = context.WithValue(spanCtx, "trace_id", traceID)
	spanCtx = context.WithValue(spanCtx, "message_id", msg.ID)
	spanCtx = context.WithValue(spanCtx, "topic", msg.Topic)
	spanCtx = context.WithValue(spanCtx, "service_name", serviceName)

	return spanCtx
}

// recordSpanError 记录 span 错误（简化实现）
func recordSpanError(ctx context.Context, err error) {
	// 实际项目中应使用 OpenTelemetry SDK 记录错误
	_ = ctx
	_ = err
}

// endSpan 结束 span（简化实现）
func endSpan(ctx context.Context) {
	// 实际项目中应使用 OpenTelemetry SDK 结束 span
	_ = ctx
}

// 注意：这是一个简化的实现示例
// 实际项目中应该使用完整的 OpenTelemetry SDK，例如：
//
// import (
//     "go.opentelemetry.io/otel"
//     "go.opentelemetry.io/otel/attribute"
//     "go.opentelemetry.io/otel/codes"
//     "go.opentelemetry.io/otel/trace"
// )
//
// func TracingMiddleware(tracer trace.Tracer) messaging.Middleware {
//     return func(next messaging.HandlerFunc) messaging.HandlerFunc {
//         return func(ctx context.Context, msg *messaging.Message) error {
//             // 从消息元数据中提取 trace context
//             traceID := msg.Metadata.Get(messaging.MetadataKeyTraceID)
//
//             // 创建 span
//             ctx, span := tracer.Start(ctx, "process_message",
//                 trace.WithAttributes(
//                     attribute.String("message.id", msg.ID),
//                     attribute.String("message.topic", msg.Topic),
//                     attribute.String("trace.id", traceID),
//                 ),
//             )
//             defer span.End()
//
//             // 调用处理器
//             err := next(ctx, msg)
//
//             // 记录错误
//             if err != nil {
//                 span.RecordError(err)
//                 span.SetStatus(codes.Error, err.Error())
//             } else {
//                 span.SetStatus(codes.Ok, "")
//             }
//
//             return err
//         }
//     }
// }
