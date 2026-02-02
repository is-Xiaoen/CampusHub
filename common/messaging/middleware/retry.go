package middleware

import (
	"context"
	"fmt"
	"math"
	"time"

	"CampusHub/common/messaging"
)

// RetryMiddleware 创建重试中间件
func RetryMiddleware(policy messaging.RetryPolicy) messaging.Middleware {
	return func(next messaging.HandlerFunc) messaging.HandlerFunc {
		return func(ctx context.Context, msg *messaging.Message) error {
			// 如果重试未启用，直接调用处理器
			if !policy.Enabled {
				return next(ctx, msg)
			}

			var lastErr error

			// 尝试处理消息，最多 MaxAttempts 次
			for attempt := 0; attempt < policy.MaxAttempts; attempt++ {
				// 如果不是第一次尝试，等待重试延迟
				if attempt > 0 {
					delay := calculateRetryDelay(policy, attempt)
					select {
					case <-time.After(delay):
						// 延迟完成，继续处理
					case <-ctx.Done():
						// 上下文已取消
						return ctx.Err()
					}
				}

				// 更新重试次数
				msg.Metadata.Set(messaging.MetadataKeyRetryCount, fmt.Sprintf("%d", attempt+1))

				// 调用处理器
				err := next(ctx, msg)

				// 如果处理成功，返回 nil
				if err == nil {
					return nil
				}

				// 记录错误
				lastErr = err
				recordError(msg, err)

				// 检查错误是否可重试
				if !isRetryableError(err, policy) {
					// 不可重试的错误，直接返回
					return err
				}

				// 如果是最后一次尝试，返回错误
				if attempt == policy.MaxAttempts-1 {
					return fmt.Errorf("超过最大重试次数 (%d): %w", policy.MaxAttempts, lastErr)
				}
			}

			// 理论上不会到达这里，但为了安全起见
			return fmt.Errorf("超过最大重试次数 (%d): %w", policy.MaxAttempts, lastErr)
		}
	}
}

// calculateRetryDelay 计算重试延迟
// 使用指数退避算法：delay = min(InitialInterval * Multiplier^(attempt-1), MaxInterval)
func calculateRetryDelay(policy messaging.RetryPolicy, attempt int) time.Duration {
	// 计算指数退避延迟
	delay := float64(policy.InitialInterval) * math.Pow(policy.Multiplier, float64(attempt-1))

	// 限制最大延迟
	if delay > float64(policy.MaxInterval) {
		delay = float64(policy.MaxInterval)
	}

	return time.Duration(delay)
}

// getRetryCount 获取消息的重试次数
func getRetryCount(msg *messaging.Message) int {
	if countStr := msg.Metadata.Get(messaging.MetadataKeyRetryCount); countStr != "" {
		var count int
		fmt.Sscanf(countStr, "%d", &count)
		return count
	}
	return 0
}

// incrementRetryCount 增加重试次数
func incrementRetryCount(msg *messaging.Message) {
	count := getRetryCount(msg)
	count++
	msg.Metadata.Set(messaging.MetadataKeyRetryCount, fmt.Sprintf("%d", count))
}

// recordError 记录错误信息
func recordError(msg *messaging.Message, err error) {
	// 记录最后一次错误
	msg.Metadata.Set(messaging.MetadataKeyLastError, err.Error())

	// 记录首次失败时间（如果还没有记录）
	if !msg.Metadata.Has(messaging.MetadataKeyFirstFailedAt) {
		msg.Metadata.Set(messaging.MetadataKeyFirstFailedAt, time.Now().Format(time.RFC3339))
	}
}

// isRetryableError 检查错误是否可重试
func isRetryableError(err error, policy messaging.RetryPolicy) bool {
	// 如果配置了特定的可重试错误类型，检查是否匹配
	if len(policy.RetryableErrors) > 0 {
		for _, retryableErr := range policy.RetryableErrors {
			if err == retryableErr {
				return true
			}
		}
		return false
	}

	// 使用错误分类逻辑判断
	return messaging.IsRetryable(err)
}
