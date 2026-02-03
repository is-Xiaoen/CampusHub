package model

import (
	"context"

	"gorm.io/gorm"
)

// ==================== ActivityTag 活动-标签关联模型 ====================
//
// 说明：
//   - 标签元数据（名称、颜色、图标）由用户服务维护，存储在 tag_cache 表
//   - 本文件只负责活动与标签的关联关系（activity_tags 表）
//   - 标签统计数据由 activity_tag_stats 表维护

// ActivityTag 活动-标签关联表
type ActivityTag struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	ActivityID uint64 `gorm:"index:idx_activity_tag,unique;not null;comment:活动ID" json:"activity_id"`
	TagID      uint64 `gorm:"index:idx_activity_tag,unique;index;not null;comment:标签ID" json:"tag_id"`
	CreatedAt  int64  `gorm:"autoCreateTime" json:"created_at"`
}

func (ActivityTag) TableName() string {
	return "activity_tags"
}

// ==================== ActivityTagModel 数据访问层 ====================
//
// 职责：管理活动与标签的关联关系
// 注意：标签验证和查询请使用 TagCacheModel

type ActivityTagModel struct {
	db *gorm.DB
}

func NewActivityTagModel(db *gorm.DB) *ActivityTagModel {
	return &ActivityTagModel{db: db}
}

// BindToActivity 绑定标签到活动（事务内使用）
//
// 参数：
//   - tx: 事务对象
//   - activityID: 活动 ID
//   - tagIDs: 标签 ID 列表（最多 5 个）
//
// 注意：调用前应先使用 TagCacheModel.ExistsByIDs 验证标签存在性
func (m *ActivityTagModel) BindToActivity(ctx context.Context, tx *gorm.DB, activityID uint64, tagIDs []uint64) error {
	if len(tagIDs) == 0 {
		return nil
	}
	// 最多 5 个标签
	if len(tagIDs) > 5 {
		tagIDs = tagIDs[:5]
	}

	// 批量插入关联
	records := make([]ActivityTag, len(tagIDs))
	for i, tagID := range tagIDs {
		records[i] = ActivityTag{
			ActivityID: activityID,
			TagID:      tagID,
		}
	}
	return tx.WithContext(ctx).Create(&records).Error
}

// UnbindFromActivity 解绑活动的所有标签（事务内使用）
//
// 参数：
//   - tx: 事务对象
//   - activityID: 活动 ID
func (m *ActivityTagModel) UnbindFromActivity(ctx context.Context, tx *gorm.DB, activityID uint64) error {
	return tx.WithContext(ctx).
		Where("activity_id = ?", activityID).
		Delete(&ActivityTag{}).Error
}

// FindIDsByActivityID 获取活动关联的标签 ID 列表
//
// 用途：删除活动时获取关联的标签 ID，用于更新统计
func (m *ActivityTagModel) FindIDsByActivityID(ctx context.Context, activityID uint64) ([]uint64, error) {
	var ids []uint64
	err := m.db.WithContext(ctx).
		Model(&ActivityTag{}).
		Where("activity_id = ?", activityID).
		Pluck("tag_id", &ids).Error
	return ids, err
}

// CountByActivityID 获取活动关联的标签数量
func (m *ActivityTagModel) CountByActivityID(ctx context.Context, activityID uint64) (int64, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&ActivityTag{}).
		Where("activity_id = ?", activityID).
		Count(&count).Error
	return count, err
}
