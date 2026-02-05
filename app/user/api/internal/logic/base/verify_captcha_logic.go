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

type VerifyCaptchaLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 校验验证码
func NewVerifyCaptchaLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VerifyCaptchaLogic {
	return &VerifyCaptchaLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *VerifyCaptchaLogic) VerifyCaptcha(req *types.VerifyCaptchaReq) (resp *types.VerifyCaptchaResp, err error) {
	rpcResp, err := l.svcCtx.CaptchaServiceRpc.CheckCaptcha(l.ctx, &captchaservice.CheckCaptchaReq{
		LotNumber:     req.LotNumber,
		CaptchaOutput: req.CaptchaOutput,
		PassToken:     req.PassToken,
		GenTime:       req.GenTime,
	})
	if err != nil {
		return nil, err
	}

	return &types.VerifyCaptchaResp{
		IsValid: rpcResp.Result == "success",
	}, nil
}
