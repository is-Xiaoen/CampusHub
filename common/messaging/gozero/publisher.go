package gozero

import (
	"context"
	"fmt"
	"time"

	"activity-platform/common/messaging"
)

// Publisher Go-Zero 发布者适配器
// 包装标准 Publisher，添加 go-zero 特定功能（追踪、日志等）
type Publisher struct {
	publisher   messaging.Publisher
	logger      Logger
	serviceName string
}

// PublisherConfig Go-Zero 发布者配置
type PublisherConfig struct {
	// Publisher 底层发布者
	Publisher messaging.Publisher

	// Logger 日志记录器（可选）
	Logger Logger

	// ServiceName 服务名称
	ServiceName string
}

// NewPublisher 创建 Go-Zero 发布者适配器
func NewPublisher(config PublisherConfig) (*Publisher, error) {
	if config.Publisher == nil {
		return nil, fmt.Errorf("发布者是必填项")
	}

	logger := config.Logger
	if logger == nil {
		logger = NewDefaultLogger()
	}

	return &Publisher{
		publisher:   config.Publisher,
		logger:      logger,
		serviceName: config.ServiceName,
	}, nil
}

// Publish 发布消息
// 自动注入追踪上下文和服务信息
func (p *Publisher) Publish(ctx context.Context, msg *messaging.Message) error {
	// 注入服务名称
	if p.serviceName != "" {
		ctx = WithServiceName(ctx, p.serviceName)
	}

	// 注入追踪上下文到消息元数据
	InjectTraceContext(ctx, msg)

	// 记录日志
	logger := p.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"message_id": msg.ID,
		"topic":      msg.Topic,
	})

	logger.Infof("Publishing message to topic %s", msg.Topic)

	// 发布消息
	start := time.Now()
	err := p.publisher.Publish(ctx, msg)
	duration := time.Since(start)

	if err != nil {
		logger.Errorf("Failed to publish message: %v (duration: %v)", err, duration)
		return err
	}

	logger.Infof("Message published successfully (duration: %v)", duration)
	return nil
}

// PublishBatch 批量发布消息
func (p *Publisher) PublishBatch(ctx context.Context, msgs []*messaging.Message) error {
	// 注入服务名称
	if p.serviceName != "" {
		ctx = WithServiceName(ctx, p.serviceName)
	}

	// 为每条消息注入追踪上下文
	for _, msg := range msgs {
		InjectTraceContext(ctx, msg)
	}

	// 记录日志
	logger := p.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"message_count": len(msgs),
	})

	logger.Infof("Publishing %d messages in batch", len(msgs))

	// 批量发布
	start := time.Now()
	err := p.publisher.PublishBatch(ctx, msgs)
	duration := time.Since(start)

	if err != nil {
		logger.Errorf("Failed to publish batch: %v (duration: %v)", err, duration)
		return err
	}

	logger.Infof("Batch published successfully (duration: %v)", duration)
	return nil
}

// Close 关闭发布者
func (p *Publisher) Close() error {
	p.logger.Info("Closing publisher")
	return p.publisher.Close()
}
