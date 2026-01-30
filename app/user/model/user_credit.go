/**
 * @projectName: CampusHub
 * @package: model
 * @className: UserCredit
 * @author: lijunqi
 * @description: 用户信用分实体及数据访问层
 * @date: 2026-01-30
 * @version: 1.0
 */

package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// UserCredit 用户信用分实体
type UserCredit struct {
	// 主键ID（数据库自增）
	ID int64 `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	// 用户ID
	UserID int64 `gorm:"uniqueIndex:uk_user_id;column:user_id;not null" json:"user_id"`
	// 信用分数（0-150）
	Score int `gorm:"column:score;not null;default:100" json:"score"`
	// 信用等级：1差 2较差 3一般 4良好 5优秀
	Level int8 `gorm:"column:level;not null;default:4" json:"level"`
	// 创建时间
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	// 更新时间
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (UserCredit) TableName() string {
	return "user_credits"
}

// IUserCreditModel 用户信用分数据访问层接口
type IUserCreditModel interface {
	// Create 创建信用记录
	Create(ctx context.Context, credit *UserCredit) error
	// FindByID 根据主键ID查询
	FindByID(ctx context.Context, id int64) (*UserCredit, error)
	// FindByUserID 根据用户ID查询信用信息
	FindByUserID(ctx context.Context, userID int64) (*UserCredit, error)
	// Update 更新信用信息
	Update(ctx context.Context, credit *UserCredit) error
	// UpdateScore 更新用户信用分数和等级
	UpdateScore(ctx context.Context, userID int64, score int, level int8) error
	// ExistsByUserID 检查用户信用记录是否存在
	ExistsByUserID(ctx context.Context, userID int64) (bool, error)
}

// 确保 UserCreditModel 实现 IUserCreditModel 接口
// 确保如果不对，可以在编译阶段就报错
var _ IUserCreditModel = (*UserCreditModel)(nil)

// UserCreditModel 用户信用分数据访问层
type UserCreditModel struct {
	db *gorm.DB
}

// NewUserCreditModel 创建用户信用分Model实例
func NewUserCreditModel(db *gorm.DB) IUserCreditModel {
	return &UserCreditModel{db: db}
}

// Create 创建信用记录
func (m *UserCreditModel) Create(ctx context.Context, credit *UserCredit) error {
	return m.db.WithContext(ctx).Create(credit).Error
}

// FindByID 根据主键ID查询
func (m *UserCreditModel) FindByID(ctx context.Context, id int64) (*UserCredit, error) {
	var credit UserCredit
	err := m.db.WithContext(ctx).First(&credit, id).Error
	if err != nil {
		return nil, err
	}
	return &credit, nil
}

// FindByUserID 根据用户ID查询信用信息
func (m *UserCreditModel) FindByUserID(ctx context.Context, userID int64) (*UserCredit, error) {
	var credit UserCredit
	err := m.db.WithContext(ctx).Where("user_id = ?", userID).First(&credit).Error
	if err != nil {
		return nil, err
	}
	return &credit, nil
}

// Update 更新信用信息
func (m *UserCreditModel) Update(ctx context.Context, credit *UserCredit) error {
	return m.db.WithContext(ctx).Save(credit).Error
}

// UpdateScore 更新用户信用分数和等级
func (m *UserCreditModel) UpdateScore(ctx context.Context, userID int64, score int, level int8) error {
	return m.db.WithContext(ctx).
		Model(&UserCredit{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"score": score,
			"level": level,
		}).Error
}

// ExistsByUserID 检查用户信用记录是否存在
func (m *UserCreditModel) ExistsByUserID(ctx context.Context, userID int64) (bool, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&UserCredit{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
