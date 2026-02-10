package messaging

import (
	"time"
)

// Config 消息中间件配置
type Config struct {
	// Redis 配置
	Redis RedisConfig

	// 服务配置
	ServiceName string

	// 中间件配置
	EnableMetrics bool
	EnableGoZero  bool // 启用 Go-Zero trace_id 传播

	// 重试配置
	RetryConfig RetryConfig

	// DLQ 配置
	DLQConfig DLQConfig

	// Redis Streams 订阅者配置
	SubscriberConfig SubscriberConfig
}

// RedisConfig Redis 连接配置
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries      int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64 // 退避倍数，默认 2.0
}

// DLQConfig 死信队列配置
type DLQConfig struct {
	Enabled          bool
	TopicSuffix      string // 默认 ".dlq"
	OnlyNonRetryable bool   // 如果为 true，只有不可重试错误进入 DLQ
}

// SubscriberConfig Redis Streams 订阅者配置
type SubscriberConfig struct {
	// ClaimInterval 声明待处理消息的间隔时间
	// 设置为 0 可以禁用自动声明（避免 XPENDING 调用）
	ClaimInterval time.Duration

	// NackResendInterval 重新发送 NACK 消息的间隔时间
	NackResendInterval time.Duration

	// MaxIdleTime 消息被认为是空闲的最大时间
	MaxIdleTime time.Duration
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Redis: RedisConfig{
			Addr: "localhost:6379",
			DB:   0,
		},
		ServiceName:   "default-service",
		EnableMetrics: true,
		EnableGoZero:  true,
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     10 * time.Second,
			Multiplier:      2.0,
		},
		DLQConfig: DLQConfig{
			Enabled:          true,
			TopicSuffix:      ".dlq",
			OnlyNonRetryable: false, // 默认：所有错误在重试后进入 DLQ
		},
		SubscriberConfig: SubscriberConfig{
			ClaimInterval:      time.Second * 30, // 默认 30 秒
			NackResendInterval: time.Second * 10, // 默认 10 秒
			MaxIdleTime:        time.Minute * 5,  // 默认 5 分钟
		},
	}
}
