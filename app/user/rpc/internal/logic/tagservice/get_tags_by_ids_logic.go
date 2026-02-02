package tagservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTagsByIdsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTagsByIdsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTagsByIdsLogic {
	return &GetTagsByIdsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetTagsByIdsLogic) GetTagsByIds(in *pb.GetTagsByIdsReq) (*pb.GetTagsByIdsResp, error) {
	// todo: add your logic here and delete this line

	return &pb.GetTagsByIdsResp{}, nil
}
