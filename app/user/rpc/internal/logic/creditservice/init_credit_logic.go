package creditservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type InitCreditLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewInitCreditLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InitCreditLogic {
	return &InitCreditLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// InitCredit 初始化信用分
func (l *InitCreditLogic) InitCredit(in *pb.InitCreditReq) (*pb.InitCreditResp, error) {
	// todo: add your logic here and delete this line

	return &pb.InitCreditResp{}, nil
}
