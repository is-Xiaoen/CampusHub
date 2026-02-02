package redis

import (
	"context"

	"CampusHub/common/messaging"
	"CampusHub/common/messaging/backend"

	"github.com/redis/go-redis/v9"
)

// Backend Redis Streams 后端实现
type Backend struct {
	client *redis.Client
	config Config
}

// NewBackend 创建 Redis 后端实例
func NewBackend(config Config) (backend.Backend, error) {
	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		MaxRetries:   config.MaxRetries,
		TLSConfig:    config.TLSConfig,
	})

	// 测试连接
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, ErrConnectionFailed(err)
	}

	return &Backend{
		client: client,
		config: config,
	}, nil
}

// CreatePublisher 创建发布者
func (b *Backend) CreatePublisher(config messaging.PublisherConfig) (messaging.Publisher, error) {
	return NewPublisher(b.client, config)
}

// CreateSubscriber 创建订阅者
func (b *Backend) CreateSubscriber(config messaging.SubscriberConfig) (messaging.Subscriber, error) {
	return NewSubscriber(b.client, config)
}

// HealthCheck 健康检查
func (b *Backend) HealthCheck(ctx context.Context) error {
	return b.client.Ping(ctx).Err()
}

// Close 关闭后端连接
func (b *Backend) Close() error {
	return b.client.Close()
}
