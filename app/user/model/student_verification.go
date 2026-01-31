/**
 * @projectName: CampusHub
 * @package: model
 * @className: StudentVerification
 * @author: lijunqi
 * @description: 学生认证实体及数据访问层
 * @date: 2026-01-30
 * @version: 1.0
 */

package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// VerificationStatus 认证状态
const (
	// VerificationStatusNone 未认证
	VerificationStatusNone int8 = 0
	// VerificationStatusPending 待审核
	VerificationStatusPending int8 = 1
	// VerificationStatusVerified 已认证
	VerificationStatusVerified int8 = 2
	// VerificationStatusRejected 已拒绝
	VerificationStatusRejected int8 = 3
)

// StudentVerification 学生认证实体
type StudentVerification struct {
	// 主键ID
	ID int64 `gorm:"primaryKey;column:id" json:"id"`
	// 用户ID
	UserID int64 `gorm:"uniqueIndex:uk_user_id;column:user_id;not null" json:"user_id"`
	// 认证状态：0未认证 1待审核 2已认证 3已拒绝
	Status int8 `gorm:"column:status;not null;default:0" json:"status"`
	// 真实姓名
	RealName string `gorm:"column:real_name;size:50" json:"real_name"`
	// 学校名称
	SchoolName string `gorm:"column:school_name;size:100" json:"school_name"`
	// 学号
	StudentID string `gorm:"uniqueIndex:uk_student_id;column:student_id;size:50" json:"student_id"`
	// 院系
	Department string `gorm:"column:department;size:100" json:"department"`
	// 入学年份
	AdmissionYear string `gorm:"column:admission_year;size:10" json:"admission_year"`
	// 拒绝原因
	RejectReason string `gorm:"column:reject_reason;size:255" json:"reject_reason"`
	// 认证通过时间
	VerifiedAt *time.Time `gorm:"column:verified_at" json:"verified_at"`
	// 创建时间
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	// 更新时间
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (StudentVerification) TableName() string {
	return "student_verifications"
}

// IStudentVerificationModel 学生认证数据访问层接口
type IStudentVerificationModel interface {
	// Create 创建认证记录
	Create(ctx context.Context, verification *StudentVerification) error
	// FindByID 根据主键ID查询
	FindByID(ctx context.Context, id int64) (*StudentVerification, error)
	// FindByUserID 根据用户ID查询认证信息
	FindByUserID(ctx context.Context, userID int64) (*StudentVerification, error)
	// Update 更新认证信息
	Update(ctx context.Context, verification *StudentVerification) error
	// UpdateStatus 更新认证状态
	UpdateStatus(ctx context.Context, userID int64, status int8, rejectReason string) error
	// ExistsByUserID 检查用户认证记录是否存在
	ExistsByUserID(ctx context.Context, userID int64) (bool, error)
	// ExistsByStudentID 检查学号是否已被认证
	ExistsByStudentID(ctx context.Context, studentID string) (bool, error)
	// IsVerified 检查用户是否已通过认证
	IsVerified(ctx context.Context, userID int64) (bool, error)
}

// 确保 StudentVerificationModel 实现 IStudentVerificationModel 接口
// 确保如果不对，可以在编译阶段就报错
var _ IStudentVerificationModel = (*StudentVerificationModel)(nil)

// StudentVerificationModel 学生认证数据访问层
type StudentVerificationModel struct {
	db *gorm.DB
}

// NewStudentVerificationModel 创建学生认证Model实例
func NewStudentVerificationModel(db *gorm.DB) IStudentVerificationModel {
	return &StudentVerificationModel{db: db}
}

// Create 创建认证记录
func (m *StudentVerificationModel) Create(
	ctx context.Context,
	verification *StudentVerification,
) error {

	return m.db.WithContext(ctx).Create(verification).Error
}

// FindByID 根据主键ID查询
func (m *StudentVerificationModel) FindByID(
	ctx context.Context,
	id int64,
) (*StudentVerification, error) {
	var verification StudentVerification
	err := m.db.WithContext(ctx).First(&verification, id).Error
	if err != nil {
		return nil, err
	}
	return &verification, nil
}

// FindByUserID 根据用户ID查询认证信息
func (m *StudentVerificationModel) FindByUserID(
	ctx context.Context,
	userID int64,
) (*StudentVerification, error) {
	var verification StudentVerification
	err := m.db.WithContext(ctx).Where("user_id = ?", userID).First(&verification).Error
	if err != nil {
		return nil, err
	}
	return &verification, nil
}

// Update 更新认证信息
func (m *StudentVerificationModel) Update(ctx context.Context, verification *StudentVerification) error {
	return m.db.WithContext(ctx).Save(verification).Error
}

// UpdateStatus 更新认证状态
func (m *StudentVerificationModel) UpdateStatus(
	ctx context.Context,
	userID int64,
	status int8,
	rejectReason string,
) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if status == VerificationStatusRejected && rejectReason != "" {
		updates["reject_reason"] = rejectReason
	}
	if status == VerificationStatusVerified {
		now := time.Now()
		updates["verified_at"] = &now
	}
	return m.db.WithContext(ctx).
		Model(&StudentVerification{}).
		Where("user_id = ?", userID).
		Updates(updates).Error
}

// ExistsByUserID 检查用户认证记录是否存在
func (m *StudentVerificationModel) ExistsByUserID(ctx context.Context, userID int64) (bool, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&StudentVerification{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsByStudentID 检查学号是否已被认证
func (m *StudentVerificationModel) ExistsByStudentID(
	ctx context.Context,
	studentID string,
) (bool, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&StudentVerification{}).
		Where("student_id = ? AND status = ?", studentID, VerificationStatusVerified).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsVerified 检查用户是否已通过认证
func (m *StudentVerificationModel) IsVerified(
	ctx context.Context,
	userID int64,
) (bool, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&StudentVerification{}).
		Where("user_id = ? AND status = ?", userID, VerificationStatusVerified).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
