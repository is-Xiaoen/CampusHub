/**
 * @projectName: CampusHub
 * @package: model
 * @className: InterestTag
 * @author: lijunqi
 * @description: 兴趣标签实体及数据访问层
 * @date: 2026-02-02
 * @version: 1.0
 */

package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// TagStatus 标签状态
const (
	// TagStatusDisabled 禁用
	TagStatusDisabled uint64 = 0
	// TagStatusNormal 正常
	TagStatusNormal uint64 = 1
)

// InterestTag 兴趣标签实体
type InterestTag struct {
	// 标签主键ID
	TagID uint64 `gorm:"primaryKey;autoIncrement;column:tag_id" json:"tag_id"`
	// 标签名称
	TagName string `gorm:"uniqueIndex:uk_tag_name;column:tag_name;size:50;not null" json:"tag_name"`
	// 标签颜色值
	Color string `gorm:"column:color;size:20;default:''" json:"color"`
	// 标签图标线上URL地址
	Icon string `gorm:"column:icon;size:255;default:''" json:"icon"`
	// 标签被用户使用的总次数
	UsageCount uint64 `gorm:"column:usage_count;default:0" json:"usage_count"`
	// 标签状态：0-禁用，1-正常
	Status uint64 `gorm:"column:status;default:1" json:"status"`
	// 标签描述
	TagDesc string `gorm:"column:tag_desc;size:200;default:''" json:"tag_desc"`
	// 标签创建时间
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	// 标签更新时间
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
}

// TableName 指定表名
func (InterestTag) TableName() string {
	return "interest_tags"
}

// IInterestTagModel 兴趣标签数据访问层接口
type IInterestTagModel interface {
	// Create 创建标签
	Create(ctx context.Context, tag *InterestTag) error
	// FindByID 根据标签ID查询
	FindByID(ctx context.Context, tagID int64) (*InterestTag, error)
	// FindByName 根据标签名称查询
	FindByName(ctx context.Context, tagName string) (*InterestTag, error)
	// List 查询标签列表
	List(ctx context.Context, offset, limit int) ([]*InterestTag, error)
	// ListAll 查询所有可用标签
	ListAll(ctx context.Context) ([]*InterestTag, error)
	// Update 更新标签
	Update(ctx context.Context, tag *InterestTag) error
	// IncrementUsageCount 增加使用次数
	IncrementUsageCount(ctx context.Context, tagID int64, delta int) error
}

// 确保 InterestTagModel 实现 IInterestTagModel 接口
var _ IInterestTagModel = (*InterestTagModel)(nil)

// InterestTagModel 兴趣标签数据访问层
type InterestTagModel struct {
	db *gorm.DB
}

// NewInterestTagModel 创建兴趣标签Model实例
func NewInterestTagModel(db *gorm.DB) IInterestTagModel {
	return &InterestTagModel{db: db}
}

// Create 创建标签
func (m *InterestTagModel) Create(ctx context.Context, tag *InterestTag) error {
	return m.db.WithContext(ctx).Create(tag).Error
}

// FindByID 根据标签ID查询
func (m *InterestTagModel) FindByID(ctx context.Context, tagID int64) (*InterestTag, error) {
	var tag InterestTag
	err := m.db.WithContext(ctx).First(&tag, tagID).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// FindByName 根据标签名称查询
func (m *InterestTagModel) FindByName(ctx context.Context, tagName string) (*InterestTag, error) {
	var tag InterestTag
	err := m.db.WithContext(ctx).Where("tag_name = ?", tagName).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// List 查询标签列表
func (m *InterestTagModel) List(ctx context.Context, offset, limit int) ([]*InterestTag, error) {
	var tags []*InterestTag
	err := m.db.WithContext(ctx).
		Order("usage_count DESC").
		Offset(offset).
		Limit(limit).
		Find(&tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// ListAll 查询所有可用标签
func (m *InterestTagModel) ListAll(ctx context.Context) ([]*InterestTag, error) {
	var tags []*InterestTag
	err := m.db.WithContext(ctx).
		Where("status = ?", TagStatusNormal).
		Order("usage_count DESC").
		Find(&tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// Update 更新标签
func (m *InterestTagModel) Update(ctx context.Context, tag *InterestTag) error {
	return m.db.WithContext(ctx).Save(tag).Error
}

// IncrementUsageCount 增加使用次数
func (m *InterestTagModel) IncrementUsageCount(ctx context.Context, tagID int64, delta int) error {
	return m.db.WithContext(ctx).
		Model(&InterestTag{}).
		Where("tag_id = ?", tagID).
		UpdateColumn("usage_count", gorm.Expr("usage_count + ?", delta)).Error
}
