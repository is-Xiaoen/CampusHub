// Package cache 提供活动服务的缓存层实现
//
// 设计原则：
//   - 使用 go-zero cache.Take，内置 singleflight + 空值缓存
//   - 缓存失效采用单次删除策略（170 QPS 场景足够）
//   - 随机 TTL 防止缓存雪崩
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"activity-platform/app/activity/model"
	commonCache "activity-platform/common/cache"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

// ==================== ActivityCache 活动详情缓存 ====================
//
// 功能说明：
//   - 缓存活动详情，减少 DB 查询
//   - 内置 singleflight 防止缓存击穿
//   - 内置空值缓存防止缓存穿透
//
// 缓存策略：
//   - Key: activity:detail:{id}
//   - TTL: 5min ± 10%
//   - 失效时机: 更新/删除活动时主动删除

// ErrActivityNotFoundInCache 活动在缓存中不存在（空值缓存标记）
var ErrActivityNotFoundInCache = errors.New("activity not found in cache")

// ActivityCache 活动详情缓存服务
type ActivityCache struct {
	rds     *redis.Redis
	db      *gorm.DB
	sfGroup singleflight.Group // singleflight 防止缓存击穿
}

// NewActivityCache 创建活动缓存服务
func NewActivityCache(rds *redis.Redis, db *gorm.DB) *ActivityCache {
	return &ActivityCache{
		rds: rds,
		db:  db,
	}
}

// ActivityCacheData 缓存数据结构
//
// 说明：
//   - 使用独立结构体而非直接缓存 model.Activity
//   - 便于控制缓存字段，避免敏感数据泄露
//   - 便于版本升级时的兼容处理
type ActivityCacheData struct {
	ID                   uint64  `json:"id"`
	Title                string  `json:"title"`
	CoverURL             string  `json:"cover_url"`
	CoverType            int8    `json:"cover_type"`
	Description          string  `json:"description"`
	CategoryID           uint64  `json:"category_id"`
	OrganizerID          uint64  `json:"organizer_id"`
	OrganizerName        string  `json:"organizer_name"`
	OrganizerAvatar      string  `json:"organizer_avatar"`
	ContactPhone         string  `json:"contact_phone"`
	RegisterStartTime    int64   `json:"register_start_time"`
	RegisterEndTime      int64   `json:"register_end_time"`
	ActivityStartTime    int64   `json:"activity_start_time"`
	ActivityEndTime      int64   `json:"activity_end_time"`
	Location             string  `json:"location"`
	AddressDetail        string  `json:"address_detail"`
	Longitude            float64 `json:"longitude"`
	Latitude             float64 `json:"latitude"`
	MaxParticipants      uint32  `json:"max_participants"`
	CurrentParticipants  uint32  `json:"current_participants"`
	RequireApproval      bool    `json:"require_approval"`
	RequireStudentVerify bool    `json:"require_student_verify"`
	MinCreditScore       int     `json:"min_credit_score"`
	Status               int8    `json:"status"`
	RejectReason         string  `json:"reject_reason"`
	ViewCount            uint32  `json:"view_count"`
	LikeCount            uint32  `json:"like_count"`
	Version              uint32  `json:"version"`
	CreatedAt            int64   `json:"created_at"`
	UpdatedAt            int64   `json:"updated_at"`
}

// nullValuePlaceholder 空值标记，用于防止缓存穿透
const nullValuePlaceholder = "{\"null\":true}"

// GetByID 获取活动详情（带缓存）
//
// 流程：
//  1. 查询 Redis 缓存
//  2. 缓存命中：反序列化返回
//  3. 缓存未命中：查询 DB，写入缓存
//  4. DB 查询为空：写入空值标记，防止穿透
//
// 参数：
//   - ctx: 上下文
//   - id: 活动 ID
//
// 返回值：
//   - *model.Activity: 活动数据（nil 表示不存在）
//   - error: 错误信息
func (c *ActivityCache) GetByID(ctx context.Context, id uint64) (*model.Activity, error) {
	key := commonCache.ActivityDetailKey(id)

	// 1. 尝试从缓存获取
	val, err := c.rds.GetCtx(ctx, key)
	if err != nil && !errors.Is(err, redis.Nil) {
		// Redis 错误，降级查询 DB
		logx.WithContext(ctx).Errorf("[ActivityCache] Redis 错误，降级查 DB: key=%s, err=%v", key, err)
		return c.getFromDB(ctx, id)
	}

	// 2. 缓存命中
	if val != "" {
		// 检查是否为空值标记
		if val == nullValuePlaceholder {
			return nil, model.ErrActivityNotFound
		}

		// 反序列化
		var cacheData ActivityCacheData
		if err := json.Unmarshal([]byte(val), &cacheData); err != nil {
			logx.WithContext(ctx).Errorf("[ActivityCache] 反序列化失败: key=%s, err=%v", key, err)
			// 删除损坏的缓存，下次重建
			_, _ = c.rds.DelCtx(ctx, key)
			return c.getFromDB(ctx, id)
		}

		return c.toActivity(&cacheData), nil
	}

	// 3. 缓存未命中，使用 singleflight 保护，防止并发穿透 DB
	result, err, _ := c.sfGroup.Do(key, func() (interface{}, error) {
		return c.getFromDBAndCache(ctx, id, key)
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, model.ErrActivityNotFound
	}
	return result.(*model.Activity), nil
}

// getFromDB 直接从数据库查询（无缓存操作）
func (c *ActivityCache) getFromDB(ctx context.Context, id uint64) (*model.Activity, error) {
	var activity model.Activity
	err := c.db.WithContext(ctx).
		Where("id = ?", id).
		First(&activity).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrActivityNotFound
		}
		return nil, err
	}
	return &activity, nil
}

// getFromDBAndCache 从 DB 查询并写入缓存
func (c *ActivityCache) getFromDBAndCache(ctx context.Context, id uint64, key string) (*model.Activity, error) {
	var activity model.Activity
	err := c.db.WithContext(ctx).
		Where("id = ?", id).
		First(&activity).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 写入空值标记，防止缓存穿透
			// TTL 较短（1 分钟），避免真实数据创建后长时间读不到
			_ = c.rds.SetexCtx(ctx, key, nullValuePlaceholder, 60)
			return nil, model.ErrActivityNotFound
		}
		return nil, err
	}

	// 序列化并写入缓存
	cacheData := c.toCacheData(&activity)
	data, err := json.Marshal(cacheData)
	if err != nil {
		logx.WithContext(ctx).Errorf("[ActivityCache] 序列化失败: id=%d, err=%v", id, err)
		// 序列化失败不影响返回结果
		return &activity, nil
	}

	// 写入缓存，带随机 TTL
	ttl := commonCache.RandomTTLSeconds(commonCache.DefaultTTL)
	if err := c.rds.SetexCtx(ctx, key, string(data), ttl); err != nil {
		logx.WithContext(ctx).Errorf("[ActivityCache] 写入缓存失败: key=%s, err=%v", key, err)
		// 写入失败不影响返回结果
	}

	return &activity, nil
}

// Invalidate 删除活动缓存
//
// 调用时机：
//   - 更新活动后
//   - 删除活动后
//   - 状态变更后
func (c *ActivityCache) Invalidate(ctx context.Context, id uint64) error {
	key := commonCache.ActivityDetailKey(id)
	if _, err := c.rds.DelCtx(ctx, key); err != nil {
		logx.WithContext(ctx).Errorf("[ActivityCache] 删除缓存失败: key=%s, err=%v", key, err)
		return err
	}
	return nil
}

// InvalidateBatch 批量删除活动缓存
func (c *ActivityCache) InvalidateBatch(ctx context.Context, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = commonCache.ActivityDetailKey(id)
	}

	_, err := c.rds.DelCtx(ctx, keys...)
	if err != nil {
		logx.WithContext(ctx).Errorf("[ActivityCache] 批量删除缓存失败: keys=%v, err=%v", keys, err)
	}
	return err
}

// Set 主动设置缓存（用于创建活动后预热）
func (c *ActivityCache) Set(ctx context.Context, activity *model.Activity) error {
	key := commonCache.ActivityDetailKey(activity.ID)
	cacheData := c.toCacheData(activity)

	data, err := json.Marshal(cacheData)
	if err != nil {
		return err
	}

	ttl := commonCache.RandomTTLSeconds(commonCache.DefaultTTL)
	return c.rds.SetexCtx(ctx, key, string(data), ttl)
}

// ==================== 数据转换 ====================

// toCacheData 将 model.Activity 转换为缓存数据结构
func (c *ActivityCache) toCacheData(a *model.Activity) *ActivityCacheData {
	return &ActivityCacheData{
		ID:                   a.ID,
		Title:                a.Title,
		CoverURL:             a.CoverURL,
		CoverType:            a.CoverType,
		Description:          a.Description,
		CategoryID:           a.CategoryID,
		OrganizerID:          a.OrganizerID,
		OrganizerName:        a.OrganizerName,
		OrganizerAvatar:      a.OrganizerAvatar,
		ContactPhone:         a.ContactPhone,
		RegisterStartTime:    a.RegisterStartTime,
		RegisterEndTime:      a.RegisterEndTime,
		ActivityStartTime:    a.ActivityStartTime,
		ActivityEndTime:      a.ActivityEndTime,
		Location:             a.Location,
		AddressDetail:        a.AddressDetail,
		Longitude:            a.Longitude,
		Latitude:             a.Latitude,
		MaxParticipants:      a.MaxParticipants,
		CurrentParticipants:  a.CurrentParticipants,
		RequireApproval:      a.RequireApproval,
		RequireStudentVerify: a.RequireStudentVerify,
		MinCreditScore:       a.MinCreditScore,
		Status:               a.Status,
		RejectReason:         a.RejectReason,
		ViewCount:            a.ViewCount,
		LikeCount:            a.LikeCount,
		Version:              a.Version,
		CreatedAt:            a.CreatedAt,
		UpdatedAt:            a.UpdatedAt,
	}
}

// toActivity 将缓存数据结构转换为 model.Activity
func (c *ActivityCache) toActivity(d *ActivityCacheData) *model.Activity {
	return &model.Activity{
		ID:                   d.ID,
		Title:                d.Title,
		CoverURL:             d.CoverURL,
		CoverType:            d.CoverType,
		Description:          d.Description,
		CategoryID:           d.CategoryID,
		OrganizerID:          d.OrganizerID,
		OrganizerName:        d.OrganizerName,
		OrganizerAvatar:      d.OrganizerAvatar,
		ContactPhone:         d.ContactPhone,
		RegisterStartTime:    d.RegisterStartTime,
		RegisterEndTime:      d.RegisterEndTime,
		ActivityStartTime:    d.ActivityStartTime,
		ActivityEndTime:      d.ActivityEndTime,
		Location:             d.Location,
		AddressDetail:        d.AddressDetail,
		Longitude:            d.Longitude,
		Latitude:             d.Latitude,
		MaxParticipants:      d.MaxParticipants,
		CurrentParticipants:  d.CurrentParticipants,
		RequireApproval:      d.RequireApproval,
		RequireStudentVerify: d.RequireStudentVerify,
		MinCreditScore:       d.MinCreditScore,
		Status:               d.Status,
		RejectReason:         d.RejectReason,
		ViewCount:            d.ViewCount,
		LikeCount:            d.LikeCount,
		Version:              d.Version,
		CreatedAt:            d.CreatedAt,
		UpdatedAt:            d.UpdatedAt,
	}
}

// ==================== 缓存统计（可选） ====================

// Stats 缓存统计信息
type Stats struct {
	Hits   int64         `json:"hits"`
	Misses int64         `json:"misses"`
	TTL    time.Duration `json:"ttl"`
}

// GetStats 获取缓存统计（用于监控）
func (c *ActivityCache) GetStats() *Stats {
	// MVP 阶段返回空统计，后续可接入 Prometheus
	return &Stats{
		TTL: commonCache.DefaultTTL,
	}
}
