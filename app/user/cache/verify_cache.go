/**
 * @projectName: CampusHub
 * @package: cache
 * @className: VerifyCache
 * @author: lijunqi
 * @description: 认证状态缓存服务
 * @date: 2026-02-06
 * @version: 1.0
 */

package cache

import (
	"context"
	"fmt"

	"activity-platform/common/constants"

	"github.com/go-redis/redis/v8"
	"github.com/zeromicro/go-zero/core/logx"
)

// IVerifyCache 认证状态缓存接口
type IVerifyCache interface {
	// Get 获取用户认证状态缓存
	// 返回: isVerified, exists, error
	Get(ctx context.Context, userID int64) (bool, bool, error)

	// Set 设置用户认证状态缓存
	Set(ctx context.Context, userID int64, isVerified bool) error

	// Delete 删除用户认证状态缓存
	Delete(ctx context.Context, userID int64) error
}

// VerifyCache 认证状态缓存实现
type VerifyCache struct {
	redis *redis.Client
}

// NewVerifyCache 创建认证状态缓存实例
func NewVerifyCache(rdb *redis.Client) IVerifyCache {
	return &VerifyCache{redis: rdb}
}

// Get 获取用户认证状态缓存
func (c *VerifyCache) Get(ctx context.Context, userID int64) (bool, bool, error) {
	cacheKey := c.buildKey(userID)
	logger := logx.WithContext(ctx)

	cached, err := c.redis.Get(ctx, cacheKey).Result()
	if err != nil {
		if err == redis.Nil {
			// 缓存未命中，不是错误
			return false, false, nil
		}
		// Redis 错误
		logger.Errorf("[VerifyCache] Redis读取失败: userId=%d, err=%v", userID, err)
		return false, false, err
	}

	isVerified := cached == "1"
	logger.Infof("[VerifyCache] 缓存命中: userId=%d, isVerified=%v", userID, isVerified)
	return isVerified, true, nil
}

// Set 设置用户认证状态缓存
// TTL = 7天（认证状态变化频率低）
func (c *VerifyCache) Set(ctx context.Context, userID int64, isVerified bool) error {
	cacheKey := c.buildKey(userID)
	logger := logx.WithContext(ctx)

	value := "0"
	if isVerified {
		value = "1"
	}

	if err := c.redis.Set(ctx, cacheKey, value, constants.CacheUserVerifiedTTL).Err(); err != nil {
		logger.Errorf("[VerifyCache] Redis写入失败: userId=%d, err=%v", userID, err)
		return err
	}

	logger.Infof("[VerifyCache] 缓存写入成功: userId=%d, isVerified=%v", userID, isVerified)
	return nil
}

// Delete 删除用户认证状态缓存
func (c *VerifyCache) Delete(ctx context.Context, userID int64) error {
	cacheKey := c.buildKey(userID)
	logger := logx.WithContext(ctx)

	if err := c.redis.Del(ctx, cacheKey).Err(); err != nil {
		logger.Errorf("[VerifyCache] 删除缓存失败: userId=%d, key=%s, err=%v", userID, cacheKey, err)
		return err
	}

	logger.Infof("[VerifyCache] 缓存删除成功: userId=%d", userID)
	return nil
}

// buildKey 构建缓存键
func (c *VerifyCache) buildKey(userID int64) string {
	return fmt.Sprintf("%s%d", constants.CacheUserVerifiedPrefix, userID)
}
