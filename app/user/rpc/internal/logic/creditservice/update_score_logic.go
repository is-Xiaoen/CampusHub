package creditservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateScoreLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateScoreLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateScoreLogic {
	return &UpdateScoreLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// UpdateScore 变更信用分
func (l *UpdateScoreLogic) UpdateScore(in *pb.UpdateScoreReq) (*pb.UpdateScoreResp, error) {
	// todo: add your logic here and delete this line

	return &pb.UpdateScoreResp{}, nil
}
