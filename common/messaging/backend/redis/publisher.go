package redis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"CampusHub/common/messaging"

	"github.com/redis/go-redis/v9"
)

// Publisher Redis Streams 发布者实现
type Publisher struct {
	client *redis.Client
	config messaging.PublisherConfig
}

// NewPublisher 创建 Redis 发布者
func NewPublisher(client *redis.Client, config messaging.PublisherConfig) (*Publisher, error) {
	// 设置默认值
	if config.PublishTimeout == 0 {
		config.PublishTimeout = 5 * time.Second
	}

	return &Publisher{
		client: client,
		config: config,
	}, nil
}

// Publish 发布单条消息
func (p *Publisher) Publish(ctx context.Context, msg *messaging.Message) error {
	// 验证消息
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("消息验证失败: %w", err)
	}

	// 设置超时
	if p.config.PublishTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.config.PublishTimeout)
		defer cancel()
	}

	// 生成消息 ID（如果未设置）
	if msg.ID == "" {
		msg.ID = generateMessageID()
	}

	// 设置创建时间
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	// 序列化元数据
	metadataJSON, err := json.Marshal(msg.Metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	// 编码负载为 base64（Redis Streams 要求字符串值）
	payloadBase64 := base64.StdEncoding.EncodeToString(msg.Payload)

	// 使用 XADD 发布消息到 Redis Streams
	args := &redis.XAddArgs{
		Stream: msg.Topic,
		ID:     "*", // 让 Redis 自动生成 ID
		Values: map[string]interface{}{
			"id":         msg.ID,
			"payload":    payloadBase64,
			"metadata":   string(metadataJSON),
			"created_at": msg.CreatedAt.Unix(),
		},
	}

	streamID, err := p.client.XAdd(ctx, args).Result()
	if err != nil {
		return ErrPublishFailed(err)
	}

	// 更新消息的 Stream ID
	msg.Metadata.Set("stream_id", streamID)

	return nil
}

// PublishBatch 批量发布消息
func (p *Publisher) PublishBatch(ctx context.Context, msgs []*messaging.Message) error {
	if len(msgs) == 0 {
		return nil
	}

	// 设置超时
	if p.config.PublishTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.config.PublishTimeout)
		defer cancel()
	}

	// 使用 Pipeline 批量发布
	pipe := p.client.Pipeline()

	for _, msg := range msgs {
		// 验证消息
		if err := msg.Validate(); err != nil {
			return fmt.Errorf("消息验证失败: %w", err)
		}

		// 生成消息 ID
		if msg.ID == "" {
			msg.ID = generateMessageID()
		}

		// 设置创建时间
		if msg.CreatedAt.IsZero() {
			msg.CreatedAt = time.Now()
		}

		// 序列化元数据
		metadataJSON, err := json.Marshal(msg.Metadata)
		if err != nil {
			return fmt.Errorf("序列化元数据失败: %w", err)
		}

		// 编码负载
		payloadBase64 := base64.StdEncoding.EncodeToString(msg.Payload)

		// 添加到 pipeline
		args := &redis.XAddArgs{
			Stream: msg.Topic,
			ID:     "*",
			Values: map[string]interface{}{
				"id":         msg.ID,
				"payload":    payloadBase64,
				"metadata":   string(metadataJSON),
				"created_at": msg.CreatedAt.Unix(),
			},
		}
		pipe.XAdd(ctx, args)
	}

	// 执行 pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return ErrPublishFailed(err)
	}

	return nil
}

// Close 关闭发布者
func (p *Publisher) Close() error {
	// Redis 客户端由 Backend 管理，这里不需要关闭
	return nil
}

// generateMessageID 生成消息 ID
func generateMessageID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Nanosecond()%1000)
}
