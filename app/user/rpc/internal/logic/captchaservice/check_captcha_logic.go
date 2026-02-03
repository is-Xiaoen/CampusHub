package captchaservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckCaptchaLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCheckCaptchaLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckCaptchaLogic {
	return &CheckCaptchaLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CheckCaptchaLogic) CheckCaptcha(in *pb.CheckCaptchaReq) (*pb.CheckCaptchaResponse, error) {
	// todo: add your logic here and delete this line

	return &pb.CheckCaptchaResponse{}, nil
}
