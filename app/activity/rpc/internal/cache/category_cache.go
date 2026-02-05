package cache

import (
	"context"
	"encoding/json"
	"errors"

	"activity-platform/app/activity/model"
	commonCache "activity-platform/common/cache"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"
)

// ==================== CategoryCache 分类列表缓存 ====================
//
// 功能说明：
//   - 缓存所有启用的活动分类
//   - 分类数据变化极少，使用较长 TTL
//
// 缓存策略：
//   - Key: activity:category:list
//   - TTL: 30min
//   - 失效时机: 分类增删改时主动删除（MVP 暂不实现，通过 TTL 自动过期）

// CategoryCache 分类缓存服务
type CategoryCache struct {
	rds *redis.Redis
	db  *gorm.DB
}

// NewCategoryCache 创建分类缓存服务
func NewCategoryCache(rds *redis.Redis, db *gorm.DB) *CategoryCache {
	return &CategoryCache{
		rds: rds,
		db:  db,
	}
}

// CategoryCacheData 分类缓存数据结构
type CategoryCacheData struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
	Sort int    `json:"sort"`
}

// GetList 获取分类列表（带缓存）
//
// 流程：
//  1. 查询 Redis 缓存
//  2. 缓存命中：反序列化返回
//  3. 缓存未命中：查询 DB，写入缓存
//
// 返回值：
//   - []model.Category: 分类列表（按 sort DESC, id ASC 排序）
//   - error: 错误信息
func (c *CategoryCache) GetList(ctx context.Context) ([]model.Category, error) {
	key := commonCache.CategoryListKey()

	// 1. 尝试从缓存获取
	val, err := c.rds.GetCtx(ctx, key)
	if err != nil && !errors.Is(err, redis.Nil) {
		// Redis 错误，降级查询 DB
		logx.WithContext(ctx).Errorf("[CategoryCache] Redis 错误，降级查 DB: err=%v", err)
		return c.getFromDB(ctx)
	}

	// 2. 缓存命中
	if val != "" {
		var cacheList []CategoryCacheData
		if err := json.Unmarshal([]byte(val), &cacheList); err != nil {
			logx.WithContext(ctx).Errorf("[CategoryCache] 反序列化失败: err=%v", err)
			_, _ = c.rds.DelCtx(ctx, key)
			return c.getFromDBAndCache(ctx, key)
		}

		return c.toCategories(cacheList), nil
	}

	// 3. 缓存未命中，查询 DB
	return c.getFromDBAndCache(ctx, key)
}

// getFromDB 直接从数据库查询
func (c *CategoryCache) getFromDB(ctx context.Context) ([]model.Category, error) {
	var categories []model.Category
	err := c.db.WithContext(ctx).
		Where("status = ?", 1).
		Order("sort DESC, id ASC").
		Find(&categories).Error
	return categories, err
}

// getFromDBAndCache 从 DB 查询并写入缓存
func (c *CategoryCache) getFromDBAndCache(ctx context.Context, key string) ([]model.Category, error) {
	categories, err := c.getFromDB(ctx)
	if err != nil {
		return nil, err
	}

	// 转换并序列化
	cacheList := c.toCacheDataList(categories)
	data, err := json.Marshal(cacheList)
	if err != nil {
		logx.WithContext(ctx).Errorf("[CategoryCache] 序列化失败: err=%v", err)
		return categories, nil
	}

	// 写入缓存（分类变化少，使用较长 TTL）
	ttl := commonCache.RandomTTLSeconds(commonCache.LongTTL)
	if err := c.rds.SetexCtx(ctx, key, string(data), ttl); err != nil {
		logx.WithContext(ctx).Errorf("[CategoryCache] 写入缓存失败: err=%v", err)
	}

	return categories, nil
}

// Invalidate 删除分类缓存
//
// 调用时机：
//   - 新增/修改/删除分类后
func (c *CategoryCache) Invalidate(ctx context.Context) error {
	key := commonCache.CategoryListKey()
	if _, err := c.rds.DelCtx(ctx, key); err != nil {
		logx.WithContext(ctx).Errorf("[CategoryCache] 删除缓存失败: err=%v", err)
		return err
	}
	return nil
}

// Warmup 预热分类缓存
//
// 调用时机：服务启动时
func (c *CategoryCache) Warmup(ctx context.Context) error {
	key := commonCache.CategoryListKey()
	_, err := c.getFromDBAndCache(ctx, key)
	return err
}

// ==================== 数据转换 ====================

func (c *CategoryCache) toCacheDataList(categories []model.Category) []CategoryCacheData {
	result := make([]CategoryCacheData, len(categories))
	for i, cat := range categories {
		result[i] = CategoryCacheData{
			ID:   cat.ID,
			Name: cat.Name,
			Icon: cat.Icon,
			Sort: cat.Sort,
		}
	}
	return result
}

func (c *CategoryCache) toCategories(cacheList []CategoryCacheData) []model.Category {
	result := make([]model.Category, len(cacheList))
	for i, d := range cacheList {
		result[i] = model.Category{
			ID:   d.ID,
			Name: d.Name,
			Icon: d.Icon,
			Sort: d.Sort,
		}
	}
	return result
}

// GetByID 根据 ID 获取分类（从列表缓存中查找）
//
// 说明：
//   - 复用列表缓存，避免单独维护分类详情缓存
//   - 分类数量少（通常 < 20），遍历查找可接受
func (c *CategoryCache) GetByID(ctx context.Context, id uint64) (*model.Category, error) {
	categories, err := c.GetList(ctx)
	if err != nil {
		return nil, err
	}

	for i := range categories {
		if categories[i].ID == id {
			return &categories[i], nil
		}
	}

	return nil, gorm.ErrRecordNotFound
}

// GetNameMap 获取分类名称映射表
//
// 返回值：map[分类ID]分类名称
// 用途：批量查询活动时快速获取分类名称
func (c *CategoryCache) GetNameMap(ctx context.Context) (map[uint64]string, error) {
	categories, err := c.GetList(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[uint64]string, len(categories))
	for _, cat := range categories {
		result[cat.ID] = cat.Name
	}
	return result, nil
}
