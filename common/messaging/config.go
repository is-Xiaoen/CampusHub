package messaging

import "time"

// PublisherConfig 发布者配置
type PublisherConfig struct {
	// Backend 后端类型（redis, kafka, mock）
	Backend string

	// BackendConfig 后端特定配置
	BackendConfig interface{}

	// Middlewares 中间件链
	Middlewares []Middleware

	// PublishTimeout 发布超时时间
	PublishTimeout time.Duration

	// EnableBatch 是否启用批量发布
	EnableBatch bool

	// BatchSize 批量发布大小
	BatchSize int

	// BatchTimeout 批量发布超时
	BatchTimeout time.Duration
}

// SubscriberConfig 订阅者配置
type SubscriberConfig struct {
	// Backend 后端类型
	Backend string

	// BackendConfig 后端特定配置
	BackendConfig interface{}

	// Middlewares 中间件链
	Middlewares []Middleware

	// ConsumerGroup 消费者组配置
	ConsumerGroup ConsumerGroupConfig

	// RetryPolicy 重试策略
	RetryPolicy RetryPolicy

	// DLQConfig DLQ 配置
	DLQConfig DLQConfig

	// ShutdownTimeout 优雅关闭超时
	ShutdownTimeout time.Duration
}

// ConsumerGroupConfig 消费者组配置
type ConsumerGroupConfig struct {
	// BatchSize 每次读取的消息数量（批量大小）
	BatchSize int // 默认 10

	// BlockTime 阻塞等待时间（毫秒）
	BlockTime time.Duration // 默认 1000ms

	// ProcessTimeout 消息处理超时时间
	ProcessTimeout time.Duration // 默认 30s

	// Concurrency 并发处理数量
	Concurrency int // 默认 1

	// AutoAck 是否自动 ACK
	AutoAck bool // 默认 false
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	// Enabled 是否启用重试
	Enabled bool // 默认 true

	// InitialInterval 初始延迟
	InitialInterval time.Duration // 默认 1s

	// MaxInterval 最大延迟
	MaxInterval time.Duration // 默认 60s

	// Multiplier 退避倍数
	Multiplier float64 // 默认 2.0

	// MaxAttempts 最大重试次数
	MaxAttempts int // 默认 3

	// RetryableErrors 可重试的错误类型
	RetryableErrors []error
}

// DLQConfig 死信队列配置
type DLQConfig struct {
	// Enabled 是否启用 DLQ
	Enabled bool // 默认 true

	// TopicSuffix DLQ 主题后缀
	TopicSuffix string // 默认 ".dlq"

	// RetentionPeriod DLQ 消息保留期
	RetentionPeriod time.Duration // 默认 30 天
}

// DefaultPublisherConfig 返回默认发布者配置
func DefaultPublisherConfig() PublisherConfig {
	return PublisherConfig{
		Backend:        "redis",
		PublishTimeout: 5 * time.Second,
		EnableBatch:    false,
		BatchSize:      100,
		BatchTimeout:   100 * time.Millisecond,
	}
}

// DefaultSubscriberConfig 返回默认订阅者配置
func DefaultSubscriberConfig() SubscriberConfig {
	return SubscriberConfig{
		Backend: "redis",
		ConsumerGroup: ConsumerGroupConfig{
			BatchSize:      10,
			BlockTime:      1 * time.Second,
			ProcessTimeout: 30 * time.Second,
			Concurrency:    1,
			AutoAck:        false,
		},
		RetryPolicy: RetryPolicy{
			Enabled:         true,
			InitialInterval: 1 * time.Second,
			MaxInterval:     60 * time.Second,
			Multiplier:      2.0,
			MaxAttempts:     3,
		},
		DLQConfig: DLQConfig{
			Enabled:         true,
			TopicSuffix:     ".dlq",
			RetentionPeriod: 30 * 24 * time.Hour, // 30 天
		},
		ShutdownTimeout: 30 * time.Second,
	}
}
