/**
 * @projectName: CampusHub
 * @package: handler
 * @className: VerifyCallbackHandler
 * @author: lijunqi
 * @description: 学生认证回调消息处理器
 * @date: 2026-01-30
 * @version: 1.0
 */

package handler

import (
	"context"
	"encoding/json"

	"activity-platform/app/user/mq/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// VerifyCallbackData 认证回调消息数据
// [待确认] 需要和 OCR 服务/审核后台 确认字段
type VerifyCallbackData struct {
	// VerifyID 认证记录ID
	VerifyID int64 `json:"verify_id"`

	// UserID 用户ID
	UserID int64 `json:"user_id"`

	// CallbackType 回调类型: ocr_result, manual_review
	CallbackType string `json:"callback_type"`

	// NewStatus 新状态
	NewStatus int32 `json:"new_status"`

	// OCRData OCR识别数据（OCR回调时有值）
	OCRData *OCRData `json:"ocr_data,omitempty"`

	// RejectReason 拒绝原因（审核拒绝时有值）
	RejectReason string `json:"reject_reason,omitempty"`

	// Operator 操作人（人工审核时有值）
	Operator string `json:"operator,omitempty"`
}

// OCRData OCR识别结果
type OCRData struct {
	SchoolName    string `json:"school_name"`
	StudentID     string `json:"student_id"`
	RealName      string `json:"real_name"`
	Department    string `json:"department"`
	AdmissionYear int32  `json:"admission_year"`
	Confidence    int32  `json:"confidence"`
}

// NewVerifyCallbackHandler 创建认证回调处理器
func NewVerifyCallbackHandler(svcCtx *svc.ServiceContext) Handler {
	return func(ctx context.Context, msg *Message) error {
		logx.WithContext(ctx).Infof("[VerifyCallbackHandler] 收到消息: id=%s", msg.ID)

		// 1. 解析消息
		var data VerifyCallbackData
		if err := json.Unmarshal([]byte(msg.Data), &data); err != nil {
			logx.WithContext(ctx).Errorf("[VerifyCallbackHandler] 解析消息失败: %v, data=%s",
				err, msg.Data)
			return nil
		}

		// 2. 参数校验
		if data.VerifyID <= 0 || data.UserID <= 0 {
			logx.WithContext(ctx).Errorf("[VerifyCallbackHandler] 无效参数: verifyId=%d, userId=%d",
				data.VerifyID, data.UserID)
			return nil
		}

		// 3. 根据回调类型处理
		switch data.CallbackType {
		case "ocr_result":
			return handleOCRResult(ctx, svcCtx, &data)
		case "manual_review":
			return handleManualReview(ctx, svcCtx, &data)
		default:
			logx.WithContext(ctx).Errorf("[VerifyCallbackHandler] 未知回调类型: %s", data.CallbackType)
			return nil
		}
	}
}

// handleOCRResult 处理OCR识别结果
func handleOCRResult(ctx context.Context, svcCtx *svc.ServiceContext, data *VerifyCallbackData) error {
	logx.WithContext(ctx).Infof("[VerifyCallbackHandler] 处理OCR结果: verifyId=%d, userId=%d",
		data.VerifyID, data.UserID)

	// TODO: 实现OCR结果处理逻辑
	_ = svcCtx // 避免未使用警告

	return nil
}

// handleManualReview 处理人工审核结果
func handleManualReview(ctx context.Context, svcCtx *svc.ServiceContext, data *VerifyCallbackData) error {
	logx.WithContext(ctx).Infof("[VerifyCallbackHandler] 处理人工审核: verifyId=%d, userId=%d, operator=%s",
		data.VerifyID, data.UserID, data.Operator)

	// TODO: 实现人工审核结果处理逻辑
	_ = svcCtx // 避免未使用警告

	return nil
}
