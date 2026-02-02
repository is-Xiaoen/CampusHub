package messaging

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// MessagingConfig 消息中间件统一配置
// 已废弃：请使用 client.MessagingConfig
// 为了向后兼容保留此类型定义
type MessagingConfig struct {
	// Redis 配置
	Redis RedisConfig `json:",optional"`

	// 服务名称（用于 go-zero 集成）
	ServiceName string `json:",optional"`

	// 发布者配置
	Publisher PublisherConfig `json:",optional"`

	// 订阅者配置
	Subscriber SubscriberConfig `json:",optional"`

	// 是否启用 Prometheus 指标
	EnableMetrics bool `json:",optional"`

	// Prometheus 命名空间
	MetricsNamespace string `json:",optional,default=campushub"`
}

// RedisConfig Redis 连接配置
// 已废弃：请使用 client.RedisConfig
// 为了向后兼容保留此类型定义
type RedisConfig struct {
	Addr         string        `json:",default=localhost:6379"`
	Password     string        `json:",optional"`
	DB           int           `json:",optional,default=0"`
	PoolSize     int           `json:",optional,default=10"`
	MinIdleConns int           `json:",optional,default=5"`
	DialTimeout  time.Duration `json:",optional,default=5s"`
	ReadTimeout  time.Duration `json:",optional,default=3s"`
	WriteTimeout time.Duration `json:",optional,default=3s"`
}

// MessagingClient 消息中间件客户端（封装了 Publisher 和 Subscriber）
// 已废弃：请使用 client.MessagingClient
// 为了向后兼容保留此类型定义
type MessagingClient struct {
	redisClient *goredis.Client
	Publisher   Publisher
	Subscriber  Subscriber
	config      MessagingConfig
}

// NewMessagingClient 创建消息中间件客户端
// 已废弃：请使用 client.NewMessagingClient 以避免循环导入
// 此函数保留用于向后兼容，但会在未来版本中移除
//
// 迁移指南：
//   import "CampusHub/common/messaging/client"
//   c, err := client.NewMessagingClient(client.MessagingConfig{...})
func NewMessagingClient(config MessagingConfig) (*MessagingClient, error) {
	// 为了避免循环导入，此函数已被移除
	// 请使用 client.NewMessagingClient
	panic("messaging.NewMessagingClient 已废弃，请使用 client.NewMessagingClient")
}

// Close 关闭消息中间件客户端
func (c *MessagingClient) Close() error {
	// 关闭订阅者
	if err := c.Subscriber.Close(10 * time.Second); err != nil {
		return err
	}

	// 关闭发布者
	if err := c.Publisher.Close(); err != nil {
		return err
	}

	// 关闭 Redis 客户端
	if c.redisClient != nil {
		if err := c.redisClient.Close(); err != nil {
			return err
		}
	}

	return nil
}

// HealthCheck 健康检查
func (c *MessagingClient) HealthCheck(ctx context.Context) error {
	if c.redisClient != nil {
		return c.redisClient.Ping(ctx).Err()
	}
	return nil
}

// GetRedisClient 获取底层 Redis 客户端（用于高级用法，如 DLQ 管理）
func (c *MessagingClient) GetRedisClient() *goredis.Client {
	return c.redisClient
}
