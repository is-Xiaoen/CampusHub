/**
 * @projectName: CampusHub
 * @package: verify
 * @className: ConfirmVerifyLogic
 * @author: lijunqi
 * @description: 用户确认/修改认证信息业务逻辑
 * @date: 2026-02-02
 * @version: 1.0
 */

package verify

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/client/verifyservice"
	"activity-platform/common/ctxdata"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

// ConfirmVerifyLogic 用户确认/修改认证信息逻辑处理器
type ConfirmVerifyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewConfirmVerifyLogic 创建用户确认/修改认证信息逻辑实例
func NewConfirmVerifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConfirmVerifyLogic {
	return &ConfirmVerifyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ConfirmVerify 用户确认/修改认证信息
func (l *ConfirmVerifyLogic) ConfirmVerify(req *types.ConfirmVerifyReq) (resp *types.ConfirmVerifyResp, err error) {
	// 1. 从 JWT 中获取当前用户ID
	userId := ctxdata.GetUserIDFromCtx(l.ctx)
	if userId <= 0 {
		l.Errorf("ConfirmVerify 获取用户ID失败")
		return nil, errorx.ErrUnauthorized()
	}

	// 2. 构建 RPC 请求
	rpcReq := &verifyservice.ConfirmStudentVerifyReq{
		UserId:      userId,
		VerifyId:    req.VerifyId,
		IsConfirmed: req.IsConfirmed,
	}

	// 3. 如果用户提交了修改数据，转换并添加到请求中
	if req.ModifiedData != nil {
		rpcReq.ModifiedData = &verifyservice.VerifyModifiedData{
			RealName:      req.ModifiedData.RealName,
			SchoolName:    req.ModifiedData.SchoolName,
			StudentId:     req.ModifiedData.StudentId,
			Department:    req.ModifiedData.Department,
			AdmissionYear: req.ModifiedData.AdmissionYear,
		}
	}

	// 4. 调用 RPC 确认/修改认证信息
	rpcResp, err := l.svcCtx.VerifyServiceRpc.ConfirmStudentVerify(l.ctx, rpcReq)
	if err != nil {
		l.Errorf("ConfirmVerify 调用 RPC 失败: userId=%d, verifyId=%d, err=%v",
			userId, req.VerifyId, err)
		return nil, errorx.FromError(err)
	}

	// 5. 转换 RPC 响应为 API 响应
	resp = &types.ConfirmVerifyResp{
		VerifyId:      rpcResp.VerifyId,
		NewStatus:     rpcResp.NewStatus,
		NewStatusDesc: rpcResp.NewStatusDesc,
	}

	l.Infof("ConfirmVerify 操作成功: userId=%d, verifyId=%d, isConfirmed=%v, newStatus=%d",
		userId, resp.VerifyId, req.IsConfirmed, resp.NewStatus)

	return resp, nil
}
