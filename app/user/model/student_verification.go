/**
 * @projectName: CampusHub
 * @package: model
 * @className: StudentVerification
 * @author: lijunqi
 * @description: 学生认证实体及数据访问层
 * @date: 2026-01-31
 * @version: 1.0
 */

package model

import (
	"context"
	"database/sql"
	"time"

	"activity-platform/common/constants"

	"gorm.io/gorm"
)

// StudentVerification 学生认证实体
type StudentVerification struct {
	// ==================== 基础字段 ====================

	// 主键ID（数据库自增）
	ID int64 `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	// 用户ID
	UserID int64 `gorm:"uniqueIndex:uk_user_id;column:user_id;not null" json:"user_id"`

	// ==================== 状态机字段 ====================

	// 认证状态：0初始 1OCR中 2待确认 3人工审核 4通过 5拒绝 6超时 7取消 8OCR失败
	Status int8 `gorm:"column:status;not null;default:0" json:"status"`

	// ==================== 认证信息字段（敏感数据加密存储） ====================

	// 真实姓名（AES加密）
	RealName string `gorm:"column:real_name;size:100" json:"real_name"`
	// 学校名称
	SchoolName string `gorm:"column:school_name;size:100" json:"school_name"`
	// 学号（AES加密）
	StudentID string `gorm:"uniqueIndex:uk_school_student,priority:2;column:student_id;size:100" json:"student_id"`
	// 院系
	Department string `gorm:"column:department;size:100" json:"department"`
	// 入学年份
	AdmissionYear string `gorm:"column:admission_year;size:10" json:"admission_year"`

	// ==================== 图片字段 ====================

	// 学生证正面图片URL
	FrontImageURL string `gorm:"column:front_image_url;size:500" json:"front_image_url"`
	// 学生证详情面图片URL
	BackImageURL string `gorm:"column:back_image_url;size:500" json:"back_image_url"`

	// ==================== OCR审计字段 ====================

	// OCR平台：tencent/aliyun
	OcrPlatform string `gorm:"column:ocr_platform;size:20;not null;default:''" json:"ocr_platform"`
	// OCR原始响应JSON（用于审计追溯）
	OcrRawJSON sql.NullString `gorm:"column:ocr_raw_json;type:text" json:"ocr_raw_json"`
	// OCR识别置信度（0-100）
	OcrConfidence sql.NullFloat64 `gorm:"column:ocr_confidence;type:decimal(5,2)" json:"ocr_confidence"`

	// ==================== 审核相关字段 ====================

	// 拒绝原因
	RejectReason string `gorm:"column:reject_reason;size:255" json:"reject_reason"`
	// 取消原因
	CancelReason string `gorm:"column:cancel_reason;size:255" json:"cancel_reason"`
	// 审核人ID（人工审核时）
	ReviewerID sql.NullInt64 `gorm:"column:reviewer_id" json:"reviewer_id"`
	// 操作来源：user_apply/ocr_callback/manual_review/timeout_job
	Operator string `gorm:"column:operator;size:50" json:"operator"`

	// ==================== 时间字段 ====================

	// 认证通过时间
	VerifiedAt *time.Time `gorm:"column:verified_at" json:"verified_at"`
	// OCR完成时间
	OcrCompletedAt *time.Time `gorm:"column:ocr_completed_at" json:"ocr_completed_at"`
	// 人工审核时间
	ReviewedAt *time.Time `gorm:"column:reviewed_at" json:"reviewed_at"`
	// 创建时间
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	// 更新时间
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (StudentVerification) TableName() string {
	return "student_verifications"
}

// ============================================================================
// 数据访问层接口定义
// ============================================================================

// IStudentVerificationModel 学生认证数据访问层接口
type IStudentVerificationModel interface {
	// ==================== 基础 CRUD ====================

	// Create 创建认证记录
	Create(ctx context.Context, verification *StudentVerification) error
	// FindByID 根据主键ID查询
	FindByID(ctx context.Context, id int64) (*StudentVerification, error)
	// FindByUserID 根据用户ID查询认证信息
	FindByUserID(ctx context.Context, userID int64) (*StudentVerification, error)
	// Update 更新认证信息
	Update(ctx context.Context, verification *StudentVerification) error

	// ==================== 状态查询 ====================

	// ExistsByUserID 检查用户认证记录是否存在
	ExistsByUserID(ctx context.Context, userID int64) (bool, error)
	// ExistsBySchoolAndStudentID 检查学校+学号是否已被占用（排除指定用户）
	ExistsBySchoolAndStudentID(ctx context.Context, schoolName, studentID string, excludeUserID int64) (bool, error)
	// IsVerified 检查用户是否已通过认证
	IsVerified(ctx context.Context, userID int64) (bool, error)

	// ==================== 状态更新 ====================

	// UpdateStatus 更新认证状态
	UpdateStatus(ctx context.Context, id int64, newStatus int8, updates map[string]interface{}) error
	// UpdateOcrResult 更新OCR识别结果
	UpdateOcrResult(ctx context.Context, id int64, ocrData *OcrResultData) error
	// UpdateToManualReview 更新为人工审核状态
	UpdateToManualReview(ctx context.Context, id int64, modifiedData *VerifyModifiedData) error

	// ==================== 查询列表 ====================

	// FindTimeoutRecords 查询超时的OCR记录
	FindTimeoutRecords(ctx context.Context, timeoutMinutes int) ([]*StudentVerification, error)
	// FindByStatus 根据状态查询列表（分页）
	FindByStatus(ctx context.Context, status int8, page, pageSize int) ([]*StudentVerification, int64, error)
}

// OcrResultData OCR识别结果数据
type OcrResultData struct {
	// 真实姓名
	RealName string
	// 学校名称
	SchoolName string
	// 学号
	StudentID string
	// 院系
	Department string
	// 入学年份
	AdmissionYear string
	// OCR平台
	OcrPlatform string
	// OCR置信度
	OcrConfidence float64
	// OCR原始响应JSON
	OcrRawJSON string
}

// VerifyModifiedData 用户修改后的数据
type VerifyModifiedData struct {
	// 真实姓名
	RealName string
	// 学校名称
	SchoolName string
	// 学号
	StudentID string
	// 院系
	Department string
	// 入学年份
	AdmissionYear string
}

// ============================================================================
// 数据访问层实现
// ============================================================================

// 确保 StudentVerificationModel 实现 IStudentVerificationModel 接口
var _ IStudentVerificationModel = (*StudentVerificationModel)(nil)

// StudentVerificationModel 学生认证数据访问层
type StudentVerificationModel struct {
	db *gorm.DB
}

// NewStudentVerificationModel 创建学生认证Model实例
func NewStudentVerificationModel(db *gorm.DB) IStudentVerificationModel {
	return &StudentVerificationModel{db: db}
}

// ==================== 基础 CRUD ====================

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
func (m *StudentVerificationModel) Update(
	ctx context.Context,
	verification *StudentVerification,
) error {
	return m.db.WithContext(ctx).Save(verification).Error
}

// ==================== 状态查询 ====================

// ExistsByUserID 检查用户认证记录是否存在
func (m *StudentVerificationModel) ExistsByUserID(
	ctx context.Context,
	userID int64,
) (bool, error) {
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

// ExistsBySchoolAndStudentID 检查学校+学号是否已被占用（排除指定用户）
func (m *StudentVerificationModel) ExistsBySchoolAndStudentID(
	ctx context.Context,
	schoolName, studentID string,
	excludeUserID int64,
) (bool, error) {
	var count int64
	query := m.db.WithContext(ctx).
		Model(&StudentVerification{}).
		Where("school_name = ? AND student_id = ?", schoolName, studentID).
		Where("status = ?", constants.VerifyStatusPassed)
	// 排除指定用户
	if excludeUserID > 0 {
		query = query.Where("user_id != ?", excludeUserID)
	}
	err := query.Count(&count).Error
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
		Where("user_id = ? AND status = ?", userID, constants.VerifyStatusPassed).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ==================== 状态更新 ====================

// UpdateStatus 更新认证状态
func (m *StudentVerificationModel) UpdateStatus(
	ctx context.Context,
	id int64,
	newStatus int8,
	updates map[string]interface{},
) error {
	if updates == nil {
		updates = make(map[string]interface{})
	}
	updates["status"] = newStatus
	return m.db.WithContext(ctx).
		Model(&StudentVerification{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdateOcrResult 更新OCR识别结果
func (m *StudentVerificationModel) UpdateOcrResult(
	ctx context.Context,
	id int64,
	ocrData *OcrResultData,
) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":           constants.VerifyStatusWaitConfirm,
		"real_name":        ocrData.RealName,
		"school_name":      ocrData.SchoolName,
		"student_id":       ocrData.StudentID,
		"department":       ocrData.Department,
		"admission_year":   ocrData.AdmissionYear,
		"ocr_platform":     ocrData.OcrPlatform,
		"ocr_confidence":   sql.NullFloat64{Float64: ocrData.OcrConfidence, Valid: true},
		"ocr_raw_json":     sql.NullString{String: ocrData.OcrRawJSON, Valid: ocrData.OcrRawJSON != ""},
		"ocr_completed_at": &now,
		"operator":         constants.VerifyOperatorOcrCallback,
	}
	return m.db.WithContext(ctx).
		Model(&StudentVerification{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdateToManualReview 更新为人工审核状态
func (m *StudentVerificationModel) UpdateToManualReview(
	ctx context.Context,
	id int64,
	modifiedData *VerifyModifiedData,
) error {
	updates := map[string]interface{}{
		"status":         constants.VerifyStatusManualReview,
		"real_name":      modifiedData.RealName,
		"school_name":    modifiedData.SchoolName,
		"student_id":     modifiedData.StudentID,
		"department":     modifiedData.Department,
		"admission_year": modifiedData.AdmissionYear,
		"operator":       constants.VerifyOperatorUserConfirm,
	}
	return m.db.WithContext(ctx).
		Model(&StudentVerification{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// ==================== 查询列表 ====================

// FindTimeoutRecords 查询超时的OCR记录
// 查找状态为OCR审核中且超过指定时间的记录
func (m *StudentVerificationModel) FindTimeoutRecords(
	ctx context.Context,
	timeoutMinutes int,
) ([]*StudentVerification, error) {
	var list []*StudentVerification
	timeoutThreshold := time.Now().Add(-time.Duration(timeoutMinutes) * time.Minute)
	err := m.db.WithContext(ctx).
		Where("status = ? AND created_at < ?", constants.VerifyStatusOcrPending, timeoutThreshold).
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// FindByStatus 根据状态查询列表（分页）
func (m *StudentVerificationModel) FindByStatus(
	ctx context.Context,
	status int8,
	page, pageSize int,
) ([]*StudentVerification, int64, error) {
	var list []*StudentVerification
	var total int64
	query := m.db.WithContext(ctx).Model(&StudentVerification{})
	if status >= 0 {
		query = query.Where("status = ?", status)
	}
	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ============================================================================
// 辅助方法
// ============================================================================

// CanApply 检查当前记录是否可以申请
func (v *StudentVerification) CanApply() bool {
	return constants.CanApply(v.Status)
}

// CanCancel 检查当前记录是否可以取消
func (v *StudentVerification) CanCancel() bool {
	return constants.CanCancel(v.Status)
}

// CanConfirm 检查当前记录是否可以确认
func (v *StudentVerification) CanConfirm() bool {
	return constants.CanConfirm(v.Status)
}

// IsPassed 检查是否已通过认证
func (v *StudentVerification) IsPassed() bool {
	return v.Status == constants.VerifyStatusPassed
}

// GetStatusName 获取状态名称
func (v *StudentVerification) GetStatusName() string {
	return constants.GetVerifyStatusName(v.Status)
}

// GetNeedAction 获取前端应执行的动作
func (v *StudentVerification) GetNeedAction() string {
	return constants.GetNeedAction(v.Status)
}
