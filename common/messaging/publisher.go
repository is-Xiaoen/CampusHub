package messaging

import "context"

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
