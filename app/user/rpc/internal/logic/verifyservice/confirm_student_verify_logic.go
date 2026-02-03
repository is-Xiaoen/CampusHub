/**
 * @projectName: CampusHub
 * @package: verifyservicelogic
 * @className: ConfirmStudentVerifyLogic
 * @author: lijunqi
 * @description: 用户确认/修改认证信息逻辑层
 * @date: 2026-01-31
 * @version: 1.0
 */

package verifyservicelogic

import (
	"context"
	"time"

	"activity-platform/app/user/model"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/constants"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// ConfirmStudentVerifyLogic 用户确认/修改认证信息逻辑处理器
type ConfirmStudentVerifyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewConfirmStudentVerifyLogic 创建确认认证信息逻辑实例
func NewConfirmStudentVerifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConfirmStudentVerifyLogic {
	return &ConfirmStudentVerifyLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ConfirmStudentVerify 用户确认/修改认证信息
// 业务逻辑:
//   - 确认无误 -> 状态改为4（已通过）
//   - 修改信息 -> 状态改为3（人工审核）
func (l *ConfirmStudentVerifyLogic) ConfirmStudentVerify(
	in *pb.ConfirmStudentVerifyReq,
) (*pb.ConfirmStudentVerifyResp, error) {
	// 1. 参数校验
	if in.UserId <= 0 {
		l.Errorf("ConfirmStudentVerify 参数错误: userId=%d", in.UserId)
		return nil, errorx.ErrInvalidParams("用户ID无效")
	}
	if in.VerifyId <= 0 {
		l.Errorf("ConfirmStudentVerify 参数错误: verifyId=%d", in.VerifyId)
		return nil, errorx.ErrInvalidParams("认证ID无效")
	}

	// 2. 查询认证记录
	verification, err := l.svcCtx.StudentVerificationModel.FindByID(l.ctx, in.VerifyId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			l.Infof("[WARN] ConfirmStudentVerify 记录不存在: verifyId=%d", in.VerifyId)
			return nil, errorx.ErrVerifyNotFound()
		}
		l.Errorf("ConfirmStudentVerify 查询失败: verifyId=%d, err=%v", in.VerifyId, err)
		return nil, errorx.ErrDBError(err)
	}

	// 3. 权限校验
	if verification.UserID != in.UserId {
		l.Infof("[WARN] ConfirmStudentVerify 无权操作: userId=%d, ownerId=%d",
			in.UserId, verification.UserID)
		return nil, errorx.ErrVerifyPermissionDeny()
	}

	// 4. 状态校验
	if !verification.CanConfirm() {
		l.Infof("[WARN] ConfirmStudentVerify 当前状态不允许确认: status=%d", verification.Status)
		return nil, errorx.ErrVerifyCannotConfirm()
	}

	var newStatus int8

	if in.IsConfirmed {
		// 5a. 用户确认无误，直接通过
		newStatus, err = l.handleConfirmed(in.VerifyId, in.UserId)
		if err != nil {
			return nil, err
		}
	} else {
		// 5b. 用户修改信息，转人工审核
		newStatus, err = l.handleModified(in, verification)
		if err != nil {
			return nil, err
		}
	}

	return &pb.ConfirmStudentVerifyResp{
		VerifyId:      in.VerifyId,
		NewStatus:     int32(newStatus),
		NewStatusDesc: constants.GetVerifyStatusName(newStatus),
	}, nil
}

// handleConfirmed 处理用户确认无误的情况
func (l *ConfirmStudentVerifyLogic) handleConfirmed(verifyID, userID int64) (int8, error) {
	newStatus := constants.VerifyStatusPassed
	now := time.Now()
	updates := map[string]interface{}{
		"verified_at": &now,
		"operator":    constants.VerifyOperatorUserConfirm,
	}

	if err := l.svcCtx.StudentVerificationModel.UpdateStatus(
		l.ctx, verifyID, newStatus, updates); err != nil {
		l.Errorf("ConfirmStudentVerify 更新状态失败: verifyId=%d, err=%v", verifyID, err)
		return 0, errorx.ErrDBError(err)
	}

	l.Infof("ConfirmStudentVerify 用户确认通过: userId=%d, verifyId=%d", userID, verifyID)
	return newStatus, nil
}

// handleModified 处理用户修改信息的情况
func (l *ConfirmStudentVerifyLogic) handleModified(
	in *pb.ConfirmStudentVerifyReq,
	verification *model.StudentVerification,
) (int8, error) {
	if in.ModifiedData == nil {
		l.Errorf("ConfirmStudentVerify 修改数据为空")
		return 0, errorx.ErrInvalidParams("修改数据不能为空")
	}

	// 校验修改后的唯一性
	if in.ModifiedData.SchoolName != "" && in.ModifiedData.StudentId != "" {
		exists, err := l.svcCtx.StudentVerificationModel.ExistsBySchoolAndStudentID(
			l.ctx, in.ModifiedData.SchoolName, in.ModifiedData.StudentId, in.UserId)
		if err != nil {
			l.Errorf("ConfirmStudentVerify 唯一性校验失败: err=%v", err)
			return 0, errorx.ErrDBError(err)
		}
		if exists {
			l.Infof("[WARN] ConfirmStudentVerify 修改后学号已被占用")
			return 0, errorx.ErrVerifyStudentIDUsed()
		}
	}

	newStatus := constants.VerifyStatusManualReview
	modifiedData := BuildModifiedData(in.ModifiedData, verification)

	if err := l.svcCtx.StudentVerificationModel.UpdateToManualReview(
		l.ctx, in.VerifyId, modifiedData); err != nil {
		l.Errorf("ConfirmStudentVerify 更新为人工审核失败: verifyId=%d, err=%v", in.VerifyId, err)
		return 0, errorx.ErrDBError(err)
	}

	l.Infof("ConfirmStudentVerify 转人工审核: userId=%d, verifyId=%d", in.UserId, in.VerifyId)
	return newStatus, nil
}
