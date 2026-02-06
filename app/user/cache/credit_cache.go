/**
 * @projectName: CampusHub
 * @package: cache
 * @className: CreditCache
 * @author: lijunqi
 * @description: 信用分缓存服务（Cache-Aside 模式）
 * @date: 2026-02-06
 * @version: 1.0
 */

package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"activity-platform/common/constants"

	"github.com/go-redis/redis/v8"
	"github.com/zeromicro/go-zero/core/logx"
)

// CreditCacheData 信用分缓存数据结构
type CreditCacheData struct {
	Score     int   `json:"score"`
	Level     int8  `json:"level"`
	UpdatedAt int64 `json:"updated_at"`
}

// ICreditCache 信用分缓存接口
type ICreditCache interface {
	// Get 获取用户信用分缓存
	// 返回: score, level, exists, error
	Get(ctx context.Context, userID int64) (*CreditCacheData, bool, error)

	// Set 设置用户信用分缓存（带防雪崩随机TTL）
	Set(ctx context.Context, userID int64, score int, level int8) error

	// Delete 删除用户信用分缓存
	Delete(ctx context.Context, userID int64) error
}

// CreditCache 信用分缓存实现
type CreditCache struct {
	redis *redis.Client
}

// NewCreditCache 创建信用分缓存实例
func NewCreditCache(rdb *redis.Client) ICreditCache {
	return &CreditCache{redis: rdb}
}

// Get 获取用户信用分缓存
func (c *CreditCache) Get(ctx context.Context, userID int64) (*CreditCacheData, bool, error) {
	cacheKey := c.buildKey(userID)
	logger := logx.WithContext(ctx)

	cached, err := c.redis.Get(ctx, cacheKey).Result()
	if err != nil {
		if err == redis.Nil {
			// 缓存未命中，不是错误
			return nil, false, nil
		}
		// Redis 错误
		logger.Errorf("[CreditCache] Redis读取失败: userId=%d, err=%v", userID, err)
		return nil, false, err
	}

	// 解析缓存数据
	var data CreditCacheData
	if err := json.Unmarshal([]byte(cached), &data); err != nil {
		// JSON 解析失败，删除脏数据
		logger.Errorf("[CreditCache] 缓存数据解析失败: userId=%d, err=%v", userID, err)
		c.redis.Del(ctx, cacheKey)
		return nil, false, nil
	}

	logger.Infof("[CreditCache] 缓存命中: userId=%d, score=%d, level=%d", userID, data.Score, data.Level)
	return &data, true, nil
}

// Set 设置用户信用分缓存
// TTL = 24h + random(0-1h) 防止缓存雪崩
func (c *CreditCache) Set(ctx context.Context, userID int64, score int, level int8) error {
	cacheKey := c.buildKey(userID)
	logger := logx.WithContext(ctx)

	data := CreditCacheData{
		Score:     score,
		Level:     level,
		UpdatedAt: time.Now().Unix(),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		logger.Errorf("[CreditCache] JSON序列化失败: userId=%d, err=%v", userID, err)
		return err
	}

	// TTL = 基础时间 + 随机偏移（防雪崩）
	ttl := constants.CacheUserCreditTTL + time.Duration(rand.Intn(constants.CacheUserCreditRandomMax))*time.Second

	if err := c.redis.Set(ctx, cacheKey, jsonData, ttl).Err(); err != nil {
		logger.Errorf("[CreditCache] Redis写入失败: userId=%d, err=%v", userID, err)
		return err
	}

	logger.Infof("[CreditCache] 缓存写入成功: userId=%d, score=%d, ttl=%v", userID, score, ttl)
	return nil
}

// Delete 删除用户信用分缓存
func (c *CreditCache) Delete(ctx context.Context, userID int64) error {
	cacheKey := c.buildKey(userID)
	logger := logx.WithContext(ctx)

	if err := c.redis.Del(ctx, cacheKey).Err(); err != nil {
		logger.Errorf("[CreditCache] 删除缓存失败: userId=%d, key=%s, err=%v", userID, cacheKey, err)
		return err
	}

	logger.Infof("[CreditCache] 缓存删除成功: userId=%d", userID)
	return nil
}

// buildKey 构建缓存键
func (c *CreditCache) buildKey(userID int64) string {
	return fmt.Sprintf("%s%d", constants.CacheUserCreditPrefix, userID)
}
