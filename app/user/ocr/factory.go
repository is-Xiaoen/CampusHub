/**
 * @projectName: CampusHub
 * @package: ocr
 * @className: factory
 * @author: lijunqi
 * @description: OCR策略工厂，实现双提供商故障转移和熔断机制
 * @date: 2026-01-31
 * @version: 1.0
 */

package ocr

import (
	"context"
	"time"

	"activity-platform/common/constants"
	"activity-platform/common/errorx"

	"github.com/go-redis/redis/v8"
	"github.com/zeromicro/go-zero/core/logx"
)

// ============================================================================
// 熔断器相关常量
// ============================================================================

const (
	// CircuitBreakerDuration 熔断持续时间（5分钟）
	CircuitBreakerDuration = 5 * time.Minute

	// FailureThreshold 触发熔断的失败次数阈值
	FailureThreshold = 3

	// FailureWindowDuration 失败计数窗口时间
	FailureWindowDuration = 5 * time.Minute
)

// ============================================================================
// ProviderFactory 提供商工厂
// ============================================================================

// ProviderFactory OCR提供商工厂
// 负责管理多个OCR提供商，实现故障转移和熔断逻辑
type ProviderFactory struct {
	// 主提供商（腾讯云）
	primary Provider

	// 备用提供商（阿里云）
	fallback Provider

	// Redis客户端（用于熔断状态存储）
	redis *redis.Client
}

// NewProviderFactory 创建提供商工厂
// 参数:
//   - primary: 主提供商（必须）
//   - fallback: 备用提供商（可选，传nil表示不使用备用）
//   - rdb: Redis客户端（用于熔断状态管理）
func NewProviderFactory(primary, fallback Provider, rdb *redis.Client) *ProviderFactory {
	return &ProviderFactory{
		primary:  primary,
		fallback: fallback,
		redis:    rdb,
	}
}

// ============================================================================
// 故障转移逻辑
// ============================================================================

// Recognize 执行OCR识别（带故障转移）
// 逻辑：
//  1. 检查主提供商是否被熔断
//  2. 未熔断 -> 尝试主提供商
//  3. 主提供商失败 -> 记录失败次数，达到阈值则熔断
//  4. 主提供商熔断或失败 -> 尝试备用提供商
//  5. 备用提供商也失败 -> 返回错误
func (f *ProviderFactory) Recognize(
	ctx context.Context,
	frontImageURL, backImageURL string,
) (*OcrResult, error) {
	// 1. 尝试主提供商
	if f.primary != nil {
		result, err := f.tryProvider(ctx, f.primary, frontImageURL, backImageURL)
		if err == nil {
			return result, nil
		}
		logx.WithContext(ctx).Errorf("主OCR提供商[%s]识别失败: %v", f.primary.Name(), err)
	}

	// 2. 尝试备用提供商
	if f.fallback != nil {
		result, err := f.tryProvider(ctx, f.fallback, frontImageURL, backImageURL)
		if err == nil {
			return result, nil
		}
		logx.WithContext(ctx).Errorf("备用OCR提供商[%s]识别失败: %v", f.fallback.Name(), err)
	}

	// 3. 所有提供商都失败
	return nil, errorx.ErrOcrServiceUnavailable()
}

// tryProvider 尝试使用指定提供商进行识别
func (f *ProviderFactory) tryProvider(
	ctx context.Context,
	provider Provider,
	frontImageURL, backImageURL string,
) (*OcrResult, error) {
	providerName := provider.Name()

	// 检查是否被熔断
	if f.isCircuitOpen(ctx, providerName) {
		logx.WithContext(ctx).Infof("OCR提供商[%s]已熔断，跳过", providerName)
		return nil, errorx.ErrOcrServiceUnavailable()
	}

	// 检查提供商是否可用
	if !provider.IsAvailable(ctx) {
		logx.WithContext(ctx).Infof("OCR提供商[%s]不可用，跳过", providerName)
		return nil, errorx.ErrOcrServiceUnavailable()
	}

	// 执行识别
	result, err := provider.Recognize(ctx, frontImageURL, backImageURL)
	if err != nil {
		// 失败，记录并检查是否需要熔断
		f.recordFailure(ctx, providerName)
		return nil, err
	}

	// 成功，重置失败计数
	f.resetFailureCount(ctx, providerName)
	return result, nil
}

// ============================================================================
// 熔断器实现
// ============================================================================

// isCircuitOpen 检查熔断器是否打开
func (f *ProviderFactory) isCircuitOpen(ctx context.Context, providerName string) bool {
	key := constants.OcrCircuitBreakerPrefix + providerName
	exists, err := f.redis.Exists(ctx, key).Result()
	if err != nil {
		logx.WithContext(ctx).Errorf("检查熔断器状态失败: %v", err)
		// Redis异常时不熔断，允许尝试
		return false
	}
	return exists > 0
}

// recordFailure 记录失败次数
func (f *ProviderFactory) recordFailure(ctx context.Context, providerName string) {
	failureKey := constants.OcrCircuitFailuresPrefix + providerName

	// 增加失败计数
	count, err := f.redis.Incr(ctx, failureKey).Result()
	if err != nil {
		logx.WithContext(ctx).Errorf("记录OCR失败次数失败: %v", err)
		return
	}

	// 设置过期时间（首次设置）
	if count == 1 {
		f.redis.Expire(ctx, failureKey, FailureWindowDuration)
	}

	// 达到阈值，触发熔断
	if count >= FailureThreshold {
		circuitKey := constants.OcrCircuitBreakerPrefix + providerName
		f.redis.Set(ctx, circuitKey, "1", CircuitBreakerDuration)
		logx.WithContext(ctx).Infof("[WARN] OCR提供商[%s]触发熔断，持续%v", providerName, CircuitBreakerDuration)
	}
}

// resetFailureCount 重置失败计数
func (f *ProviderFactory) resetFailureCount(ctx context.Context, providerName string) {
	failureKey := constants.OcrCircuitFailuresPrefix + providerName
	if err := f.redis.Del(ctx, failureKey).Err(); err != nil {
		logx.WithContext(ctx).Errorf("重置OCR失败计数失败: %v", err)
	}
}

// ============================================================================
// 辅助方法
// ============================================================================

// GetAvailableProvider 获取当前可用的提供商（用于日志记录）
func (f *ProviderFactory) GetAvailableProvider(ctx context.Context) Provider {
	if f.primary != nil && !f.isCircuitOpen(ctx, f.primary.Name()) && f.primary.IsAvailable(ctx) {
		return f.primary
	}
	if f.fallback != nil && !f.isCircuitOpen(ctx, f.fallback.Name()) && f.fallback.IsAvailable(ctx) {
		return f.fallback
	}
	return nil
}

// ResetCircuitBreaker 手动重置熔断器（用于测试或运维）
func (f *ProviderFactory) ResetCircuitBreaker(ctx context.Context, providerName string) error {
	circuitKey := constants.OcrCircuitBreakerPrefix + providerName
	failureKey := constants.OcrCircuitFailuresPrefix + providerName
	if err := f.redis.Del(ctx, circuitKey, failureKey).Err(); err != nil {
		return err
	}
	logx.WithContext(ctx).Infof("OCR提供商[%s]熔断器已重置", providerName)
	return nil
}

// GetCircuitStatus 获取熔断器状态（用于监控）
func (f *ProviderFactory) GetCircuitStatus(ctx context.Context, providerName string) (isOpen bool, failureCount int64) {
	circuitKey := constants.OcrCircuitBreakerPrefix + providerName
	failureKey := constants.OcrCircuitFailuresPrefix + providerName

	exists, _ := f.redis.Exists(ctx, circuitKey).Result()
	isOpen = exists > 0

	count, _ := f.redis.Get(ctx, failureKey).Int64()
	failureCount = count

	return
}
