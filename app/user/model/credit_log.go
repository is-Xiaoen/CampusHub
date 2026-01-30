/**
 * @projectName: CampusHub
 * @package: model
 * @className: CreditLog
 * @author: lijunqi
 * @description: 信用变更记录实体及数据访问层
 * @date: 2026-01-30
 * @version: 1.0
 */

package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// CreditChangeType 信用变更类型
const (
	// CreditChangeTypeAdd 加分
	CreditChangeTypeAdd int8 = 1
	// CreditChangeTypeDeduct 扣分
	CreditChangeTypeDeduct int8 = 2
)

// CreditLog 信用变更记录实体
type CreditLog struct {
	// 主键ID（数据库自增）
	ID int64 `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	// 用户ID
	UserID int64 `gorm:"index:idx_user_id;column:user_id;not null" json:"user_id"`
	// 变更类型：1加分 2扣分
	ChangeType int8 `gorm:"column:change_type;not null" json:"change_type"`
	// 来源ID（幂等键，如：activity_123_complete）
	SourceID string `gorm:"uniqueIndex:uk_source_id;column:source_id;size:128;not null" json:"source_id"`
	// 变更前分数
	BeforeScore int `gorm:"column:before_score;not null" json:"before_score"`
	// 变更后分数
	AfterScore int `gorm:"column:after_score;not null" json:"after_score"`
	// 变更值（正数加分，负数扣分）
	Delta int `gorm:"column:delta;not null" json:"delta"`
	// 变更原因
	Reason string `gorm:"column:reason;size:255" json:"reason"`
	// 创建时间
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName 指定表名
func (CreditLog) TableName() string {
	return "credit_logs"
}

// ICreditLogModel 信用变更记录数据访问层接口
type ICreditLogModel interface {
	// Create 创建信用变更记录
	Create(ctx context.Context, log *CreditLog) error
	// FindByID 根据主键ID查询
	FindByID(ctx context.Context, id int64) (*CreditLog, error)
	// FindBySourceID 根据来源ID查询（用于幂等检查）
	FindBySourceID(ctx context.Context, sourceID string) (*CreditLog, error)
	// ExistsBySourceID 检查来源ID是否已存在（幂等检查）
	ExistsBySourceID(ctx context.Context, sourceID string) (bool, error)
	// ListByUserID 查询用户的信用变更记录列表
	ListByUserID(ctx context.Context, userID int64, offset, limit int) ([]*CreditLog, error)
	// CountByUserID 统计用户的信用变更记录数量
	CountByUserID(ctx context.Context, userID int64) (int64, error)
}

// 确保 CreditLogModel 实现 ICreditLogModel 接口
// 确保如果不对，可以在编译阶段就报错
var _ ICreditLogModel = (*CreditLogModel)(nil)

// CreditLogModel 信用变更记录数据访问层
type CreditLogModel struct {
	db *gorm.DB
}

// NewCreditLogModel 创建信用变更记录Model实例
func NewCreditLogModel(db *gorm.DB) ICreditLogModel {
	return &CreditLogModel{db: db}
}

// Create 创建信用变更记录
func (m *CreditLogModel) Create(ctx context.Context, log *CreditLog) error {
	return m.db.WithContext(ctx).Create(log).Error
}

// FindByID 根据主键ID查询
func (m *CreditLogModel) FindByID(ctx context.Context, id int64) (*CreditLog, error) {
	var log CreditLog
	err := m.db.WithContext(ctx).First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// FindBySourceID 根据来源ID查询（用于幂等检查）
func (m *CreditLogModel) FindBySourceID(ctx context.Context, sourceID string) (*CreditLog, error) {
	var log CreditLog
	err := m.db.WithContext(ctx).Where("source_id = ?", sourceID).First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// ExistsBySourceID 检查来源ID是否已存在（幂等检查）
func (m *CreditLogModel) ExistsBySourceID(
	ctx context.Context,
	sourceID string,
) (bool, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&CreditLog{}).
		Where("source_id = ?", sourceID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListByUserID 查询用户的信用变更记录列表
func (m *CreditLogModel) ListByUserID(
	ctx context.Context,
	userID int64,
	offset, limit int,
) ([]*CreditLog, error) {
	var logs []*CreditLog
	err := m.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

// CountByUserID 统计用户的信用变更记录数量
func (m *CreditLogModel) CountByUserID(
	ctx context.Context,
	userID int64,
) (int64, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&CreditLog{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}
