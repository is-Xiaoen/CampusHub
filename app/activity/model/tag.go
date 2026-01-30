package model

import (
	"context"

	"gorm.io/gorm"
)

// Tag 标签模型
type Tag struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	Name       string `gorm:"type:varchar(30);uniqueIndex;not null;comment:标签名称" json:"name"`
	Color      string `gorm:"type:varchar(20);default:'orange';comment:标签颜色主题"    json:"color"`
	Icon       string `gorm:"type:varchar(50);default:'';comment:标签图标" json:"icon"`
	UsageCount uint32 `gorm:"default:0;comment:使用次数"  json:"usage_count"`
	Status     int8   `gorm:"default:1;comment:状态: 1启用 0禁用" json:"status"`
	CreatedAt  int64  `gorm:"autoCreateTime" json:"created_at"`
}

func (Tag) TableName() string {
	return "tags"
}

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

// ==================== TagModel 数据访问层 ====================

type TagModel struct {
	db *gorm.DB
}

func NewTagModel(db *gorm.DB) *TagModel {
	return &TagModel{db: db}
}

// FindAll 获取所有启用的标签
func (m *TagModel) FindAll(ctx context.Context) ([]Tag, error) {
	var tags []Tag
	err := m.db.WithContext(ctx).
		Where("status = ?", 1).
		Order("usage_count DESC, id ASC").
		Find(&tags).Error
	return tags, err
}

// FindHot 获取热门标签（按使用次数）
func (m *TagModel) FindHot(ctx context.Context, limit int) ([]Tag, error) {
	if limit <= 0 {
		limit = 10
	}
	var tags []Tag
	err := m.db.WithContext(ctx).
		Where("status = ?", 1).
		Order("usage_count DESC").
		Limit(limit).
		Find(&tags).Error
	return tags, err
}

// FindByIDs 根据ID列表查询
func (m *TagModel) FindByIDs(ctx context.Context, ids []uint64) ([]Tag, error) {
	if len(ids) == 0 {
		return []Tag{}, nil
	}
	var tags []Tag
	err := m.db.WithContext(ctx).
		Where("id IN ? AND status = ?", ids, 1).
		Find(&tags).Error
	return tags, err
}

// IncrUsageCount 增加使用次数（原子操作）
func (m *TagModel) IncrUsageCount(ctx context.Context, ids []uint64, delta int) error {
	if len(ids) == 0 {
		return nil
	}
	return m.db.WithContext(ctx).
		Model(&Tag{}).
		Where("id IN ?", ids).
		Update("usage_count", gorm.Expr("usage_count + ?",
			delta)).Error
}

// FindByActivityID 获取活动的所有标签
func (m *TagModel) FindByActivityID(ctx context.Context,
	activityID uint64) ([]Tag, error) {
	var tags []Tag
	err := m.db.WithContext(ctx).
		Table("tags t").
		Select("t.*").
		Joins("INNER JOIN activity_tags at ON t.id = at.tag_id").
		Where("at.activity_id = ? AND t.status = ?", activityID,
			1).
		Find(&tags).Error
	return tags, err
}

// BindToActivity 绑定标签到活动（事务内使用）
func (m *TagModel) BindToActivity(ctx context.Context, tx *gorm.DB, activityID uint64, tagIDs []uint64) error {
	if len(tagIDs) == 0 {
		return nil
	}
	// 最多5个标签
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
func (m *TagModel) UnbindFromActivity(ctx context.Context, tx *gorm.DB, activityID uint64) error {
	return tx.WithContext(ctx).
		Where("activity_id = ?", activityID).
		Delete(&ActivityTag{}).Error
}
