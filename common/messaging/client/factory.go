package client

import (
	"context"
	"fmt"
	"time"

	"activity-platform/common/messaging"
	"activity-platform/common/messaging/backend/redis"
	"activity-platform/common/messaging/gozero"
	"activity-platform/common/messaging/metrics"
	"activity-platform/common/messaging/middleware"

	goredis "github.com/redis/go-redis/v9"
)

// MessagingConfig 消息中间件统一配置
type MessagingConfig struct {
	// Redis 配置
	Redis RedisConfig `json:",optional"`

	// 服务名称（用于 go-zero 集成）
	ServiceName string `json:",optional"`

	// 发布者配置
	Publisher messaging.PublisherConfig `json:",optional"`

	// 订阅者配置
	Subscriber messaging.SubscriberConfig `json:",optional"`

	// 是否启用 Prometheus 指标
	EnableMetrics bool `json:",optional"`

	// Prometheus 命名空间
	MetricsNamespace string `json:",optional,default=campushub"`
}

// RedisConfig Redis 连接配置
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
type MessagingClient struct {
	redisClient *goredis.Client
	Publisher   messaging.Publisher
	Subscriber  messaging.Subscriber
	config      MessagingConfig
}

// NewMessagingClient 创建消息中间件客户端（推荐使用）
//
// 这是一个工厂函数，简化了消息中间件的初始化过程。
// 它会自动：
// 1. 创建 Redis 客户端
// 2. 创建 Publisher 和 Subscriber
// 3. 集成 go-zero（如果配置了 ServiceName）
// 4. 添加 Prometheus 指标（如果启用）
//
// 使用示例：
//
//	config := client.MessagingConfig{
//	    Redis: client.RedisConfig{
//	        Addr: "localhost:6379",
//	    },
//	    ServiceName: "user-service",
//	    EnableMetrics: true,
//	}
//	client, err := client.NewMessagingClient(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
func NewMessagingClient(config MessagingConfig) (*MessagingClient, error) {
	// 1. 创建 Redis 客户端
	redisClient := goredis.NewClient(&goredis.Options{
		Addr:         config.Redis.Addr,
		Password:     config.Redis.Password,
		DB:           config.Redis.DB,
		PoolSize:     config.Redis.PoolSize,
		MinIdleConns: config.Redis.MinIdleConns,
		DialTimeout:  config.Redis.DialTimeout,
		ReadTimeout:  config.Redis.ReadTimeout,
		WriteTimeout: config.Redis.WriteTimeout,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("连接Redis失败: %w", err)
	}

	// 2. 设置默认配置
	if config.Publisher.PublishTimeout == 0 {
		config.Publisher = messaging.DefaultPublisherConfig()
	}
	if config.Subscriber.RetryPolicy.MaxAttempts == 0 {
		config.Subscriber = messaging.DefaultSubscriberConfig()
	}

	// 3. 添加 Prometheus 指标中间件（如果启用）
	if config.EnableMetrics {
		if config.MetricsNamespace == "" {
			config.MetricsNamespace = "campushub"
		}
		metricsCollector := metrics.NewPrometheusCollector(config.MetricsNamespace)
		config.Subscriber.Middlewares = append(
			[]messaging.Middleware{middleware.MetricsMiddleware(metricsCollector)},
			config.Subscriber.Middlewares...,
		)
	}

	// 4. 创建基础 Publisher 和 Subscriber
	basePublisher, err := redis.NewPublisher(redisClient, config.Publisher)
	if err != nil {
		redisClient.Close()
		return nil, fmt.Errorf("创建发布者失败: %w", err)
	}

	baseSubscriber, err := redis.NewSubscriber(redisClient, config.Subscriber)
	if err != nil {
		redisClient.Close()
		return nil, fmt.Errorf("创建订阅者失败: %w", err)
	}

	// 5. 如果配置了 ServiceName，则使用 go-zero 适配器
	var publisher messaging.Publisher = basePublisher
	var subscriber messaging.Subscriber = baseSubscriber

	if config.ServiceName != "" {
		gzPublisher, err := gozero.NewPublisher(gozero.PublisherConfig{
			Publisher:   basePublisher,
			ServiceName: config.ServiceName,
		})
		if err != nil {
			redisClient.Close()
			return nil, fmt.Errorf("创建go-zero发布者失败: %w", err)
		}
		publisher = gzPublisher

		gzSubscriber, err := gozero.NewSubscriber(gozero.SubscriberConfig{
			Subscriber:  baseSubscriber,
			ServiceName: config.ServiceName,
		})
		if err != nil {
			redisClient.Close()
			return nil, fmt.Errorf("创建go-zero订阅者失败: %w", err)
		}
		subscriber = gzSubscriber
	}

	return &MessagingClient{
		redisClient: redisClient,
		Publisher:   publisher,
		Subscriber:  subscriber,
		config:      config,
	}, nil
}

// Close 关闭消息中间件客户端
func (c *MessagingClient) Close() error {
	// 关闭订阅者
	if err := c.Subscriber.Close(10 * time.Second); err != nil {
		return fmt.Errorf("关闭订阅者失败: %w", err)
	}

	// 关闭发布者
	if err := c.Publisher.Close(); err != nil {
		return fmt.Errorf("关闭发布者失败: %w", err)
	}

	// 关闭 Redis 客户端
	if err := c.redisClient.Close(); err != nil {
		return fmt.Errorf("关闭redis客户端失败: %w", err)
	}

	return nil
}

// HealthCheck 健康检查
func (c *MessagingClient) HealthCheck(ctx context.Context) error {
	return c.redisClient.Ping(ctx).Err()
}

// GetRedisClient 获取底层 Redis 客户端（用于高级用法，如 DLQ 管理）
func (c *MessagingClient) GetRedisClient() *goredis.Client {
	return c.redisClient
}
