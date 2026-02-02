// Package messaging 定义了 Watermill 消息中间件 SDK 的核心接口
// 这些接口提供了消息发布、订阅和后端抽象的标准契约
package messaging

import (
	"context"
	"time"
)

// ============================================================================
// 核心接口
// ============================================================================

// Publisher 定义了消息发布者的接口
// 发布者负责将消息发送到指定的主题
type Publisher interface {
	// Publish 发布单条消息到指定主题
	// 参数:
	//   - ctx: 上下文，用于超时控制和取消
	//   - msg: 要发布的消息
	// 返回:
	//   - error: 发布失败时返回错误
	Publish(ctx context.Context, msg *Message) error

	// PublishBatch 批量发布消息到指定主题
	// 参数:
	//   - ctx: 上下文
	//   - msgs: 要发布的消息列表
	// 返回:
	//   - error: 任何一条消息发布失败都会返回错误
	PublishBatch(ctx context.Context, msgs []*Message) error

	// Close 关闭发布者，释放资源
	// 返回:
	//   - error: 关闭失败时返回错误
	Close() error
}

// Subscriber 定义了消息订阅者的接口
// 订阅者负责从指定主题接收消息并调用处理器
type Subscriber interface {
	// Subscribe 订阅指定主题的消息
	// 参数:
	//   - ctx: 上下文，用于控制订阅生命周期
	//   - topic: 主题名称
	//   - consumerGroup: 消费者组名称
	//   - handler: 消息处理函数
	// 返回:
	//   - error: 订阅失败时返回错误
	Subscribe(ctx context.Context, topic string, consumerGroup string, handler HandlerFunc) error

	// Close 关闭订阅者，优雅地停止消息处理
	// 参数:
	//   - timeout: 等待处理中消息完成的超时时间
	// 返回:
	//   - error: 关闭失败时返回错误
	Close(timeout time.Duration) error
}

// Backend 定义了消息后端的抽象接口
// 不同的消息代理（Redis Streams, Kafka, etc.）实现此接口
type Backend interface {
	// CreatePublisher 创建一个发布者实例
	// 参数:
	//   - config: 发布者配置
	// 返回:
	//   - Publisher: 发布者实例
	//   - error: 创建失败时返回错误
	CreatePublisher(config PublisherConfig) (Publisher, error)

	// CreateSubscriber 创建一个订阅者实例
	// 参数:
	//   - config: 订阅者配置
	// 返回:
	//   - Subscriber: 订阅者实例
	//   - error: 创建失败时返回错误
	CreateSubscriber(config SubscriberConfig) (Subscriber, error)

	// HealthCheck 检查后端健康状态
	// 参数:
	//   - ctx: 上下文
	// 返回:
	//   - error: 后端不健康时返回错误
	HealthCheck(ctx context.Context) error

	// Close 关闭后端连接
	// 返回:
	//   - error: 关闭失败时返回错误
	Close() error
}

// ============================================================================
// 处理器和中间件
// ============================================================================

// HandlerFunc 定义了消息处理函数的签名
// 参数:
//   - ctx: 上下文，包含追踪信息等
//   - msg: 接收到的消息
// 返回:
//   - error: 处理失败时返回错误，将触发重试机制
type HandlerFunc func(ctx context.Context, msg *Message) error

// Middleware 定义了中间件的签名
// 中间件可以在消息处理前后执行额外逻辑（如日志、追踪、重试）
// 参数:
//   - next: 下一个处理器
// 返回:
//   - HandlerFunc: 包装后的处理器
type Middleware func(next HandlerFunc) HandlerFunc

// ============================================================================
// 序列化器接口
// ============================================================================

// Serializer 定义了消息序列化器的接口
// 用于将消息负载序列化为字节流，或从字节流反序列化
type Serializer interface {
	// Marshal 将对象序列化为字节流
	// 参数:
	//   - v: 要序列化的对象
	// 返回:
	//   - []byte: 序列化后的字节流
	//   - error: 序列化失败时返回错误
	Marshal(v interface{}) ([]byte, error)

	// Unmarshal 将字节流反序列化为对象
	// 参数:
	//   - data: 字节流
	//   - v: 目标对象指针
	// 返回:
	//   - error: 反序列化失败时返回错误
	Unmarshal(data []byte, v interface{}) error

	// ContentType 返回序列化器的内容类型
	// 返回:
	//   - string: 内容类型，如 "application/json"
	ContentType() string
}

// ============================================================================
// DLQ 管理接口
// ============================================================================

// DLQManager 定义了死信队列管理的接口
// 用于查询、检查和重新处理死信队列中的消息
type DLQManager interface {
	// List 列出死信队列中的消息
	// 参数:
	//   - ctx: 上下文
	//   - topic: 原始主题名称
	//   - limit: 返回的最大消息数量
	//   - offset: 偏移量
	// 返回:
	//   - []*DLQMessage: 死信队列消息列表
	//   - error: 查询失败时返回错误
	List(ctx context.Context, topic string, limit, offset int) ([]*DLQMessage, error)

	// Get 获取指定 ID 的死信队列消息
	// 参数:
	//   - ctx: 上下文
	//   - topic: 原始主题名称
	//   - messageID: 消息 ID
	// 返回:
	//   - *DLQMessage: 死信队列消息
	//   - error: 查询失败时返回错误
	Get(ctx context.Context, topic string, messageID string) (*DLQMessage, error)

	// Reprocess 将死信队列消息重新投递到原始主题
	// 参数:
	//   - ctx: 上下文
	//   - topic: 原始主题名称
	//   - messageID: 消息 ID
	// 返回:
	//   - error: 重新投递失败时返回错误
	Reprocess(ctx context.Context, topic string, messageID string) error

	// ReprocessBatch 批量重新投递死信队列消息
	// 参数:
	//   - ctx: 上下文
	//   - topic: 原始主题名称
	//   - messageIDs: 消息 ID 列表
	// 返回:
	//   - error: 任何一条消息重新投递失败都会返回错误
	ReprocessBatch(ctx context.Context, topic string, messageIDs []string) error

	// Delete 删除死信队列消息
	// 参数:
	//   - ctx: 上下文
	//   - topic: 原始主题名称
	//   - messageID: 消息 ID
	// 返回:
	//   - error: 删除失败时返回错误
	Delete(ctx context.Context, topic string, messageID string) error

	// DeleteBatch 批量删除死信队列消息
	// 参数:
	//   - ctx: 上下文
	//   - topic: 原始主题名称
	//   - messageIDs: 消息 ID 列表
	// 返回:
	//   - error: 任何一条消息删除失败都会返回错误
	DeleteBatch(ctx context.Context, topic string, messageIDs []string) error

	// Count 统计死信队列中的消息数量
	// 参数:
	//   - ctx: 上下文
	//   - topic: 原始主题名称
	// 返回:
	//   - int64: 消息数量
	//   - error: 统计失败时返回错误
	Count(ctx context.Context, topic string) (int64, error)
}

// ============================================================================
// 指标接口
// ============================================================================

// MetricsCollector 定义了指标收集器的接口
// 用于收集和暴露 SDK 的运行指标
type MetricsCollector interface {
	// RecordPublish 记录消息发布事件
	// 参数:
	//   - topic: 主题名称
	//   - success: 是否成功
	//   - duration: 发布耗时
	RecordPublish(topic string, success bool, duration time.Duration)

	// RecordConsume 记录消息消费事件
	// 参数:
	//   - topic: 主题名称
	//   - consumerGroup: 消费者组名称
	//   - success: 是否成功
	//   - duration: 消费耗时
	RecordConsume(topic string, consumerGroup string, success bool, duration time.Duration)

	// RecordRetry 记录消息重试事件
	// 参数:
	//   - topic: 主题名称
	//   - consumerGroup: 消费者组名称
	//   - attempt: 重试次数
	RecordRetry(topic string, consumerGroup string, attempt int)

	// RecordDLQ 记录消息进入死信队列事件
	// 参数:
	//   - topic: 主题名称
	//   - consumerGroup: 消费者组名称
	RecordDLQ(topic string, consumerGroup string)

	// UpdateConsumerLag 更新消费者 lag
	// 参数:
	//   - topic: 主题名称
	//   - consumerGroup: 消费者组名称
	//   - lag: lag 值（待处理消息数）
	UpdateConsumerLag(topic string, consumerGroup string, lag int64)

	// UpdateActiveConsumers 更新活跃消费者数量
	// 参数:
	//   - topic: 主题名称
	//   - consumerGroup: 消费者组名称
	//   - count: 消费者数量
	UpdateActiveConsumers(topic string, consumerGroup string, count int)
}

// ============================================================================
// 日志接口
// ============================================================================

// Logger 定义了日志记录器的接口
// 用于记录 SDK 的运行日志
type Logger interface {
	// Debug 记录调试级别日志
	Debug(msg string, fields ...Field)

	// Info 记录信息级别日志
	Info(msg string, fields ...Field)

	// Warn 记录警告级别日志
	Warn(msg string, fields ...Field)

	// Error 记录错误级别日志
	Error(msg string, fields ...Field)

	// With 创建一个带有额外字段的子日志记录器
	With(fields ...Field) Logger
}

// Field 定义了日志字段
type Field struct {
	Key   string
	Value interface{}
}

// ============================================================================
// 工厂接口
// ============================================================================

// Factory 定义了 SDK 工厂的接口
// 用于创建发布者、订阅者和其他组件
type Factory interface {
	// NewPublisher 创建一个新的发布者
	// 参数:
	//   - config: 发布者配置
	// 返回:
	//   - Publisher: 发布者实例
	//   - error: 创建失败时返回错误
	NewPublisher(config PublisherConfig) (Publisher, error)

	// NewSubscriber 创建一个新的订阅者
	// 参数:
	//   - config: 订阅者配置
	// 返回:
	//   - Subscriber: 订阅者实例
	//   - error: 创建失败时返回错误
	NewSubscriber(config SubscriberConfig) (Subscriber, error)

	// NewDLQManager 创建一个新的 DLQ 管理器
	// 参数:
	//   - config: 后端配置
	// 返回:
	//   - DLQManager: DLQ 管理器实例
	//   - error: 创建失败时返回错误
	NewDLQManager(config interface{}) (DLQManager, error)
}

// ============================================================================
// 接口使用示例
// ============================================================================

/*
// 示例 1: 创建发布者并发布消息
func ExamplePublisher() {
	// 创建 Redis 后端
	backend, _ := redis.NewBackend(redis.Config{
		Addr: "localhost:6379",
	})

	// 创建发布者
	publisher, _ := backend.CreatePublisher(PublisherConfig{
		Serializer: &JSONSerializer{},
	})
	defer publisher.Close()

	// 发布消息
	msg := &Message{
		Topic:   "user.registered",
		Payload: []byte(`{"user_id": "123", "email": "user@example.com"}`),
		Metadata: Metadata{
			"event_type": "user.registered",
			"trace_id":   "abc123",
		},
	}

	ctx := context.Background()
	if err := publisher.Publish(ctx, msg); err != nil {
		log.Fatal(err)
	}
}

// 示例 2: 创建订阅者并处理消息
func ExampleSubscriber() {
	// 创建 Redis 后端
	backend, _ := redis.NewBackend(redis.Config{
		Addr: "localhost:6379",
	})

	// 创建订阅者
	subscriber, _ := backend.CreateSubscriber(SubscriberConfig{
		Serializer: &JSONSerializer{},
		RetryPolicy: RetryPolicy{
			Enabled:         true,
			InitialInterval: time.Second,
			MaxAttempts:     3,
		},
	})
	defer subscriber.Close(30 * time.Second)

	// 定义处理器
	handler := func(ctx context.Context, msg *Message) error {
		log.Printf("Received message: %s", string(msg.Payload))
		// 处理消息...
		return nil
	}

	// 订阅主题
	ctx := context.Background()
	if err := subscriber.Subscribe(ctx, "user.registered", "email-sender", handler); err != nil {
		log.Fatal(err)
	}

	// 等待信号...
}

// 示例 3: 使用中间件
func ExampleMiddleware() {
	// 日志中间件
	loggingMiddleware := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *Message) error {
			log.Printf("Processing message: %s", msg.ID)
			err := next(ctx, msg)
			if err != nil {
				log.Printf("Error processing message: %v", err)
			}
			return err
		}
	}

	// 追踪中间件
	tracingMiddleware := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *Message) error {
			ctx, span := tracer.Start(ctx, "messaging.consume")
			defer span.End()
			return next(ctx, msg)
		}
	}

	// 组合中间件
	handler := func(ctx context.Context, msg *Message) error {
		// 实际处理逻辑
		return nil
	}

	wrappedHandler := loggingMiddleware(tracingMiddleware(handler))

	// 使用包装后的处理器...
}
*/
