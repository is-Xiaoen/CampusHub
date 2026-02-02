package backend

import (
	"context"

	"activity-platform/common/messaging"
)

// Backend 定义了消息后端的抽象接口
// 不同的消息代理（Redis Streams, Kafka, etc.）实现此接口
type Backend interface {
	// CreatePublisher 创建一个发布者实例
	// 参数:
	//   - config: 发布者配置
	// 返回:
	//   - Publisher: 发布者实例
	//   - error: 创建失败时返回错误
	CreatePublisher(config messaging.PublisherConfig) (messaging.Publisher, error)

	// CreateSubscriber 创建一个订阅者实例
	// 参数:
	//   - config: 订阅者配置
	// 返回:
	//   - Subscriber: 订阅者实例
	//   - error: 创建失败时返回错误
	CreateSubscriber(config messaging.SubscriberConfig) (messaging.Subscriber, error)

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
