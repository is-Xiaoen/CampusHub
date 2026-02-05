// Package cache 提供通用缓存工具
//
// 设计原则：
//   - Key 命名规范：{业务}:{模块}:{标识}，如 activity:detail:123
//   - 随机 TTL 防止缓存雪崩
//   - 与 go-zero cache.Cache 配合使用
package cache

import (
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/mathx"
)

// ==================== 默认配置 ====================

const (
	// DefaultTTL 默认缓存过期时间（5 分钟）
	DefaultTTL = 5 * time.Minute

	// LongTTL 长缓存过期时间（30 分钟，适用于变化少的数据如分类列表）
	LongTTL = 30 * time.Minute

	// DefaultJitter 默认 TTL 抖动系数（±10%）
	// 5min ± 10% = 4.5min ~ 5.5min
	DefaultJitter = 0.1
)

// unstable 随机数生成器，用于 TTL 抖动
var unstable = mathx.NewUnstable(DefaultJitter)

// ==================== TTL 工具函数 ====================

// RandomTTL 生成带抖动的 TTL，防止缓存雪崩
//
// 原理：
//   - 如果大量缓存同时设置相同 TTL，会在同一时间过期
//   - 大量请求同时穿透到 DB，造成缓存雪崩
//   - 添加 ±10% 随机抖动，使过期时间分散
//
// 示例：
//
//	RandomTTL(5 * time.Minute) => 4.5min ~ 5.5min
func RandomTTL(base time.Duration) time.Duration {
	return time.Duration(unstable.AroundDuration(base))
}

// RandomTTLSeconds 返回带抖动的 TTL（秒数）
//
// 用于需要秒数的场景，如 Redis SETEX
func RandomTTLSeconds(base time.Duration) int {
	return int(RandomTTL(base).Seconds())
}

// ==================== Key 生成函数 ====================

// 活动相关 Key

// ActivityDetailKey 活动详情缓存 Key
//
// 格式：activity:detail:{id}
// TTL：5min ± 10%
// 用途：缓存单个活动的完整信息
func ActivityDetailKey(id uint64) string {
	return fmt.Sprintf("activity:detail:%d", id)
}

// CategoryListKey 分类列表缓存 Key
//
// 格式：activity:category:list
// TTL：30min（分类数据变化少）
// 用途：缓存所有启用的活动分类
func CategoryListKey() string {
	return "activity:category:list"
}

// HotActivitiesKey 热门活动缓存 Key
//
// 格式：activity:hot:top10
// TTL：5min
// 用途：缓存按热度排序的 Top10 活动
func HotActivitiesKey() string {
	return "activity:hot:top10"
}

// ViewCountKey 浏览量防刷 Key
//
// 格式：activity:view:{activity_id}:{user_or_ip}
// TTL：1 小时
// 用途：防止同一用户/IP 短时间内重复计数
func ViewCountKey(activityID uint64, userOrIP string) string {
	return fmt.Sprintf("activity:view:%d:%s", activityID, userOrIP)
}

// ==================== 缓存统计 Key ====================

// CacheStatsKey 缓存统计 Key
func CacheStatsKey(cacheType string) string {
	return fmt.Sprintf("activity:cache:stats:%s", cacheType)
}
