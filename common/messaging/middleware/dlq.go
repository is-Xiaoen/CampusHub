package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"activity-platform/common/messaging"
)

// DLQMiddleware 创建死信队列中间件
// 当消息处理失败且超过最大重试次数时，将消息发送到死信队列
func DLQMiddleware(dlqManager messaging.DLQManager, config messaging.DLQConfig) messaging.Middleware {
	return func(next messaging.HandlerFunc) messaging.HandlerFunc {
		return func(ctx context.Context, msg *messaging.Message) error {
			// 如果 DLQ 未启用，直接调用处理器
			if !config.Enabled {
				return next(ctx, msg)
			}

			// 调用处理器
			err := next(ctx, msg)

			// 如果处理成功，返回 nil
			if err == nil {
				return nil
			}

			// 检查是否超过最大重试次数
			if !isMaxRetriesExceeded(err) {
				// 未超过最大重试次数，返回错误继续重试
				return err
			}

			// 超过最大重试次数，将消息发送到 DLQ
			dlqMsg := buildDLQMessage(msg, err)

			if sendErr := dlqManager.Send(dlqMsg); sendErr != nil {
				// 发送到 DLQ 失败，记录错误但不阻塞
				// 返回原始错误
				return fmt.Errorf("发送消息到DLQ失败: %w (原始错误: %v)", sendErr, err)
			}

			// 消息已成功移至 DLQ，返回 nil 表示处理完成
			// 这样消息会被 ACK，不会再次重试
			return nil
		}
	}
}

// isMaxRetriesExceeded 检查是否超过最大重试次数
func isMaxRetriesExceeded(err error) bool {
	// 检查错误信息中是否包含 "超过最大重试次数"
	if err == nil {
		return false
	}

	errMsg := err.Error()
	return contains(errMsg, "超过最大重试次数") || contains(errMsg, "max retry attempts") || contains(errMsg, "exceeded")
}

// buildDLQMessage 构建死信队列消息
func buildDLQMessage(msg *messaging.Message, err error) *messaging.DLQMessage {
	// 获取重试次数
	retryCountStr := msg.Metadata.Get(messaging.MetadataKeyRetryCount)
	retryCount, _ := strconv.Atoi(retryCountStr)

	// 获取首次失败时间
	firstFailedAtStr := msg.Metadata.Get(messaging.MetadataKeyFirstFailedAt)
	firstFailedAt, _ := time.Parse(time.RFC3339, firstFailedAtStr)
	if firstFailedAt.IsZero() {
		firstFailedAt = time.Now()
	}

	// 获取最后错误
	lastError := msg.Metadata.Get(messaging.MetadataKeyLastError)
	if lastError == "" {
		lastError = err.Error()
	}

	// 构建错误历史
	errorHistory := []messaging.ErrorRecord{
		{
			Timestamp: time.Now(),
			Error:     err.Error(),
			Attempt:   retryCount,
		},
	}

	// 克隆原始消息（避免修改原消息）
	originalMsg := &messaging.Message{
		ID:         msg.ID,
		Topic:      msg.Topic,
		Payload:    msg.Payload,
		Metadata:   msg.Metadata.Clone(),
		CreatedAt:  msg.CreatedAt,
		ReceivedAt: msg.ReceivedAt,
	}

	return &messaging.DLQMessage{
		OriginalMessage: originalMsg,
		FailureReason:   lastError,
		FailureCount:    retryCount,
		FirstFailedAt:   firstFailedAt,
		LastFailedAt:    time.Now(),
		ErrorHistory:    errorHistory,
		MovedToDLQAt:    time.Now(),
	}
}

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
