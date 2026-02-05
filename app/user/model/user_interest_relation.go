/**
 * @projectName: CampusHub
 * @package: model
 * @className: UserInterestRelation
 * @author: lijunqi
 * @description: 用户兴趣标签关联实体及数据访问层
 * @date: 2026-02-02
 * @version: 1.0
 */

package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// UserInterestRelation 用户兴趣标签关联实体
type UserInterestRelation struct {
	// 主键ID
	ID int64 `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	// 用户ID
	UserID int64 `gorm:"column:user_id;not null;default:0" json:"user_id"`
	// 标签ID
	TagID int64 `gorm:"column:tag_id;not null;default:0" json:"tag_id"`
	// 创建时间
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
}

// TableName 指定表名
func (UserInterestRelation) TableName() string {
	return "user_interest_relations"
}

// IUserInterestRelationModel 用户兴趣标签关联数据访问层接口
type IUserInterestRelationModel interface {
	// Create 创建关联
	Create(ctx context.Context, relation *UserInterestRelation) error
	// Delete 删除关联
	Delete(ctx context.Context, userID, tagID int64) error
	// DeleteByUserID 删除用户所有兴趣标签
	DeleteByUserID(ctx context.Context, userID int64) error
	// ListByUserID 查询用户的兴趣标签关联
	ListByUserID(ctx context.Context, userID int64) ([]*UserInterestRelation, error)
	// ListByTagID 查询拥有某标签的用户关联
	ListByTagID(ctx context.Context, tagID int64, offset, limit int) ([]*UserInterestRelation, error)
	// Exists 检查关联是否存在
	Exists(ctx context.Context, userID, tagID int64) (bool, error)
}

// 确保 UserInterestRelationModel 实现 IUserInterestRelationModel 接口
var _ IUserInterestRelationModel = (*UserInterestRelationModel)(nil)

// UserInterestRelationModel 用户兴趣标签关联数据访问层
type UserInterestRelationModel struct {
	db *gorm.DB
}

// NewUserInterestRelationModel 创建用户兴趣标签关联Model实例
func NewUserInterestRelationModel(db *gorm.DB) IUserInterestRelationModel {
	return &UserInterestRelationModel{db: db}
}

// Create 创建关联
func (m *UserInterestRelationModel) Create(ctx context.Context, relation *UserInterestRelation) error {
	return m.db.WithContext(ctx).Create(relation).Error
}

// Delete 删除关联
func (m *UserInterestRelationModel) Delete(ctx context.Context, userID, tagID int64) error {
	return m.db.WithContext(ctx).
		Where("user_id = ? AND tag_id = ?", userID, tagID).
		Delete(&UserInterestRelation{}).Error
}

// DeleteByUserID 删除用户所有兴趣标签
func (m *UserInterestRelationModel) DeleteByUserID(ctx context.Context, userID int64) error {
	return m.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&UserInterestRelation{}).Error
}

// ListByUserID 查询用户的兴趣标签关联
func (m *UserInterestRelationModel) ListByUserID(ctx context.Context, userID int64) ([]*UserInterestRelation, error) {
	var relations []*UserInterestRelation
	err := m.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&relations).Error
	if err != nil {
		return nil, err
	}
	return relations, nil
}

// ListByTagID 查询拥有某标签的用户关联
func (m *UserInterestRelationModel) ListByTagID(ctx context.Context, tagID int64, offset, limit int) ([]*UserInterestRelation, error) {
	var relations []*UserInterestRelation
	err := m.db.WithContext(ctx).
		Where("tag_id = ?", tagID).
		Offset(offset).
		Limit(limit).
		Find(&relations).Error
	if err != nil {
		return nil, err
	}
	return relations, nil
}

// Exists 检查关联是否存在
func (m *UserInterestRelationModel) Exists(ctx context.Context, userID, tagID int64) (bool, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&UserInterestRelation{}).
		Where("user_id = ? AND tag_id = ?", userID, tagID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
