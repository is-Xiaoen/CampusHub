package model

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ==================== TagCache 标签缓存模型 ====================
//
// 数据来源：用户服务 interest_tag 表
// 同步策略：MQ 实时同步 + 定时全量兜底
// 使用场景：活动服务本地查询标签信息，避免跨服务 RPC 调用

// TagCache 标签缓存（从用户服务同步）
type TagCache struct {
	ID          uint64 `gorm:"column:id;primaryKey" json:"id"`                              // 标签ID（与用户服务一致）
	Name        string `gorm:"column:name;type:varchar(50);not null" json:"name"`           // 标签名称
	Color       string `gorm:"column:color;type:varchar(20);not null" json:"color"`         // 标签颜色
	Icon        string `gorm:"column:icon;type:varchar(255);not null" json:"icon"`          // 标签图标URL
	Status      int8   `gorm:"column:status;default:1" json:"status"`                       // 状态: 1启用 0禁用
	Description string `gorm:"column:description;type:varchar(200);not null" json:"description"` // 标签描述
	SyncedAt    int64  `gorm:"column:synced_at;not null" json:"synced_at"`                  // 最后同步时间戳
	CreatedAt   int64  `gorm:"column:created_at;not null" json:"created_at"`                // 原始创建时间
	UpdatedAt   int64  `gorm:"column:updated_at;not null" json:"updated_at"`                // 原始更新时间
}

func (TagCache) TableName() string {
	return "tag_cache"
}

// ==================== TagCacheModel 数据访问层 ====================

type TagCacheModel struct {
	db *gorm.DB
}

func NewTagCacheModel(db *gorm.DB) *TagCacheModel {
	return &TagCacheModel{db: db}
}

// ==================== 查询方法 ====================

// FindAll 获取所有启用的标签
func (m *TagCacheModel) FindAll(ctx context.Context) ([]TagCache, error) {
	var tags []TagCache
	err := m.db.WithContext(ctx).
		Where("status = ?", 1).
		Order("id ASC").
		Find(&tags).Error
	return tags, err
}

// FindHot 获取热门标签（需要结合 activity_tag_stats）
// 如果没有统计数据，按 ID 排序返回前 N 个
func (m *TagCacheModel) FindHot(ctx context.Context, limit int) ([]TagCache, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	var tags []TagCache
	// 关联统计表获取热门标签
	err := m.db.WithContext(ctx).
		Table("tag_cache tc").
		Select("tc.*").
		Joins("LEFT JOIN activity_tag_stats ats ON tc.id = ats.tag_id").
		Where("tc.status = ?", 1).
		Order("COALESCE(ats.activity_count, 0) DESC, tc.id ASC").
		Limit(limit).
		Find(&tags).Error
	return tags, err
}

// FindByID 根据 ID 查询单个标签
func (m *TagCacheModel) FindByID(ctx context.Context, id uint64) (*TagCache, error) {
	var tag TagCache
	err := m.db.WithContext(ctx).
		Where("id = ? AND status = ?", id, 1).
		First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// FindByIDs 根据 ID 列表查询
func (m *TagCacheModel) FindByIDs(ctx context.Context, ids []uint64) ([]TagCache, error) {
	if len(ids) == 0 {
		return []TagCache{}, nil
	}
	var tags []TagCache
	err := m.db.WithContext(ctx).
		Where("id IN ? AND status = ?", ids, 1).
		Find(&tags).Error
	return tags, err
}

// Exists 检查标签是否存在且启用
func (m *TagCacheModel) Exists(ctx context.Context, id uint64) (bool, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&TagCache{}).
		Where("id = ? AND status = ?", id, 1).
		Count(&count).Error
	return count > 0, err
}

// ExistsByIDs 批量检查标签是否存在
// 返回存在的标签 ID 列表和不存在的标签 ID 列表
func (m *TagCacheModel) ExistsByIDs(ctx context.Context, ids []uint64) (existIDs, invalidIDs []uint64, err error) {
	if len(ids) == 0 {
		return []uint64{}, []uint64{}, nil
	}

	var existingIDs []uint64
	err = m.db.WithContext(ctx).
		Model(&TagCache{}).
		Where("id IN ? AND status = ?", ids, 1).
		Pluck("id", &existingIDs).Error
	if err != nil {
		return nil, nil, err
	}

	// 构建存在的 ID 集合
	existSet := make(map[uint64]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		existSet[id] = struct{}{}
	}

	// 分离存在和不存在的 ID
	for _, id := range ids {
		if _, ok := existSet[id]; ok {
			existIDs = append(existIDs, id)
		} else {
			invalidIDs = append(invalidIDs, id)
		}
	}

	return existIDs, invalidIDs, nil
}

// FindByActivityID 获取活动的所有标签（关联查询）
func (m *TagCacheModel) FindByActivityID(ctx context.Context, activityID uint64) ([]TagCache, error) {
	var tags []TagCache
	err := m.db.WithContext(ctx).
		Table("tag_cache tc").
		Select("tc.*").
		Joins("INNER JOIN activity_tags at ON tc.id = at.tag_id").
		Where("at.activity_id = ? AND tc.status = ?", activityID, 1).
		Find(&tags).Error
	return tags, err
}

// ActivityTagCacheInfo 活动标签关联信息（用于批量查询）
type ActivityTagCacheInfo struct {
	ActivityID uint64
	TagCache
}

// FindByActivityIDs 批量获取多个活动的标签（避免 N+1 查询问题）
//
// 返回值：map[活动ID][]TagCache
//
// SQL 示例：
//
//	SELECT at.activity_id, tc.*
//	FROM activity_tags at
//	INNER JOIN tag_cache tc ON tc.id = at.tag_id
//	WHERE at.activity_id IN (1, 2, 3) AND tc.status = 1
func (m *TagCacheModel) FindByActivityIDs(ctx context.Context, activityIDs []uint64) (map[uint64][]TagCache, error) {
	result := make(map[uint64][]TagCache)
	if len(activityIDs) == 0 {
		return result, nil
	}

	var infos []ActivityTagCacheInfo
	err := m.db.WithContext(ctx).
		Table("activity_tags at").
		Select("at.activity_id, tc.id, tc.name, tc.color, tc.icon, tc.status, tc.description").
		Joins("INNER JOIN tag_cache tc ON tc.id = at.tag_id").
		Where("at.activity_id IN ? AND tc.status = ?", activityIDs, 1).
		Scan(&infos).Error
	if err != nil {
		return nil, err
	}

	// 按活动 ID 分组
	for _, info := range infos {
		result[info.ActivityID] = append(result[info.ActivityID], info.TagCache)
	}

	return result, nil
}

// ==================== 同步方法（用于数据同步） ====================

// Upsert 更新或插入单个标签（用于 MQ 消费）
func (m *TagCacheModel) Upsert(ctx context.Context, tag *TagCache) error {
	tag.SyncedAt = time.Now().Unix()
	return m.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "color", "icon", "status", "description", "synced_at", "updated_at"}),
		}).
		Create(tag).Error
}

// UpsertBatch 批量更新或插入标签（用于定时同步）
func (m *TagCacheModel) UpsertBatch(ctx context.Context, tags []TagCache) error {
	if len(tags) == 0 {
		return nil
	}

	now := time.Now().Unix()
	for i := range tags {
		tags[i].SyncedAt = now
	}

	return m.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "color", "icon", "status", "description", "synced_at", "updated_at"}),
		}).
		CreateInBatches(tags, 100).Error
}

// DeleteByID 删除单个标签（用于 MQ 消费，标签被删除时）
func (m *TagCacheModel) DeleteByID(ctx context.Context, id uint64) error {
	return m.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&TagCache{}).Error
}

// GetLastSyncTime 获取最后同步时间
func (m *TagCacheModel) GetLastSyncTime(ctx context.Context) (int64, error) {
	var maxSyncedAt int64
	err := m.db.WithContext(ctx).
		Model(&TagCache{}).
		Select("COALESCE(MAX(synced_at), 0)").
		Scan(&maxSyncedAt).Error
	return maxSyncedAt, err
}

// Count 获取缓存标签总数
func (m *TagCacheModel) Count(ctx context.Context) (int64, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&TagCache{}).
		Count(&count).Error
	return count, err
}
