package messaging

import (
	"context"
	"fmt"

	"activity-platform/common/messaging/gozero"
	"activity-platform/common/messaging/middleware"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	wmMiddleware "github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/redis/go-redis/v9"
)

// Client Watermill 消息客户端
type Client struct {
	Publisher  message.Publisher
	Subscriber message.Subscriber
	Router     *message.Router
	config     Config
	redisClient *redis.Client
}

// NewClient 创建新的消息客户端
func NewClient(config Config) (*Client, error) {
	// 创建 Redis 客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Addr,
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
	})

	// 测试 Redis 连接
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// 创建 Watermill logger
	logger := newWatermillLogger(config.ServiceName)

	// 创建 Publisher
	publisher, err := redisstream.NewPublisher(
		redisstream.PublisherConfig{
			Client: redisClient,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher: %w", err)
	}

	// 创建 Subscriber
	subscriber, err := redisstream.NewSubscriber(
		redisstream.SubscriberConfig{
			Client:        redisClient,
			ConsumerGroup: config.ServiceName,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriber: %w", err)
	}

	// 创建 Router（用于中间件）
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create router: %w", err)
	}

	// 应用中间件（按顺序）
	
	// 1. Go-Zero trace_id 传播中间件（最先执行，确保 context 中有 trace_id）
	if config.EnableGoZero {
		router.AddMiddleware(gozero.NewGoZeroMiddleware(config.ServiceName))
	}

	// 2. Prometheus 指标中间件
	if config.EnableMetrics {
		router.AddMiddleware(middleware.NewMetricsMiddleware(config.ServiceName))
	}

	// 3. 重试中间件
	if config.RetryConfig.MaxRetries > 0 {
		retryMiddleware := wmMiddleware.Retry{
			MaxRetries:      config.RetryConfig.MaxRetries,
			InitialInterval: config.RetryConfig.InitialInterval,
			MaxInterval:     config.RetryConfig.MaxInterval,
			Multiplier:      config.RetryConfig.Multiplier,
			Logger:          logger,
		}
		router.AddMiddleware(retryMiddleware.Middleware)
	}

	// 4. DLQ (Poison Queue) 中间件
	// 注意：Watermill 的 PoisonQueue 需要为每个 topic 单独配置
	// 这里我们暂时跳过，在 AddHandler 时可以为特定 topic 添加 DLQ

	return &Client{
		Publisher:   publisher,
		Subscriber:  subscriber,
		Router:      router,
		config:      config,
		redisClient: redisClient,
	}, nil
}

// Close 关闭客户端
func (c *Client) Close() error {
	if err := c.Publisher.Close(); err != nil {
		return fmt.Errorf("failed to close publisher: %w", err)
	}
	if err := c.Subscriber.Close(); err != nil {
		return fmt.Errorf("failed to close subscriber: %w", err)
	}
	if err := c.Router.Close(); err != nil {
		return fmt.Errorf("failed to close router: %w", err)
	}
	if err := c.redisClient.Close(); err != nil {
		return fmt.Errorf("failed to close redis client: %w", err)
	}
	return nil
}

// Publish 发布消息（便捷方法）
func (c *Client) Publish(ctx context.Context, topic string, payload []byte) error {
	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("topic", topic)
	
	// 如果启用了 Go-Zero，注入 trace_id
	if c.config.EnableGoZero {
		gozero.InjectTraceID(ctx, msg)
	}
	
	return c.Publisher.Publish(topic, msg)
}

// Subscribe 订阅消息（便捷方法）
// 注意：这个方法会直接添加 handler 到 Router，需要调用 Router.Run() 来启动
func (c *Client) Subscribe(topic string, handlerName string, handler message.NoPublishHandlerFunc) {
	c.Router.AddNoPublisherHandler(
		handlerName,
		topic,
		c.Subscriber,
		handler,
	)
}

// Run 启动 Router（阻塞）
func (c *Client) Run(ctx context.Context) error {
	return c.Router.Run(ctx)
}

// Running 返回一个 channel，当 Router 运行时关闭
func (c *Client) Running() chan struct{} {
	return c.Router.Running()
}
