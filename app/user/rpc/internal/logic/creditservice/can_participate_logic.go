package creditservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CanParticipateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCanParticipateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CanParticipateLogic {
	return &CanParticipateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CanParticipate 校验是否允许报名
func (l *CanParticipateLogic) CanParticipate(in *pb.CanParticipateReq) (*pb.CanParticipateResp, error) {
	// todo: add your logic here and delete this line

	return &pb.CanParticipateResp{}, nil
}
