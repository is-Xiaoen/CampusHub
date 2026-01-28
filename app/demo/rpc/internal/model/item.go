// ============================================================================
// Model 层 - Item 数据模型（简化版示例）
// ============================================================================

package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Item 数据实体
type Item struct {
	ID          int64          `gorm:"primaryKey;autoIncrement:false" json:"id"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Description string         `gorm:"size:500" json:"description"`
	Status      int32          `gorm:"default:1" json:"status"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Item) TableName() string {
	return "items"
}

// ItemModel 数据访问层
type ItemModel struct {
	db *gorm.DB
}

// NewItemModel 创建实例
func NewItemModel(db *gorm.DB) *ItemModel {
	return &ItemModel{db: db}
}

// Create 创建
func (m *ItemModel) Create(ctx context.Context, item *Item) error {
	return m.db.WithContext(ctx).Create(item).Error
}

// FindByID 根据ID查询
func (m *ItemModel) FindByID(ctx context.Context, id int64) (*Item, error) {
	var item Item
	err := m.db.WithContext(ctx).First(&item, id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// Update 更新
func (m *ItemModel) Update(ctx context.Context, item *Item) error {
	return m.db.WithContext(ctx).Save(item).Error
}

// Delete 删除
func (m *ItemModel) Delete(ctx context.Context, id int64) error {
	return m.db.WithContext(ctx).Delete(&Item{}, id).Error
}

// ListOption 列表选项
type ListOption struct {
	Page     int
	PageSize int
	Keyword  string
	Status   int32
}

// List 分页查询
func (m *ItemModel) List(ctx context.Context, opt *ListOption) ([]*Item, int64, error) {
	var items []*Item
	var total int64

	query := m.db.WithContext(ctx).Model(&Item{})

	if opt.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+opt.Keyword+"%")
	}
	if opt.Status > 0 {
		query = query.Where("status = ?", opt.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (opt.Page - 1) * opt.PageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(opt.PageSize).Find(&items).Error

	return items, total, err
}
