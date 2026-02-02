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
	Enabled     bool
	TopicSuffix string // 默认 ".dlq"
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
			Enabled:     true,
			TopicSuffix: ".dlq",
		},
	}
}
