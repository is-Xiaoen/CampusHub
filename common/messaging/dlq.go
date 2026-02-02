package messaging

import "time"

// DLQMessage 死信队列消息
type DLQMessage struct {
	// OriginalMessage 原始消息
	OriginalMessage *Message

	// FailureReason 失败原因
	FailureReason string

	// FailureCount 失败次数
	FailureCount int

	// FirstFailedAt 首次失败时间
	FirstFailedAt time.Time

	// LastFailedAt 最后失败时间
	LastFailedAt time.Time

	// ErrorHistory 错误历史记录
	ErrorHistory []ErrorRecord

	// MovedToDLQAt 移入 DLQ 的时间
	MovedToDLQAt time.Time
}

// ErrorRecord 错误记录
type ErrorRecord struct {
	// Timestamp 错误发生时间
	Timestamp time.Time

	// Error 错误信息
	Error string

	// Attempt 尝试次数
	Attempt int
}

// DLQManager 死信队列管理器接口
type DLQManager interface {
	// Send 发送消息到死信队列
	// 当消息处理失败且超过最大重试次数时调用
	Send(dlqMsg *DLQMessage) error

	// List 列出死信队列中的消息
	// offset: 偏移量，limit: 返回数量
	List(topic string, offset, limit int) ([]*DLQMessage, error)

	// Get 获取指定的死信队列消息
	Get(topic, messageID string) (*DLQMessage, error)

	// Reprocess 重新处理死信队列消息
	// 将消息从 DLQ 移回原始主题进行重新处理
	Reprocess(topic, messageID string) error

	// ReprocessBatch 批量重新处理死信队列消息
	ReprocessBatch(topic string, messageIDs []string) error

	// Delete 删除死信队列消息
	Delete(topic, messageID string) error

	// DeleteBatch 批量删除死信队列消息
	DeleteBatch(topic string, messageIDs []string) error

	// Count 统计死信队列消息数量
	Count(topic string) (int64, error)

	// Purge 清空指定主题的死信队列
	Purge(topic string) error

	// Close 关闭 DLQ 管理器
	Close() error
}

// DLQStats 死信队列统计信息
type DLQStats struct {
	// Topic 主题名称
	Topic string

	// MessageCount 消息数量
	MessageCount int64

	// OldestMessage 最早的消息时间
	OldestMessage time.Time

	// NewestMessage 最新的消息时间
	NewestMessage time.Time
}
