package base

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/client/userbasicservice"
	ctxUtils "activity-platform/common/utils/context"

	"github.com/zeromicro/go-zero/core/logx"
)

type ForgetPasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 忘记密码
func NewForgetPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ForgetPasswordLogic {
	return &ForgetPasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ForgetPasswordLogic) ForgetPassword(req *types.ForgetPasswordReq) (resp *types.ForgetPasswordResp, err error) {
	// 从上下文获取 userId（go-zero jwt:Auth 自动注入）
	userId, err := ctxUtils.GetUserIdFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}

	// 调用 RPC 层忘记密码
	rpcResp, err := l.svcCtx.UserBasicServiceRpc.ForgetPassword(l.ctx, &userbasicservice.ForgetPasswordReq{
		QqCode:      req.QqCode,
		NewPassword: req.NewPassword,
		UserId:      userId,
	})
	if err != nil {
		return nil, err
	}

	return &types.ForgetPasswordResp{
		Success: rpcResp.Success,
	}, nil
}
