/**
 * @projectName: CampusHub
 * @package: verifyservicelogic
 * @className: verify_helper
 * @author: lijunqi
 * @description: 学生认证服务公共辅助函数
 * @date: 2026-01-31
 * @version: 1.0
 */

package verifyservicelogic

import (
	"context"
	"database/sql"
	"time"

	"activity-platform/app/user/model"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/constants"
)

// ============================================================================
// 数据脱敏函数
// ============================================================================

// MaskRealName 真实姓名脱敏
// 规则：保留第一个字，其余用*替代
// 示例：张三 -> 张*，李四五 -> 李**
func MaskRealName(name string) string {
	if len(name) == 0 {
		return ""
	}
	runes := []rune(name)
	if len(runes) <= 1 {
		return name
	}
	masked := string(runes[0])
	for i := 1; i < len(runes); i++ {
		masked += "*"
	}
	return masked
}

// MaskStudentID 学号脱敏
// 规则：保留前4位和后2位，中间用****替代
// 示例：202301001 -> 2023****01
func MaskStudentID(studentID string) string {
	if len(studentID) <= 6 {
		return studentID
	}
	return studentID[:4] + "****" + studentID[len(studentID)-2:]
}

// ============================================================================
// 状态判断辅助函数
// ============================================================================

// ShouldReturnOcrData 判断是否应该返回OCR数据
// 待确认、人工审核中、已通过 状态时返回OCR数据
func ShouldReturnOcrData(status int8) bool {
	return status == constants.VerifyStatusWaitConfirm ||
		status == constants.VerifyStatusPassed ||
		status == constants.VerifyStatusManualReview
}

// ============================================================================
// Proto 转换辅助函数
// ============================================================================

// BuildVerifyOcrDataFromModel 从 Model 构建 VerifyOcrData
func BuildVerifyOcrDataFromModel(v *model.StudentVerification) *pb.VerifyOcrData {
	if v == nil {
		return nil
	}
	data := &pb.VerifyOcrData{
		RealName:      v.RealName,
		SchoolName:    v.SchoolName,
		StudentId:     v.StudentID,
		Department:    v.Department,
		AdmissionYear: v.AdmissionYear,
		OcrPlatform:   v.OcrPlatform,
	}
	if v.OcrConfidence.Valid {
		data.OcrConfidence = v.OcrConfidence.Float64
	}
	return data
}

// BuildMaskedVerifyInfo 构建脱敏后的认证信息
func BuildMaskedVerifyInfo(v *model.StudentVerification) *pb.GetVerifyInfoResp {
	if v == nil {
		return &pb.GetVerifyInfoResp{IsVerified: false}
	}

	var verifiedAt int64
	if v.VerifiedAt != nil {
		verifiedAt = v.VerifiedAt.Unix()
	}

	return &pb.GetVerifyInfoResp{
		IsVerified:    true,
		VerifyId:      v.ID,
		RealName:      MaskRealName(v.RealName),
		SchoolName:    v.SchoolName,
		StudentId:     MaskStudentID(v.StudentID),
		Department:    v.Department,
		AdmissionYear: v.AdmissionYear,
		VerifiedAt:    verifiedAt,
	}
}

// BuildIsVerifiedResp 构建 IsVerified 响应
func BuildIsVerifiedResp(v *model.StudentVerification) *pb.IsVerifiedResp {
	if v == nil || v.Status != constants.VerifyStatusPassed {
		return &pb.IsVerifiedResp{IsVerified: false}
	}

	var verifiedAt int64
	if v.VerifiedAt != nil {
		verifiedAt = v.VerifiedAt.Unix()
	}

	return &pb.IsVerifiedResp{
		IsVerified:    true,
		SchoolName:    v.SchoolName,
		StudentId:     MaskStudentID(v.StudentID),
		Department:    v.Department,
		AdmissionYear: v.AdmissionYear,
		VerifiedAt:    verifiedAt,
	}
}

// ============================================================================
// 确认/修改辅助函数
// ============================================================================

// GetOrDefault 获取值或默认值
func GetOrDefault(value, defaultValue string) string {
	if value != "" {
		return value
	}
	return defaultValue
}

// BuildModifiedData 构建修改后的数据
func BuildModifiedData(modifiedData *pb.VerifyModifiedData, original *model.StudentVerification) *model.VerifyModifiedData {
	return &model.VerifyModifiedData{
		RealName:      GetOrDefault(modifiedData.RealName, original.RealName),
		SchoolName:    GetOrDefault(modifiedData.SchoolName, original.SchoolName),
		StudentID:     GetOrDefault(modifiedData.StudentId, original.StudentID),
		Department:    GetOrDefault(modifiedData.Department, original.Department),
		AdmissionYear: GetOrDefault(modifiedData.AdmissionYear, original.AdmissionYear),
	}
}

// ============================================================================
// 状态更新辅助函数（用于 UpdateVerifyStatus）
// ============================================================================

// StatusUpdateContext 状态更新上下文
type StatusUpdateContext struct {
	VerifyID     int64
	NewStatus    int8
	Operator     string
	OcrData      *pb.VerifyOcrData
	RejectReason string
	ReviewerID   int64
}

// BuildPassedUpdates 构建通过状态的更新字段
func BuildPassedUpdates(ctx *StatusUpdateContext) map[string]interface{} {
	now := time.Now()
	updates := map[string]interface{}{
		"verified_at": &now,
		"operator":    ctx.Operator,
	}
	if ctx.Operator == constants.VerifyOperatorManualReview {
		updates["reviewed_at"] = &now
		if ctx.ReviewerID > 0 {
			updates["reviewer_id"] = sql.NullInt64{Int64: ctx.ReviewerID, Valid: true}
		}
	}
	return updates
}

// BuildRejectedUpdates 构建拒绝状态的更新字段
func BuildRejectedUpdates(ctx *StatusUpdateContext) map[string]interface{} {
	now := time.Now()
	updates := map[string]interface{}{
		"reject_reason": ctx.RejectReason,
		"reviewed_at":   &now,
		"operator":      ctx.Operator,
	}
	if ctx.ReviewerID > 0 {
		updates["reviewer_id"] = sql.NullInt64{Int64: ctx.ReviewerID, Valid: true}
	}
	return updates
}

// BuildOcrResultData 从 Proto 构建 OcrResultData
func BuildOcrResultData(ocrData *pb.VerifyOcrData) *model.OcrResultData {
	if ocrData == nil {
		return nil
	}
	return &model.OcrResultData{
		RealName:      ocrData.RealName,
		SchoolName:    ocrData.SchoolName,
		StudentID:     ocrData.StudentId,
		Department:    ocrData.Department,
		AdmissionYear: ocrData.AdmissionYear,
		OcrPlatform:   ocrData.OcrPlatform,
		OcrConfidence: ocrData.OcrConfidence,
		OcrRawJSON:    ocrData.OcrRawJson,
	}
}

// ============================================================================
// 数据加密辅助函数（预留接口）
// ============================================================================

// EncryptSensitiveData 加密敏感数据（预留接口）
// TODO: 实现AES加密，当前直接返回原文
func EncryptSensitiveData(ctx context.Context, plainText string) string {
	// TODO: 接入加密服务
	return plainText
}

// DecryptSensitiveData 解密敏感数据（预留接口）
// TODO: 实现AES解密，当前直接返回原文
func DecryptSensitiveData(ctx context.Context, cipherText string) string {
	// TODO: 接入解密服务
	return cipherText
}
