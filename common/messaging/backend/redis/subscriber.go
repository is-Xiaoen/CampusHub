package redis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"activity-platform/common/messaging"
	"activity-platform/common/messaging/middleware"

	"github.com/redis/go-redis/v9"
)

// Subscriber Redis Streams 订阅者实现
type Subscriber struct {
	client        *redis.Client
	config        messaging.SubscriberConfig
	subscriptions map[string]*subscription
	mu            sync.RWMutex
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
}

// subscription 订阅信息
type subscription struct {
	topic         string
	consumerGroup string
	handler       messaging.HandlerFunc
	cancel        context.CancelFunc
}

// NewSubscriber 创建 Redis 订阅者
func NewSubscriber(client *redis.Client, config messaging.SubscriberConfig) (*Subscriber, error) {
	// 设置默认值
	if config.ConsumerGroup.BatchSize == 0 {
		config.ConsumerGroup.BatchSize = 10
	}
	if config.ConsumerGroup.BlockTime == 0 {
		config.ConsumerGroup.BlockTime = 1 * time.Second
	}
	if config.ConsumerGroup.ProcessTimeout == 0 {
		config.ConsumerGroup.ProcessTimeout = 30 * time.Second
	}
	if config.ConsumerGroup.Concurrency == 0 {
		config.ConsumerGroup.Concurrency = 1
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Subscriber{
		client:        client,
		config:        config,
		subscriptions: make(map[string]*subscription),
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

// Subscribe 订阅主题
func (s *Subscriber) Subscribe(ctx context.Context, topic string, consumerGroup string, handler messaging.HandlerFunc) error {
	// 验证主题名称
	if err := messaging.ValidateTopicName(topic); err != nil {
		return fmt.Errorf("无效主题: %w", err)
	}

	// 检查是否已订阅
	s.mu.Lock()
	key := fmt.Sprintf("%s:%s", topic, consumerGroup)
	if _, exists := s.subscriptions[key]; exists {
		s.mu.Unlock()
		return fmt.Errorf("已订阅主题 %s 的消费者组 %s", topic, consumerGroup)
	}

	// 创建消费者组（如果不存在）
	if err := s.createConsumerGroup(ctx, topic, consumerGroup); err != nil {
		s.mu.Unlock()
		return err
	}

	// 应用中间件链
	wrappedHandler := s.applyMiddlewares(handler)

	// 创建订阅
	subCtx, subCancel := context.WithCancel(s.ctx)
	sub := &subscription{
		topic:         topic,
		consumerGroup: consumerGroup,
		handler:       wrappedHandler,
		cancel:        subCancel,
	}
	s.subscriptions[key] = sub
	s.mu.Unlock()

	// 启动消费者
	s.wg.Add(1)
	go s.consume(subCtx, sub)

	return nil
}

// applyMiddlewares 应用中间件链
func (s *Subscriber) applyMiddlewares(handler messaging.HandlerFunc) messaging.HandlerFunc {
	// 从后向前应用中间件，确保第一个中间件最外层
	wrappedHandler := handler

	// 首先应用重试中间件（如果启用）
	if s.config.RetryPolicy.Enabled {
		retryMiddleware := middleware.RetryMiddleware(s.config.RetryPolicy)
		wrappedHandler = retryMiddleware(wrappedHandler)
	}

	// 然后应用 DLQ 中间件（如果启用）
	// DLQ 中间件应该在重试中间件之外，以捕获超过最大重试次数的消息
	if s.config.DLQConfig.Enabled {
		dlqManager := NewDLQManager(s.client, s.config.DLQConfig)
		dlqMiddleware := middleware.DLQMiddleware(dlqManager, s.config.DLQConfig)
		wrappedHandler = dlqMiddleware(wrappedHandler)
	}

	// 最后应用配置的中间件链（从后向前）
	for i := len(s.config.Middlewares) - 1; i >= 0; i-- {
		wrappedHandler = s.config.Middlewares[i](wrappedHandler)
	}

	return wrappedHandler
}

// createConsumerGroup 创建消费者组
func (s *Subscriber) createConsumerGroup(ctx context.Context, topic string, consumerGroup string) error {
	// 尝试创建消费者组，从 "0" 开始（读取所有消息）
	err := s.client.XGroupCreateMkStream(ctx, topic, consumerGroup, "0").Err()
	if err != nil {
		// 如果消费者组已存在，忽略错误
		if err.Error() == "BUSYGROUP Consumer Group name already exists" {
			return nil
		}
		return fmt.Errorf("创建消费者组失败: %w", err)
	}
	return nil
}

// consume 消费消息
func (s *Subscriber) consume(ctx context.Context, sub *subscription) {
	defer s.wg.Done()

	consumerName := fmt.Sprintf("consumer-%d", time.Now().UnixNano())

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 使用 XREADGROUP 读取消息
			streams, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    sub.consumerGroup,
				Consumer: consumerName,
				Streams:  []string{sub.topic, ">"},
				Count:    int64(s.config.ConsumerGroup.BatchSize),
				Block:    s.config.ConsumerGroup.BlockTime,
			}).Result()

			if err != nil {
				if err == redis.Nil {
					// 没有新消息，继续等待
					continue
				}
				if ctx.Err() != nil {
					// 上下文已取消
					return
				}
				// 其他错误，短暂休眠后重试
				time.Sleep(time.Second)
				continue
			}

			// 处理消息
			for _, stream := range streams {
				for _, message := range stream.Messages {
					s.processMessage(ctx, sub, message)
				}
			}
		}
	}
}

// processMessage 处理单条消息
func (s *Subscriber) processMessage(ctx context.Context, sub *subscription, xmsg redis.XMessage) {
	// 解析消息
	msg, err := s.parseMessage(xmsg, sub.topic)
	if err != nil {
		// 解析失败，ACK 消息以避免重复处理
		_ = s.client.XAck(ctx, sub.topic, sub.consumerGroup, xmsg.ID)
		return
	}

	// 设置 ACK/NACK 函数
	msg.Ack = func() error {
		return s.client.XAck(ctx, sub.topic, sub.consumerGroup, xmsg.ID).Err()
	}
	msg.Nack = func() error {
		// NACK 不做任何操作，消息会保留在 pending 列表中
		return nil
	}

	// 设置接收时间
	msg.ReceivedAt = time.Now()

	// 调用处理器
	processCtx := ctx
	if s.config.ConsumerGroup.ProcessTimeout > 0 {
		var cancel context.CancelFunc
		processCtx, cancel = context.WithTimeout(ctx, s.config.ConsumerGroup.ProcessTimeout)
		defer cancel()
	}

	err = sub.handler(processCtx, msg)

	// 根据处理结果 ACK 或 NACK
	if err == nil {
		// 处理成功，ACK 消息
		if !s.config.ConsumerGroup.AutoAck {
			_ = msg.Ack()
		}
	} else {
		// 处理失败，NACK 消息（保留在 pending 列表）
		_ = msg.Nack()
	}
}

// parseMessage 解析 Redis Stream 消息
func (s *Subscriber) parseMessage(xmsg redis.XMessage, topic string) (*messaging.Message, error) {
	// 提取字段
	id, _ := xmsg.Values["id"].(string)
	payloadBase64, _ := xmsg.Values["payload"].(string)
	metadataJSON, _ := xmsg.Values["metadata"].(string)
	createdAtUnix, _ := xmsg.Values["created_at"].(string)

	// 解码负载
	payload, err := base64.StdEncoding.DecodeString(payloadBase64)
	if err != nil {
		return nil, fmt.Errorf("解码负载失败: %w", err)
	}

	// 解析元数据
	var metadata messaging.Metadata
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		return nil, fmt.Errorf("反序列化元数据失败: %w", err)
	}

	// 解析创建时间
	var createdAt time.Time
	if createdAtUnix != "" {
		var timestamp int64
		fmt.Sscanf(createdAtUnix, "%d", &timestamp)
		createdAt = time.Unix(timestamp, 0)
	}

	// 构造消息
	msg := &messaging.Message{
		ID:        id,
		Topic:     topic,
		Payload:   payload,
		Metadata:  metadata,
		CreatedAt: createdAt,
	}

	// 保存 Stream ID
	msg.Metadata.Set("stream_id", xmsg.ID)

	return msg, nil
}

// Close 关闭订阅者
func (s *Subscriber) Close(timeout time.Duration) error {
	// 取消所有订阅
	s.cancel()

	// 等待所有消费者停止（带超时）
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("等待订阅者关闭超时")
	}
}
