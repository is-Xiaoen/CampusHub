/**
 * @projectName: CampusHub
 * @package: verify
 * @className: CancelVerifyLogic
 * @author: lijunqi
 * @description: 取消认证申请业务逻辑
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

// CancelVerifyLogic 取消认证申请逻辑处理器
type CancelVerifyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewCancelVerifyLogic 创建取消认证申请逻辑实例
func NewCancelVerifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelVerifyLogic {
	return &CancelVerifyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// CancelVerify 取消认证申请
func (l *CancelVerifyLogic) CancelVerify(req *types.CancelVerifyReq) (resp *types.CancelVerifyResp, err error) {
	// 1. 从 JWT 中获取当前用户ID
	userId := ctxdata.GetUserIDFromCtx(l.ctx)
	if userId <= 0 {
		l.Errorf("CancelVerify 获取用户ID失败")
		return nil, errorx.ErrUnauthorized()
	}

	// 2. 调用 RPC 取消认证申请
	rpcResp, err := l.svcCtx.VerifyServiceRpc.CancelStudentVerify(l.ctx, &verifyservice.CancelStudentVerifyReq{
		UserId:       userId,
		VerifyId:     req.VerifyId,
		CancelReason: req.CancelReason,
	})
	if err != nil {
		l.Errorf("CancelVerify 调用 RPC 失败: userId=%d, verifyId=%d, err=%v",
			userId, req.VerifyId, err)
		return nil, errorx.FromError(err)
	}

	// 3. 转换 RPC 响应为 API 响应
	resp = &types.CancelVerifyResp{
		VerifyId:      rpcResp.VerifyId,
		NewStatus:     rpcResp.NewStatus,
		NewStatusDesc: rpcResp.NewStatusDesc,
	}

	l.Infof("CancelVerify 操作成功: userId=%d, verifyId=%d, newStatus=%d",
		userId, resp.VerifyId, resp.NewStatus)

	return resp, nil
}
