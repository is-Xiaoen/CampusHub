package messaging

import "time"

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	// RecordPublish 记录消息发布指标
	RecordPublish(topic string, duration time.Duration, err error)

	// RecordPublishBatch 记录批量发布指标
	RecordPublishBatch(topic string, count int, duration time.Duration, err error)

	// RecordProcess 记录消息处理指标
	RecordProcess(topic string, duration time.Duration, err error)

	// RecordRetry 记录重试指标
	RecordRetry(topic string, attempt int, success bool)

	// RecordDLQ 记录死信队列指标
	RecordDLQ(topic string, reason string)

	// IncrementMessageCount 增加消息计数
	IncrementMessageCount(topic string, status string)

	// ObserveMessageSize 观察消息大小
	ObserveMessageSize(topic string, size int)

	// SetDLQSize 设置 DLQ 大小
	SetDLQSize(topic string, size int64)
}

// NoOpMetricsCollector 空操作指标收集器（用于禁用指标收集）
type NoOpMetricsCollector struct{}

// RecordPublish 空实现
func (n *NoOpMetricsCollector) RecordPublish(topic string, duration time.Duration, err error) {}

// RecordPublishBatch 空实现
func (n *NoOpMetricsCollector) RecordPublishBatch(topic string, count int, duration time.Duration, err error) {
}

// RecordProcess 空实现
func (n *NoOpMetricsCollector) RecordProcess(topic string, duration time.Duration, err error) {}

// RecordRetry 空实现
func (n *NoOpMetricsCollector) RecordRetry(topic string, attempt int, success bool) {}

// RecordDLQ 空实现
func (n *NoOpMetricsCollector) RecordDLQ(topic string, reason string) {}

// IncrementMessageCount 空实现
func (n *NoOpMetricsCollector) IncrementMessageCount(topic string, status string) {}

// ObserveMessageSize 空实现
func (n *NoOpMetricsCollector) ObserveMessageSize(topic string, size int) {}

// SetDLQSize 空实现
func (n *NoOpMetricsCollector) SetDLQSize(topic string, size int64) {}
