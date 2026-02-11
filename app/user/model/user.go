/**
 * @projectName: CampusHub
 * @package: model
 * @className: User
 * @author: lijunqi
 * @description: 用户基础信息实体及数据访问层
 * @date: 2026-02-02
 * @version: 1.0
 */

package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// UserStatus 用户状态
const (
	// UserStatusDisabled 禁用
	UserStatusDisabled int64 = 0
	// UserStatusNormal 正常
	UserStatusNormal int64 = 1
	// UserStatusDeleted 注销
	UserStatusDeleted int64 = 2
)

// UserGender 用户性别
const (
	// UserGenderUnknown 未知
	UserGenderUnknown int64 = 0
	// UserGenderMale 男
	UserGenderMale int64 = 1
	// UserGenderFemale 女
	UserGenderFemale int64 = 2
)

// User 用户基础信息实体
type User struct {
	// 用户主键ID
	UserID int64 `gorm:"primaryKey;autoIncrement;column:user_id" json:"user_id"`
	// QQ邮箱（用户登录/标识用）
	QQEmail string `gorm:"uniqueIndex:uk_qqemail;column:QQemail;size:100;not null" json:"qq_email"`
	// 用户昵称
	Nickname string `gorm:"column:nickname;size:50;not null" json:"nickname"`
	// 头像图片ID（关联SysImage）
	AvatarID int64 `gorm:"column:avatar_id;default:0" json:"avatar_id"`
	// 用户个人简介
	Introduction string `gorm:"column:introduction;size:500;default:''" json:"introduction"`
	// 用户状态：0-禁用，1-正常，2-注销
	Status int64 `gorm:"column:status;not null;default:1" json:"status"`
	// 用户密码
	Password string `gorm:"column:password;size:255;not null" json:"-"`
	// 性别：0-未知，1-男，2-女
	Gender int64 `gorm:"column:gender;default:0" json:"gender"`
	// 用户年龄
	Age int64 `gorm:"column:age;default:0" json:"age"`
	// 用户创建时间
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	// 用户信息更新时间
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// IUserModel 用户数据访问层接口
type IUserModel interface {
	// Create 创建用户
	Create(ctx context.Context, user *User) error
	// FindByUserID 根据用户ID查询
	FindByUserID(ctx context.Context, userID int64) (*User, error)
	// FindByQQEmail 根据QQ邮箱查询
	FindByQQEmail(ctx context.Context, qqEmail string) (*User, error)
	// Update 更新用户信息
	Update(ctx context.Context, user *User) error
	// UpdatePassword 更新密码
	UpdatePassword(ctx context.Context, userID int64, password string) error
	// ExistsByQQEmail 检查QQ邮箱是否存在
	ExistsByQQEmail(ctx context.Context, qqEmail string) (bool, error)
	// FindByIDs 根据ID列表查询
	FindByIDs(ctx context.Context, ids []int64) ([]*User, error)
}

// 确保 UserModel 实现 IUserModel 接口
var _ IUserModel = (*UserModel)(nil)

// UserModel 用户数据访问层
type UserModel struct {
	db *gorm.DB
}

// NewUserModel 创建用户Model实例
func NewUserModel(db *gorm.DB) IUserModel {
	return &UserModel{db: db}
}

// Create 创建用户
func (m *UserModel) Create(ctx context.Context, user *User) error {
	return m.db.WithContext(ctx).Create(user).Error
}

// FindByUserID 根据用户ID查询
func (m *UserModel) FindByUserID(ctx context.Context, userID int64) (*User, error) {
	var user User
	err := m.db.WithContext(ctx).First(&user, userID).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByQQEmail 根据QQ邮箱查询
func (m *UserModel) FindByQQEmail(ctx context.Context, qqEmail string) (*User, error) {
	var user User
	err := m.db.WithContext(ctx).Where("QQemail = ?", qqEmail).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update 更新用户信息
func (m *UserModel) Update(ctx context.Context, user *User) error {
	return m.db.WithContext(ctx).Save(user).Error
}

// UpdatePassword 更新密码
func (m *UserModel) UpdatePassword(ctx context.Context, userID int64, password string) error {
	return m.db.WithContext(ctx).
		Model(&User{}).
		Where("user_id = ?", userID).
		Update("password", password).Error
}

// ExistsByQQEmail 检查QQ邮箱是否存在
func (m *UserModel) ExistsByQQEmail(ctx context.Context, qqEmail string) (bool, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&User{}).
		Where("QQemail = ?", qqEmail).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// FindByIDs 根据ID列表查询
func (m *UserModel) FindByIDs(ctx context.Context, ids []int64) ([]*User, error) {
	var users []*User
	err := m.db.WithContext(ctx).Where("user_id IN ?", ids).Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}
