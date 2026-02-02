package messaging

import (
	"context"
	"time"
)

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

// HandlerFunc 定义了消息处理函数的签名
// 参数:
//   - ctx: 上下文，包含追踪信息等
//   - msg: 接收到的消息
// 返回:
//   - error: 处理失败时返回错误，将触发重试机制
type HandlerFunc func(ctx context.Context, msg *Message) error
