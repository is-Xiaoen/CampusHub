package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// UserCache 用户信息缓存
type UserCache struct {
	redis *redis.Client
	ctx   context.Context
}

// NewUserCache 创建用户缓存实例
func NewUserCache(redisClient *redis.Client) *UserCache {
	return &UserCache{
		redis: redisClient,
		ctx:   context.Background(),
	}
}

// Get 从缓存获取用户名
// 返回用户名和是否存在
func (c *UserCache) Get(userID uint64) (string, bool) {
	key := c.getUserKey(userID)
	val, err := c.redis.Get(c.ctx, key).Result()
	if err != nil {
		return "", false
	}
	return val, true
}

// Set 设置用户名到缓存
// ttl: 缓存过期时间
func (c *UserCache) Set(userID uint64, userName string, ttl time.Duration) error {
	key := c.getUserKey(userID)
	return c.redis.Set(c.ctx, key, userName, ttl).Err()
}

// Delete 删除用户缓存
func (c *UserCache) Delete(userID uint64) error {
	key := c.getUserKey(userID)
	return c.redis.Del(c.ctx, key).Err()
}

// getUserKey 生成用户缓存键
func (c *UserCache) getUserKey(userID uint64) string {
	return fmt.Sprintf("chat:user:name:%d", userID)
}
