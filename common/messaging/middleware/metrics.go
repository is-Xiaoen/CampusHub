package middleware

import (
	"context"
	"time"

	"CampusHub/common/messaging"
)

// MetricsMiddleware 创建指标收集中间件
// 自动收集消息处理的各项指标
func MetricsMiddleware(collector messaging.MetricsCollector) messaging.Middleware {
	return func(next messaging.HandlerFunc) messaging.HandlerFunc {
		return func(ctx context.Context, msg *messaging.Message) error {
			// 记录消息大小
			messageSize := len(msg.Payload)
			collector.ObserveMessageSize(msg.Topic, messageSize)

			// 记录处理开始
			start := time.Now()

			// 调用处理器
			err := next(ctx, msg)

			// 记录处理时长
			duration := time.Since(start)

			// 记录处理指标
			collector.RecordProcess(msg.Topic, duration, err)

			// 记录消息计数
			status := "success"
			if err != nil {
				status = "error"
			}
			collector.IncrementMessageCount(msg.Topic, status)

			return err
		}
	}
}
