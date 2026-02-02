package tagservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAllTagsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAllTagsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAllTagsLogic {
	return &GetAllTagsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetAllTagsLogic) GetAllTags(in *pb.GetAllTagsReq) (*pb.GetAllTagsResp, error) {
	// todo: add your logic here and delete this line

	return &pb.GetAllTagsResp{}, nil
}
