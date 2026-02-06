/**
 * @projectName: CampusHub
 * @package: verifyservicelogic
 * @className: CancelStudentVerifyLogic
 * @author: lijunqi
 * @description: 取消认证申请逻辑层
 * @date: 2026-01-31
 * @version: 1.0
 */

package verifyservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/constants"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// CancelStudentVerifyLogic 取消认证申请逻辑处理器
type CancelStudentVerifyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewCancelStudentVerifyLogic 创建取消认证申请逻辑实例
func NewCancelStudentVerifyLogic(
	ctx context.Context,
	svcCtx *svc.ServiceContext,
) *CancelStudentVerifyLogic {
	return &CancelStudentVerifyLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CancelStudentVerify 取消认证申请
// 业务逻辑:
//   - 状态改为7（已取消）
//   - 用户可重新申请
func (l *CancelStudentVerifyLogic) CancelStudentVerify(
	in *pb.CancelStudentVerifyReq,
) (*pb.CancelStudentVerifyResp, error) {
	// 1. 参数校验
	if in.UserId <= 0 {
		l.Errorf("CancelStudentVerify 参数错误: userId=%d", in.UserId)
		return nil, errorx.ErrInvalidParams("用户ID无效")
	}
	if in.VerifyId <= 0 {
		l.Errorf("CancelStudentVerify 参数错误: verifyId=%d", in.VerifyId)
		return nil, errorx.ErrInvalidParams("认证ID无效")
	}

	// 2. 查询认证记录
	verification, err := l.svcCtx.StudentVerificationModel.FindByID(l.ctx, in.VerifyId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			l.Infof("[WARN] CancelStudentVerify 记录不存在: verifyId=%d", in.VerifyId)
			return nil, errorx.ErrVerifyNotFound()
		}
		l.Errorf("CancelStudentVerify 查询失败: verifyId=%d, err=%v", in.VerifyId, err)
		return nil, errorx.ErrDBError(err)
	}

	// 3. 权限校验
	if verification.UserID != in.UserId {
		l.Infof("[WARN] CancelStudentVerify 无权操作: userId=%d, ownerId=%d",
			in.UserId, verification.UserID)
		return nil, errorx.ErrVerifyPermissionDeny()
	}

	// 4. 状态校验
	if !verification.CanCancel() {
		l.Infof("[WARN] CancelStudentVerify 当前状态不允许取消: status=%d", verification.Status)
		return nil, errorx.ErrVerifyCannotCancel()
	}

	// 5. 更新状态为已取消
	newStatus := constants.VerifyStatusCancelled
	updates := map[string]interface{}{
		"cancel_reason": in.CancelReason,
		"operator":      constants.VerifyOperatorUserCancel,
	}

	if err := l.svcCtx.StudentVerificationModel.UpdateStatus(
		l.ctx, in.VerifyId, newStatus, updates); err != nil {
		l.Errorf("CancelStudentVerify 更新状态失败: verifyId=%d, err=%v", in.VerifyId, err)
		return nil, errorx.ErrDBError(err)
	}

	l.Infof("CancelStudentVerify 取消成功: userId=%d, verifyId=%d, reason=%s",
		in.UserId, in.VerifyId, in.CancelReason)

	return &pb.CancelStudentVerifyResp{
		VerifyId:      in.VerifyId,
		NewStatus:     int32(newStatus),
		NewStatusDesc: constants.GetVerifyStatusName(newStatus),
	}, nil
}
