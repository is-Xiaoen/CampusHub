/**
 * @projectName: CampusHub
 * @package: verifyservicelogic
 * @className: GetVerifyCurrentLogic
 * @author: lijunqi
 * @description: 获取当前认证进度逻辑层
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

// GetVerifyCurrentLogic 获取当前认证进度逻辑处理器
type GetVerifyCurrentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewGetVerifyCurrentLogic 创建获取当前认证进度逻辑实例
func NewGetVerifyCurrentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetVerifyCurrentLogic {
	return &GetVerifyCurrentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetVerifyCurrent 获取当前认证进度
// 业务逻辑:
//   - 返回用户当前的认证状态、可执行的操作、OCR识别数据等
//   - 用于前端进入认证页面时展示当前状态
func (l *GetVerifyCurrentLogic) GetVerifyCurrent(in *pb.GetVerifyCurrentReq) (*pb.GetVerifyCurrentResp, error) {
	// 1. 参数校验
	if in.UserId <= 0 {
		l.Errorf("GetVerifyCurrent 参数错误: userId=%d", in.UserId)
		return nil, errorx.ErrInvalidParams("用户ID无效")
	}

	// 2. 查询认证记录
	verification, err := l.svcCtx.StudentVerificationModel.FindByUserID(l.ctx, in.UserId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 没有认证记录，返回初始状态
			l.Infof("GetVerifyCurrent 无认证记录: userId=%d", in.UserId)
			return &pb.GetVerifyCurrentResp{
				HasRecord:  false,
				VerifyId:   0,
				Status:     int32(constants.VerifyStatusInit),
				StatusDesc: constants.GetVerifyStatusName(constants.VerifyStatusInit),
				CanApply:   true,
				CanConfirm: false,
				CanCancel:  false,
				NeedAction: constants.VerifyActionApply,
			}, nil
		}
		l.Errorf("GetVerifyCurrent 查询失败: userId=%d, err=%v", in.UserId, err)
		return nil, errorx.ErrDBError(err)
	}

	// 3. 构建响应
	resp := &pb.GetVerifyCurrentResp{
		HasRecord:    true,
		VerifyId:     verification.ID,
		Status:       int32(verification.Status),
		StatusDesc:   verification.GetStatusName(),
		CanApply:     verification.CanApply(),
		CanConfirm:   verification.CanConfirm(),
		CanCancel:    verification.CanCancel(),
		NeedAction:   verification.GetNeedAction(),
		RejectReason: verification.RejectReason,
		CreatedAt:    verification.CreatedAt.Unix(),
		UpdatedAt:    verification.UpdatedAt.Unix(),
	}

	// 4. 待确认或已通过状态时，返回OCR识别数据
	if ShouldReturnOcrData(verification.Status) {
		resp.VerifyData = BuildVerifyOcrDataFromModel(verification)
	}

	l.Infof("GetVerifyCurrent 查询成功: userId=%d, status=%d, action=%s",
		in.UserId, verification.Status, resp.NeedAction)

	return resp, nil
}
