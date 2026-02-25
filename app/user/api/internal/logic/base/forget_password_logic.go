package base

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/client/userbasicservice"
	"activity-platform/common/errorx"

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
	if req.QqEmail == "" {
		return nil, errorx.ErrInvalidParams("qq_email不能为空")
	}

	rpcResp, err := l.svcCtx.UserBasicServiceRpc.ForgetPassword(l.ctx, &userbasicservice.ForgetPasswordReq{
		QqEmail:     req.QqEmail,
		QqCode:      req.QqCode,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		return nil, err
	}

	return &types.ForgetPasswordResp{
		Success: rpcResp.Success,
	}, nil
}
