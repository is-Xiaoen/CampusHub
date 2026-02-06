package model

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ==================== ActivityTagStats 活动标签统计模型 ====================
//
// 用途：统计活动服务维度的标签使用情况
// 更新时机：活动创建/删除/标签变更时
// 数据归属：活动服务独立维护，与用户服务的 usage_count 分开

// ActivityTagStats 活动标签统计
type ActivityTagStats struct {
	TagID         uint64 `gorm:"column:tag_id;primaryKey" json:"tag_id"`                // 标签ID
	ActivityCount uint32 `gorm:"column:activity_count;default:0" json:"activity_count"` // 关联的活动数量
	ViewCount     uint64 `gorm:"column:view_count;default:0" json:"view_count"`         // 标签关联活动的总浏览量（预留）
	UpdatedAt     int64  `gorm:"column:updated_at;not null" json:"updated_at"`          // 更新时间戳
}

func (ActivityTagStats) TableName() string {
	return "activity_tag_stats"
}

// ==================== ActivityTagStatsModel 数据访问层 ====================

type ActivityTagStatsModel struct {
	db *gorm.DB
}

func NewActivityTagStatsModel(db *gorm.DB) *ActivityTagStatsModel {
	return &ActivityTagStatsModel{db: db}
}

// ==================== 更新方法 ====================

// IncrActivityCount 增加活动使用次数（原子操作）
//
// 使用 UPSERT 语义：如果记录不存在则插入，存在则更新
// 适用场景：活动创建时绑定标签
func (m *ActivityTagStatsModel) IncrActivityCount(ctx context.Context, tagIDs []uint64, delta int) error {
	if len(tagIDs) == 0 || delta == 0 {
		return nil
	}

	// 负数走 DecrActivityCount，防止 uint32(负数) 溢出
	if delta < 0 {
		return m.DecrActivityCount(ctx, tagIDs, -delta)
	}

	now := time.Now().Unix()

	// 使用 UPSERT 确保记录存在
	for _, tagID := range tagIDs {
		stats := &ActivityTagStats{
			TagID:         tagID,
			ActivityCount: uint32(delta),
			UpdatedAt:     now,
		}
		err := m.db.WithContext(ctx).
			Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "tag_id"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"activity_count": gorm.Expr("activity_count + ?", delta),
					"updated_at":     now,
				}),
			}).
			Create(stats).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// DecrActivityCount 减少活动使用次数（原子操作）
//
// 适用场景：活动删除或解绑标签时
// 注意：使用 GREATEST 确保不会变成负数
func (m *ActivityTagStatsModel) DecrActivityCount(ctx context.Context, tagIDs []uint64, delta int) error {
	if len(tagIDs) == 0 || delta == 0 {
		return nil
	}

	now := time.Now().Unix()
	return m.db.WithContext(ctx).
		Model(&ActivityTagStats{}).
		Where("tag_id IN ?", tagIDs).
		Updates(map[string]interface{}{
			"activity_count": gorm.Expr("GREATEST(activity_count - ?, 0)", delta),
			"updated_at":     now,
		}).Error
}

// BatchIncrActivityCount 批量增加活动使用次数（事务内使用）
//
// 适用场景：活动创建时一次性更新多个标签统计
func (m *ActivityTagStatsModel) BatchIncrActivityCount(ctx context.Context, tx *gorm.DB, tagIDs []uint64) error {
	if len(tagIDs) == 0 {
		return nil
	}

	now := time.Now().Unix()
	for _, tagID := range tagIDs {
		stats := &ActivityTagStats{
			TagID:         tagID,
			ActivityCount: 1,
			UpdatedAt:     now,
		}
		err := tx.WithContext(ctx).
			Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "tag_id"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"activity_count": gorm.Expr("activity_count + 1"),
					"updated_at":     now,
				}),
			}).
			Create(stats).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// BatchDecrActivityCount 批量减少活动使用次数（事务内使用）
//
// 适用场景：活动删除时一次性更新多个标签统计
func (m *ActivityTagStatsModel) BatchDecrActivityCount(ctx context.Context, tx *gorm.DB, tagIDs []uint64) error {
	if len(tagIDs) == 0 {
		return nil
	}

	now := time.Now().Unix()
	return tx.WithContext(ctx).
		Model(&ActivityTagStats{}).
		Where("tag_id IN ?", tagIDs).
		Updates(map[string]interface{}{
			"activity_count": gorm.Expr("GREATEST(activity_count - 1, 0)"),
			"updated_at":     now,
		}).Error
}

// ==================== 查询方法 ====================

// FindByTagID 查询单个标签统计
func (m *ActivityTagStatsModel) FindByTagID(ctx context.Context, tagID uint64) (*ActivityTagStats, error) {
	var stats ActivityTagStats
	err := m.db.WithContext(ctx).
		Where("tag_id = ?", tagID).
		First(&stats).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// FindByTagIDs 批量查询标签统计
func (m *ActivityTagStatsModel) FindByTagIDs(ctx context.Context, tagIDs []uint64) ([]ActivityTagStats, error) {
	if len(tagIDs) == 0 {
		return []ActivityTagStats{}, nil
	}

	var statsList []ActivityTagStats
	err := m.db.WithContext(ctx).
		Where("tag_id IN ?", tagIDs).
		Find(&statsList).Error
	return statsList, err
}

// GetTopTags 获取使用次数最多的标签 ID 列表
//
// 返回值：按 activity_count 降序排列的标签 ID 列表
func (m *ActivityTagStatsModel) GetTopTags(ctx context.Context, limit int) ([]uint64, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	var tagIDs []uint64
	err := m.db.WithContext(ctx).
		Model(&ActivityTagStats{}).
		Where("activity_count > 0").
		Order("activity_count DESC").
		Limit(limit).
		Pluck("tag_id", &tagIDs).Error
	return tagIDs, err
}

// ==================== 维护方法 ====================

// RecalculateStats 重新计算标签统计（用于数据修复）
//
// 根据 activity_tags 表重新计算每个标签的活动数量
func (m *ActivityTagStatsModel) RecalculateStats(ctx context.Context) error {
	now := time.Now().Unix()

	// 使用原生 SQL 进行批量更新
	sql := `
		INSERT INTO activity_tag_stats (tag_id, activity_count, view_count, updated_at)
		SELECT
			at.tag_id,
			COUNT(DISTINCT at.activity_id) as activity_count,
			0 as view_count,
			? as updated_at
		FROM activity_tags at
		INNER JOIN activities a ON a.id = at.activity_id AND a.deleted_at IS NULL
		GROUP BY at.tag_id
		ON DUPLICATE KEY UPDATE
			activity_count = VALUES(activity_count),
			updated_at = VALUES(updated_at)
	`
	return m.db.WithContext(ctx).Exec(sql, now).Error
}

// CleanupOrphanStats 清理孤儿统计记录（标签已被删除但统计还在）
func (m *ActivityTagStatsModel) CleanupOrphanStats(ctx context.Context) error {
	sql := `
		DELETE ats FROM activity_tag_stats ats
		LEFT JOIN tag_cache tc ON ats.tag_id = tc.id
		WHERE tc.id IS NULL
	`
	return m.db.WithContext(ctx).Exec(sql).Error
}
