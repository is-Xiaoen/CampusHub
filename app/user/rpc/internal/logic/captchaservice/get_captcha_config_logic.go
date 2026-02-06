package captchaservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCaptchaConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetCaptchaConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCaptchaConfigLogic {
	return &GetCaptchaConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetCaptchaConfigLogic) GetCaptchaConfig(in *pb.GetCaptchaConfigReq) (*pb.GetCaptchaConfigResponse, error) {
	return &pb.GetCaptchaConfigResponse{
		CaptchaId: l.svcCtx.Config.Captcha.CaptchaId,
	}, nil
}
