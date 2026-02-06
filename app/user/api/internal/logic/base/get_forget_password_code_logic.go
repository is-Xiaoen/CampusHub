// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package base

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/client/qqemail"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetForgetPasswordCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取忘记密码QQ验证码
func NewGetForgetPasswordCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetForgetPasswordCodeLogic {
	return &GetForgetPasswordCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetForgetPasswordCodeLogic) GetForgetPasswordCode(req *types.GetForgetPasswordCodeReq) (resp *types.GetCodeResp, err error) {
	_, err = l.svcCtx.QQEmailRpc.SendQQEmail(l.ctx, &qqemail.SendQQEmailReq{
		QqEmail: req.QqEmail,
		Scene:   "forget_password",
	})
	if err != nil {
		return nil, err
	}

	return &types.GetCodeResp{}, nil
}
