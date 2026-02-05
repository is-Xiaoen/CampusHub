// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package base

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/client/captchaservice"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCaptchaLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取验证码配置
func NewGetCaptchaLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCaptchaLogic {
	return &GetCaptchaLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCaptchaLogic) GetCaptcha() (resp *types.GetCaptchaResp, err error) {
	rpcResp, err := l.svcCtx.CaptchaServiceRpc.GetCaptchaConfig(l.ctx, &captchaservice.GetCaptchaConfigReq{})
	if err != nil {
		return nil, err
	}

	return &types.GetCaptchaResp{
		CaptchaId: rpcResp.CaptchaId,
	}, nil
}
