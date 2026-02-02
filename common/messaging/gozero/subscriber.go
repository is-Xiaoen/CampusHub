package gozero

import (
	"context"
	"fmt"
	"time"

	"CampusHub/common/messaging"
)

// Subscriber Go-Zero 订阅者适配器
// 包装标准 Subscriber，添加 go-zero 特定功能（追踪、日志等）
type Subscriber struct {
	subscriber  messaging.Subscriber
	logger      Logger
	serviceName string
}

// SubscriberConfig Go-Zero 订阅者配置
type SubscriberConfig struct {
	// Subscriber 底层订阅者
	Subscriber messaging.Subscriber

	// Logger 日志记录器（可选）
	Logger Logger

	// ServiceName 服务名称
	ServiceName string
}

// NewSubscriber 创建 Go-Zero 订阅者适配器
func NewSubscriber(config SubscriberConfig) (*Subscriber, error) {
	if config.Subscriber == nil {
		return nil, fmt.Errorf("订阅者是必填项")
	}

	logger := config.Logger
	if logger == nil {
		logger = NewDefaultLogger()
	}

	return &Subscriber{
		subscriber:  config.Subscriber,
		logger:      logger,
		serviceName: config.ServiceName,
	}, nil
}

// Subscribe 订阅主题
// 自动提取追踪上下文并注入到处理器的 context 中
func (s *Subscriber) Subscribe(ctx context.Context, topic string, consumerGroup string, handler messaging.HandlerFunc) error {
	// 包装处理器，添加 go-zero 特定功能
	wrappedHandler := s.wrapHandler(handler)

	// 记录日志
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"topic":          topic,
		"consumer_group": consumerGroup,
	})

	logger.Infof("Subscribing to topic %s with consumer group %s", topic, consumerGroup)

	// 订阅
	err := s.subscriber.Subscribe(ctx, topic, consumerGroup, wrappedHandler)
	if err != nil {
		logger.Errorf("Failed to subscribe: %v", err)
		return err
	}

	logger.Info("Subscribed successfully")
	return nil
}

// wrapHandler 包装处理器，添加追踪上下文提取和日志记录
func (s *Subscriber) wrapHandler(handler messaging.HandlerFunc) messaging.HandlerFunc {
	return func(ctx context.Context, msg *messaging.Message) error {
		// 从消息元数据中提取追踪上下文
		ctx = ExtractTraceContext(ctx, msg)

		// 注入服务名称
		if s.serviceName != "" {
			ctx = WithServiceName(ctx, s.serviceName)
		}

		// 创建带上下文的日志记录器
		logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
			"message_id": msg.ID,
			"topic":      msg.Topic,
		})

		// 记录开始处理
		logger.Infof("Processing message from topic %s", msg.Topic)

		// 调用处理器
		start := time.Now()
		err := handler(ctx, msg)
		duration := time.Since(start)

		// 记录处理结果
		if err != nil {
			logger.Errorf("Failed to process message: %v (duration: %v)", err, duration)
			return err
		}

		logger.Infof("Message processed successfully (duration: %v)", duration)
		return nil
	}
}

// Close 关闭订阅者
func (s *Subscriber) Close(timeout time.Duration) error {
	s.logger.Infof("Closing subscriber (timeout: %v)", timeout)
	return s.subscriber.Close(timeout)
}
