package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"activity-platform/app/activity/model"
	commonCache "activity-platform/common/cache"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

// ==================== HotCache 热门活动缓存 ====================
//
// 功能说明：
//   - 缓存按热度排序的 Top N 活动
//   - 热度算法：报名人数 * 0.6 + 浏览量 * 0.4
//
// 缓存策略：
//   - Key: activity:hot:top10
//   - TTL: 5min
//   - 失效时机: 通过 TTL 自动过期（热门活动变化频繁，不主动删除）

// HotCache 热门活动缓存服务
type HotCache struct {
	rds     *redis.Redis
	db      *gorm.DB
	sfGroup singleflight.Group // singleflight 防止缓存击穿
}

// NewHotCache 创建热门活动缓存服务
func NewHotCache(rds *redis.Redis, db *gorm.DB) *HotCache {
	return &HotCache{
		rds: rds,
		db:  db,
	}
}

// HotActivityCacheData 热门活动缓存数据结构
//
// 说明：只缓存列表展示需要的字段，不包含详情字段
type HotActivityCacheData struct {
	ID                  uint64 `json:"id"`
	Title               string `json:"title"`
	CoverURL            string `json:"cover_url"`
	CoverType           int8   `json:"cover_type"`
	CategoryID          uint64 `json:"category_id"`
	OrganizerName       string `json:"organizer_name"`
	OrganizerAvatar     string `json:"organizer_avatar"`
	ActivityStartTime   int64  `json:"activity_start_time"`
	Location            string `json:"location"`
	MaxParticipants     uint32 `json:"max_participants"`
	CurrentParticipants uint32 `json:"current_participants"`
	Status              int8   `json:"status"`
	ViewCount           uint32 `json:"view_count"`
	CreatedAt           int64  `json:"created_at"`
}

// GetTopN 获取热门活动列表（带缓存）
//
// 参数：
//   - ctx: 上下文
//   - limit: 返回数量（1-20，默认 10）
//
// 返回值：
//   - []model.Activity: 热门活动列表
//   - error: 错误信息
func (c *HotCache) GetTopN(ctx context.Context, limit int) ([]model.Activity, error) {
	// 参数规范化
	if limit <= 0 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}

	key := commonCache.HotActivitiesKey()

	// 1. 尝试从缓存获取
	val, err := c.rds.GetCtx(ctx, key)
	if err != nil && !errors.Is(err, redis.Nil) {
		// Redis 错误，降级查询 DB
		logx.WithContext(ctx).Errorf("[HotCache] Redis 错误，降级查 DB: err=%v", err)
		return c.getFromDB(ctx, limit)
	}

	// 2. 缓存命中
	if val != "" {
		var cacheList []HotActivityCacheData
		if err := json.Unmarshal([]byte(val), &cacheList); err != nil {
			logx.WithContext(ctx).Errorf("[HotCache] 反序列化失败: err=%v", err)
			_, _ = c.rds.DelCtx(ctx, key)
			return c.getFromDBAndCache(ctx, limit, key)
		}

		// 裁剪到请求数量
		activities := c.toActivities(cacheList)
		if limit < len(activities) {
			activities = activities[:limit]
		}
		return activities, nil
	}

	// 3. 缓存未命中，使用 singleflight 保护
	sfKey := fmt.Sprintf("%s:%d", key, limit)
	result, err, _ := c.sfGroup.Do(sfKey, func() (interface{}, error) {
		return c.getFromDBAndCache(ctx, limit, key)
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return []model.Activity{}, nil
	}
	return result.([]model.Activity), nil
}

// getFromDB 直接从数据库查询热门活动
//
// 热度算法说明：
//   - 简化版：current_participants DESC（按报名人数）
//   - 完整版：current_participants * 0.6 + view_count * 0.4
//
// 筛选条件：
//   - 状态为已发布(2)或进行中(3)
//   - 活动未结束（activity_end_time > NOW()）
func (c *HotCache) getFromDB(ctx context.Context, limit int) ([]model.Activity, error) {
	now := time.Now().Unix()
	var activities []model.Activity

	// 使用简化版热度算法：按报名人数排序
	// 后续可替换为加权算法
	err := c.db.WithContext(ctx).
		Where("status IN ? AND activity_end_time > ?",
			[]int8{model.StatusPublished, model.StatusOngoing}, now).
		Order("current_participants DESC, created_at DESC").
		Limit(limit).
		Find(&activities).Error

	return activities, err
}

// getFromDBAndCache 从 DB 查询并写入缓存
func (c *HotCache) getFromDBAndCache(ctx context.Context, limit int, key string) ([]model.Activity, error) {
	// 查询时获取更多数据（缓存 Top20，请求时裁剪）
	cacheLimit := 20
	if limit > cacheLimit {
		cacheLimit = limit
	}

	activities, err := c.getFromDB(ctx, cacheLimit)
	if err != nil {
		return nil, err
	}

	// 转换并序列化
	cacheList := c.toCacheDataList(activities)
	data, err := json.Marshal(cacheList)
	if err != nil {
		logx.WithContext(ctx).Errorf("[HotCache] 序列化失败: err=%v", err)
		// 裁剪并返回
		if limit < len(activities) {
			return activities[:limit], nil
		}
		return activities, nil
	}

	// 写入缓存
	ttl := commonCache.RandomTTLSeconds(commonCache.DefaultTTL)
	if err := c.rds.SetexCtx(ctx, key, string(data), ttl); err != nil {
		logx.WithContext(ctx).Errorf("[HotCache] 写入缓存失败: err=%v", err)
	}

	// 裁剪到请求数量
	if limit < len(activities) {
		return activities[:limit], nil
	}
	return activities, nil
}

// Refresh 刷新热门活动缓存
//
// 调用时机：
//   - 手动触发刷新
//   - 定时任务刷新（可选）
func (c *HotCache) Refresh(ctx context.Context) error {
	key := commonCache.HotActivitiesKey()
	if _, err := c.rds.DelCtx(ctx, key); err != nil {
		logx.WithContext(ctx).Errorf("[HotCache] 删除缓存失败: err=%v", err)
		return err
	}
	return nil
}

// Warmup 预热热门活动缓存
//
// 调用时机：服务启动时
func (c *HotCache) Warmup(ctx context.Context) error {
	key := commonCache.HotActivitiesKey()
	_, err := c.getFromDBAndCache(ctx, 10, key)
	return err
}

// ==================== 数据转换 ====================

func (c *HotCache) toCacheDataList(activities []model.Activity) []HotActivityCacheData {
	result := make([]HotActivityCacheData, len(activities))
	for i, act := range activities {
		result[i] = HotActivityCacheData{
			ID:                  act.ID,
			Title:               act.Title,
			CoverURL:            act.CoverURL,
			CoverType:           act.CoverType,
			CategoryID:          act.CategoryID,
			OrganizerName:       act.OrganizerName,
			OrganizerAvatar:     act.OrganizerAvatar,
			ActivityStartTime:   act.ActivityStartTime,
			Location:            act.Location,
			MaxParticipants:     act.MaxParticipants,
			CurrentParticipants: act.CurrentParticipants,
			Status:              act.Status,
			ViewCount:           act.ViewCount,
			CreatedAt:           act.CreatedAt,
		}
	}
	return result
}

func (c *HotCache) toActivities(cacheList []HotActivityCacheData) []model.Activity {
	result := make([]model.Activity, len(cacheList))
	for i, d := range cacheList {
		result[i] = model.Activity{
			ID:                  d.ID,
			Title:               d.Title,
			CoverURL:            d.CoverURL,
			CoverType:           d.CoverType,
			CategoryID:          d.CategoryID,
			OrganizerName:       d.OrganizerName,
			OrganizerAvatar:     d.OrganizerAvatar,
			ActivityStartTime:   d.ActivityStartTime,
			Location:            d.Location,
			MaxParticipants:     d.MaxParticipants,
			CurrentParticipants: d.CurrentParticipants,
			Status:              d.Status,
			ViewCount:           d.ViewCount,
			CreatedAt:           d.CreatedAt,
		}
	}
	return result
}
