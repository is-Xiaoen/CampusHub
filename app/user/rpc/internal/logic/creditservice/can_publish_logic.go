package creditservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CanPublishLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCanPublishLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CanPublishLogic {
	return &CanPublishLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CanPublish 校验是否允许发布活动
func (l *CanPublishLogic) CanPublish(in *pb.CanPublishReq) (*pb.CanPublishResp, error) {
	// todo: add your logic here and delete this line

	return &pb.CanPublishResp{}, nil
}
