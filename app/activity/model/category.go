package model

import (
	"context"

	"gorm.io/gorm"
)

// Category 活动分类模型
type Category struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string `gorm:"type:varchar(50);uniqueIndex;not null;comment:分类名称" json:"name"`
	Icon      string `gorm:"type:varchar(100);default:'';comment:分类图标(FontAwesome类名)" json:"icon"`
	Sort      int    `gorm:"default:0;comment:排序权重(越大越靠前)" json:"sort"`
	Status    int8   `gorm:"default:1;comment:状态: 1启用 0禁用" json:"status"`
	CreatedAt int64  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt int64  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (Category) TableName() string {
	return "categories"
}

// CategoryModel 数据访问层
type CategoryModel struct {
	db *gorm.DB
}

func NewCategoryModel(db *gorm.DB) *CategoryModel {
	return &CategoryModel{db: db}
}

// FindAll 获取所有启用的分类（按排序）
func (m *CategoryModel) FindAll(ctx context.Context) ([]Category,
	error) {
	var categories []Category
	err := m.db.WithContext(ctx).
		Where("status = ?", 1).
		Order("sort DESC, id ASC").
		Find(&categories).Error
	return categories, err
}

// FindByID 根据ID查询
func (m *CategoryModel) FindByID(ctx context.Context, id uint64) (*Category, error) {
	var category Category
	err := m.db.WithContext(ctx).
		Where("id = ? AND status = ?", id, 1).
		First(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// Exists 检查分类是否存在且启用
func (m *CategoryModel) Exists(ctx context.Context, id uint64) (bool, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&Category{}).
		Where("id = ? AND status = ?", id, 1).
		Count(&count).Error
	return count > 0, err
}
