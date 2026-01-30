package creditservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCreditInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetCreditInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCreditInfoLogic {
	return &GetCreditInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetCreditInfo 获取用户信用信息
func (l *GetCreditInfoLogic) GetCreditInfo(in *pb.GetCreditInfoReq) (*pb.GetCreditInfoResp, error) {
	// todo: add your logic here and delete this line

	return &pb.GetCreditInfoResp{}, nil
}
