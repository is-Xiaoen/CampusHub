package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"CampusHub/common/messaging"

	"github.com/redis/go-redis/v9"
)

// DLQManager Redis 死信队列管理器实现
type DLQManager struct {
	client *redis.Client
	config messaging.DLQConfig
	ctx    context.Context
}

// NewDLQManager 创建 Redis DLQ 管理器
func NewDLQManager(client *redis.Client, config messaging.DLQConfig) *DLQManager {
	return &DLQManager{
		client: client,
		config: config,
		ctx:    context.Background(),
	}
}

// Send 发送消息到死信队列
func (m *DLQManager) Send(dlqMsg *messaging.DLQMessage) error {
	// 构建 DLQ 主题名称
	dlqTopic := m.getDLQTopic(dlqMsg.OriginalMessage.Topic)

	// 序列化 DLQ 消息
	data, err := json.Marshal(dlqMsg)
	if err != nil {
		return fmt.Errorf("序列化DLQ消息失败: %w", err)
	}

	// 使用 XADD 添加到 DLQ Stream
	args := &redis.XAddArgs{
		Stream: dlqTopic,
		Values: map[string]interface{}{
			"message_id":    dlqMsg.OriginalMessage.ID,
			"data":          string(data),
			"moved_at":      dlqMsg.MovedToDLQAt.Unix(),
			"failure_count": dlqMsg.FailureCount,
		},
	}

	// 如果配置了保留期，设置 MAXLEN
	if m.config.RetentionPeriod > 0 {
		// 估算最大长度（假设每秒 100 条消息）
		maxLen := int64(m.config.RetentionPeriod.Seconds() * 100)
		args.MaxLen = maxLen
		args.Approx = true // 使用近似裁剪以提高性能
	}

	if _, err := m.client.XAdd(m.ctx, args).Result(); err != nil {
		return fmt.Errorf("添加消息到DLQ失败: %w", err)
	}

	return nil
}

// List 列出死信队列中的消息
func (m *DLQManager) List(topic string, offset, limit int) ([]*messaging.DLQMessage, error) {
	dlqTopic := m.getDLQTopic(topic)

	// 使用 XRANGE 读取消息
	// offset 转换为 stream ID（使用时间戳）
	start := "-"
	if offset > 0 {
		// 跳过前 offset 条消息
		// 先读取 offset 条获取最后一个 ID
		skipMsgs, err := m.client.XRange(m.ctx, dlqTopic, "-", "+").Result()
		if err != nil {
			return nil, fmt.Errorf("读取DLQ消息失败: %w", err)
		}
		if offset < len(skipMsgs) {
			start = skipMsgs[offset].ID
		} else {
			// offset 超出范围，返回空列表
			return []*messaging.DLQMessage{}, nil
		}
	}

	// 读取消息
	messages, err := m.client.XRange(m.ctx, dlqTopic, start, "+").Result()
	if err != nil {
		return nil, fmt.Errorf("读取DLQ消息失败: %w", err)
	}

	// 限制返回数量
	if limit > 0 && len(messages) > limit {
		messages = messages[:limit]
	}

	// 解析消息
	result := make([]*messaging.DLQMessage, 0, len(messages))
	for _, msg := range messages {
		dataStr, ok := msg.Values["data"].(string)
		if !ok {
			continue
		}

		var dlqMsg messaging.DLQMessage
		if err := json.Unmarshal([]byte(dataStr), &dlqMsg); err != nil {
			continue
		}

		result = append(result, &dlqMsg)
	}

	return result, nil
}

// Get 获取指定的死信队列消息
func (m *DLQManager) Get(topic, messageID string) (*messaging.DLQMessage, error) {
	dlqTopic := m.getDLQTopic(topic)

	// 读取所有消息并查找匹配的消息 ID
	messages, err := m.client.XRange(m.ctx, dlqTopic, "-", "+").Result()
	if err != nil {
		return nil, fmt.Errorf("读取DLQ消息失败: %w", err)
	}

	for _, msg := range messages {
		msgID, ok := msg.Values["message_id"].(string)
		if !ok || msgID != messageID {
			continue
		}

		dataStr, ok := msg.Values["data"].(string)
		if !ok {
			continue
		}

		var dlqMsg messaging.DLQMessage
		if err := json.Unmarshal([]byte(dataStr), &dlqMsg); err != nil {
			return nil, fmt.Errorf("反序列化DLQ消息失败: %w", err)
		}

		return &dlqMsg, nil
	}

	return nil, fmt.Errorf("DLQ中未找到消息: %s", messageID)
}

// Reprocess 重新处理死信队列消息
func (m *DLQManager) Reprocess(topic, messageID string) error {
	// 获取 DLQ 消息
	dlqMsg, err := m.Get(topic, messageID)
	if err != nil {
		return err
	}

	// 创建发布者将消息重新发送到原始主题
	pubConfig := messaging.DefaultPublisherConfig()
	publisher, err := NewPublisher(m.client, pubConfig)
	if err != nil {
		return fmt.Errorf("创建发布者失败: %w", err)
	}

	// 清除重试相关的元数据
	dlqMsg.OriginalMessage.Metadata.Delete(messaging.MetadataKeyRetryCount)
	dlqMsg.OriginalMessage.Metadata.Delete(messaging.MetadataKeyLastError)
	dlqMsg.OriginalMessage.Metadata.Delete(messaging.MetadataKeyFirstFailedAt)

	// 添加重新处理标记
	dlqMsg.OriginalMessage.Metadata.Set("reprocessed_from_dlq", "true")
	dlqMsg.OriginalMessage.Metadata.Set("reprocessed_at", time.Now().Format(time.RFC3339))

	// 重新发布消息
	if err := publisher.Publish(m.ctx, dlqMsg.OriginalMessage); err != nil {
		return fmt.Errorf("重新发布消息失败: %w", err)
	}

	// 从 DLQ 中删除消息
	return m.Delete(topic, messageID)
}

// ReprocessBatch 批量重新处理死信队列消息
func (m *DLQManager) ReprocessBatch(topic string, messageIDs []string) error {
	var lastErr error
	successCount := 0

	for _, messageID := range messageIDs {
		if err := m.Reprocess(topic, messageID); err != nil {
			lastErr = err
			continue
		}
		successCount++
	}

	if lastErr != nil {
		return fmt.Errorf("已重新处理 %d/%d 条消息, 最后错误: %w",
			successCount, len(messageIDs), lastErr)
	}

	return nil
}

// Delete 删除死信队列消息
func (m *DLQManager) Delete(topic, messageID string) error {
	dlqTopic := m.getDLQTopic(topic)

	// 查找消息的 Stream ID
	messages, err := m.client.XRange(m.ctx, dlqTopic, "-", "+").Result()
	if err != nil {
		return fmt.Errorf("读取DLQ消息失败: %w", err)
	}

	var streamID string
	for _, msg := range messages {
		msgID, ok := msg.Values["message_id"].(string)
		if ok && msgID == messageID {
			streamID = msg.ID
			break
		}
	}

	if streamID == "" {
		return fmt.Errorf("DLQ中未找到消息: %s", messageID)
	}

	// 使用 XDEL 删除消息
	if err := m.client.XDel(m.ctx, dlqTopic, streamID).Err(); err != nil {
		return fmt.Errorf("从DLQ删除消息失败: %w", err)
	}

	return nil
}

// DeleteBatch 批量删除死信队列消息
func (m *DLQManager) DeleteBatch(topic string, messageIDs []string) error {
	dlqTopic := m.getDLQTopic(topic)

	// 查找所有消息的 Stream ID
	messages, err := m.client.XRange(m.ctx, dlqTopic, "-", "+").Result()
	if err != nil {
		return fmt.Errorf("读取DLQ消息失败: %w", err)
	}

	// 构建 messageID -> streamID 映射
	streamIDs := make([]string, 0, len(messageIDs))
	for _, msg := range messages {
		msgID, ok := msg.Values["message_id"].(string)
		if !ok {
			continue
		}

		for _, targetID := range messageIDs {
			if msgID == targetID {
				streamIDs = append(streamIDs, msg.ID)
				break
			}
		}
	}

	if len(streamIDs) == 0 {
		return fmt.Errorf("DLQ中未找到消息")
	}

	// 批量删除
	if err := m.client.XDel(m.ctx, dlqTopic, streamIDs...).Err(); err != nil {
		return fmt.Errorf("从DLQ批量删除消息失败: %w", err)
	}

	return nil
}

// Count 统计死信队列消息数量
func (m *DLQManager) Count(topic string) (int64, error) {
	dlqTopic := m.getDLQTopic(topic)

	// 使用 XLEN 获取 Stream 长度
	count, err := m.client.XLen(m.ctx, dlqTopic).Result()
	if err != nil {
		return 0, fmt.Errorf("获取DLQ计数失败: %w", err)
	}

	return count, nil
}

// Purge 清空指定主题的死信队列
func (m *DLQManager) Purge(topic string) error {
	dlqTopic := m.getDLQTopic(topic)

	// 删除整个 Stream
	if err := m.client.Del(m.ctx, dlqTopic).Err(); err != nil {
		return fmt.Errorf("清空DLQ失败: %w", err)
	}

	return nil
}

// Close 关闭 DLQ 管理器
func (m *DLQManager) Close() error {
	// Redis DLQ 管理器不需要特殊的关闭操作
	return nil
}

// getDLQTopic 获取 DLQ 主题名称
func (m *DLQManager) getDLQTopic(topic string) string {
	return topic + m.config.TopicSuffix
}
