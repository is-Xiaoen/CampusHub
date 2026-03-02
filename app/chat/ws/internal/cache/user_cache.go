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

// GetUserInfo 从缓存获取用户名和头像
// 返回 (name, avatar, ok)
func (c *UserCache) GetUserInfo(userID uint64) (name string, avatar string, ok bool) {
	key := c.getUserKey(userID)
	vals, err := c.redis.HMGet(c.ctx, key, "name", "avatar").Result()
	if err != nil || vals[0] == nil {
		return "", "", false
	}
	name, _ = vals[0].(string)
	if vals[1] != nil {
		avatar, _ = vals[1].(string)
	}
	return name, avatar, true
}

// SetUserInfo 将用户名和头像写入缓存
func (c *UserCache) SetUserInfo(userID uint64, name string, avatar string, ttl time.Duration) error {
	key := c.getUserKey(userID)
	if err := c.redis.HMSet(c.ctx, key, "name", name, "avatar", avatar).Err(); err != nil {
		return err
	}
	return c.redis.Expire(c.ctx, key, ttl).Err()
}

// Delete 删除用户缓存
func (c *UserCache) Delete(userID uint64) error {
	key := c.getUserKey(userID)
	return c.redis.Del(c.ctx, key).Err()
}

// getUserKey 生成用户缓存键
func (c *UserCache) getUserKey(userID uint64) string {
	return fmt.Sprintf("chat:user:info:%d", userID)
}
