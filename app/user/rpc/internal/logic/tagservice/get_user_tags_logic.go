package tagservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserTagsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserTagsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserTagsLogic {
	return &GetUserTagsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetUserTagsLogic) GetUserTags(in *pb.GetUserTagsReq) (*pb.GetUserTagsResponse, error) {
	// todo: add your logic here and delete this line

	return &pb.GetUserTagsResponse{}, nil
}
