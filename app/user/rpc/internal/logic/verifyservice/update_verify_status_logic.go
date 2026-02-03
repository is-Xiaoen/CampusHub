/**
 * @projectName: CampusHub
 * @package: verifyservicelogic
 * @className: UpdateVerifyStatusLogic
 * @author: lijunqi
 * @description: 更新认证状态逻辑层（内部接口）
 * @date: 2026-01-31
 * @version: 1.0
 */

package verifyservicelogic

import (
	"context"
	"time"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/constants"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// UpdateVerifyStatusLogic 更新认证状态逻辑处理器
type UpdateVerifyStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewUpdateVerifyStatusLogic 创建更新认证状态逻辑实例
func NewUpdateVerifyStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateVerifyStatusLogic {
	return &UpdateVerifyStatusLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// UpdateVerifyStatus 更新认证状态
// 业务逻辑:
//   - 供 MQ Consumer（OCR回调、人工审核结果）、定时任务（超时处理）调用
//   - 处理认证流程中的状态流转
func (l *UpdateVerifyStatusLogic) UpdateVerifyStatus(in *pb.UpdateVerifyStatusReq) (*pb.UpdateVerifyStatusResp, error) {
	// 1. 参数校验
	if err := l.validateParams(in); err != nil {
		return nil, err
	}

	// 2. 查询认证记录
	verification, err := l.svcCtx.StudentVerificationModel.FindByID(l.ctx, in.VerifyId)
	if err != nil {
		return nil, l.handleFindError(err, in.VerifyId)
	}

	// 3. 用户ID校验（如果提供了）
	if in.UserId > 0 && verification.UserID != in.UserId {
		l.Infof("[WARN] UpdateVerifyStatus 用户ID不匹配: expected=%d, got=%d",
			verification.UserID, in.UserId)
		return nil, errorx.ErrVerifyPermissionDeny()
	}

	beforeStatus := verification.Status
	newStatus := int8(in.NewStatus)

	// 4. 状态转换校验
	if !constants.CanVerifyTransition(beforeStatus, newStatus) {
		l.Infof("[WARN] UpdateVerifyStatus 无效的状态转换: from=%d, to=%d",
			beforeStatus, newStatus)
		return nil, errorx.ErrVerifyInvalidTransit()
	}

	// 5. 根据新状态执行不同的更新逻辑
	if err := l.executeStatusUpdate(in, newStatus); err != nil {
		return nil, err
	}

	return &pb.UpdateVerifyStatusResp{
		Success:      true,
		BeforeStatus: int32(beforeStatus),
		AfterStatus:  int32(newStatus),
	}, nil
}

// validateParams 参数校验
func (l *UpdateVerifyStatusLogic) validateParams(in *pb.UpdateVerifyStatusReq) error {
	if in.VerifyId <= 0 {
		l.Errorf("UpdateVerifyStatus 参数错误: verifyId=%d", in.VerifyId)
		return errorx.ErrInvalidParams("认证ID无效")
	}
	if in.NewStatus < 0 || in.NewStatus > 8 {
		l.Errorf("UpdateVerifyStatus 参数错误: newStatus=%d", in.NewStatus)
		return errorx.ErrInvalidParams("无效的状态值")
	}
	return nil
}

// handleFindError 处理查询错误
func (l *UpdateVerifyStatusLogic) handleFindError(err error, verifyID int64) error {
	if err == gorm.ErrRecordNotFound {
		l.Infof("[WARN] UpdateVerifyStatus 记录不存在: verifyId=%d", verifyID)
		return errorx.ErrVerifyNotFound()
	}
	l.Errorf("UpdateVerifyStatus 查询失败: verifyId=%d, err=%v", verifyID, err)
	return errorx.ErrDBError(err)
}

// executeStatusUpdate 执行状态更新
// 根据目标状态分发到不同的处理函数
func (l *UpdateVerifyStatusLogic) executeStatusUpdate(in *pb.UpdateVerifyStatusReq, newStatus int8) error {
	switch newStatus {
	case constants.VerifyStatusWaitConfirm:
		// 状态2: OCR识别成功，等待用户确认
		return l.handleOcrSuccess(in)
	case constants.VerifyStatusPassed:
		// 状态4: 认证通过（用户确认无误/人工审核通过）
		return l.handlePassed(in, newStatus)
	case constants.VerifyStatusRejected:
		// 状态5: 认证拒绝（人工审核拒绝）
		return l.handleRejected(in, newStatus)
	case constants.VerifyStatusTimeout:
		// 状态6: OCR超时（10分钟未完成）
		return l.handleTimeout(in, newStatus)
	case constants.VerifyStatusOcrFailed:
		// 状态8: OCR失败（双OCR都失败）
		return l.handleOcrFailed(in, newStatus)
	default:
		// 其他状态：通用更新处理
		return l.handleDefault(in, newStatus)
	}
}

// handleOcrSuccess 处理OCR成功
func (l *UpdateVerifyStatusLogic) handleOcrSuccess(in *pb.UpdateVerifyStatusReq) error {
	if in.OcrData != nil {
		ocrData := BuildOcrResultData(in.OcrData)
		if err := l.svcCtx.StudentVerificationModel.UpdateOcrResult(
			l.ctx, in.VerifyId, ocrData); err != nil {
			l.Errorf("UpdateVerifyStatus 更新OCR结果失败: verifyId=%d, err=%v", in.VerifyId, err)
			return errorx.ErrDBError(err)
		}
		l.Infof("UpdateVerifyStatus OCR成功: verifyId=%d, platform=%s",
			in.VerifyId, in.OcrData.OcrPlatform)
	} else {
		now := time.Now()
		updates := map[string]interface{}{
			"ocr_completed_at": &now,
			"operator":         in.Operator,
		}
		if err := l.svcCtx.StudentVerificationModel.UpdateStatus(
			l.ctx, in.VerifyId, constants.VerifyStatusWaitConfirm, updates); err != nil {
			l.Errorf("UpdateVerifyStatus 更新状态失败: verifyId=%d, err=%v", in.VerifyId, err)
			return errorx.ErrDBError(err)
		}
	}
	return nil
}

// handlePassed 处理认证通过
func (l *UpdateVerifyStatusLogic) handlePassed(in *pb.UpdateVerifyStatusReq, newStatus int8) error {
	ctx := &StatusUpdateContext{
		VerifyID:   in.VerifyId,
		NewStatus:  newStatus,
		Operator:   in.Operator,
		ReviewerID: in.UserId,
	}
	updates := BuildPassedUpdates(ctx)

	if err := l.svcCtx.StudentVerificationModel.UpdateStatus(
		l.ctx, in.VerifyId, newStatus, updates); err != nil {
		l.Errorf("UpdateVerifyStatus 更新状态失败: verifyId=%d, err=%v", in.VerifyId, err)
		return errorx.ErrDBError(err)
	}
	l.Infof("UpdateVerifyStatus 认证通过: verifyId=%d, operator=%s", in.VerifyId, in.Operator)
	return nil
}

// handleRejected 处理认证拒绝
func (l *UpdateVerifyStatusLogic) handleRejected(in *pb.UpdateVerifyStatusReq, newStatus int8) error {
	ctx := &StatusUpdateContext{
		VerifyID:     in.VerifyId,
		NewStatus:    newStatus,
		Operator:     in.Operator,
		RejectReason: in.RejectReason,
		ReviewerID:   in.UserId,
	}
	updates := BuildRejectedUpdates(ctx)

	if err := l.svcCtx.StudentVerificationModel.UpdateStatus(
		l.ctx, in.VerifyId, newStatus, updates); err != nil {
		l.Errorf("UpdateVerifyStatus 更新状态失败: verifyId=%d, err=%v", in.VerifyId, err)
		return errorx.ErrDBError(err)
	}
	l.Infof("UpdateVerifyStatus 认证拒绝: verifyId=%d, reason=%s", in.VerifyId, in.RejectReason)
	return nil
}

// handleTimeout 处理OCR超时
func (l *UpdateVerifyStatusLogic) handleTimeout(in *pb.UpdateVerifyStatusReq, newStatus int8) error {
	updates := map[string]interface{}{"operator": in.Operator}
	if err := l.svcCtx.StudentVerificationModel.UpdateStatus(
		l.ctx, in.VerifyId, newStatus, updates); err != nil {
		l.Errorf("UpdateVerifyStatus 更新状态失败: verifyId=%d, err=%v", in.VerifyId, err)
		return errorx.ErrDBError(err)
	}
	l.Infof("UpdateVerifyStatus OCR超时: verifyId=%d", in.VerifyId)
	return nil
}

// handleOcrFailed 处理OCR失败
func (l *UpdateVerifyStatusLogic) handleOcrFailed(in *pb.UpdateVerifyStatusReq, newStatus int8) error {
	updates := map[string]interface{}{"operator": in.Operator}
	if err := l.svcCtx.StudentVerificationModel.UpdateStatus(
		l.ctx, in.VerifyId, newStatus, updates); err != nil {
		l.Errorf("UpdateVerifyStatus 更新状态失败: verifyId=%d, err=%v", in.VerifyId, err)
		return errorx.ErrDBError(err)
	}
	l.Infof("UpdateVerifyStatus OCR失败: verifyId=%d", in.VerifyId)
	return nil
}

// handleDefault 处理其他状态
func (l *UpdateVerifyStatusLogic) handleDefault(in *pb.UpdateVerifyStatusReq, newStatus int8) error {
	updates := map[string]interface{}{"operator": in.Operator}
	if err := l.svcCtx.StudentVerificationModel.UpdateStatus(
		l.ctx, in.VerifyId, newStatus, updates); err != nil {
		l.Errorf("UpdateVerifyStatus 更新状态失败: verifyId=%d, err=%v", in.VerifyId, err)
		return errorx.ErrDBError(err)
	}
	l.Infof("UpdateVerifyStatus 状态更新: verifyId=%d, to=%d", in.VerifyId, newStatus)
	return nil
}
