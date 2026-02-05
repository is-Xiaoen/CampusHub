package userbasicservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetGroupUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetGroupUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupUserLogic {
	return &GetGroupUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 批量获取群聊用户的信息
func (l *GetGroupUserLogic) GetGroupUser(in *pb.GetGroupUserReq) (*pb.GetGroupUserResponse, error) {
	// todo: add your logic here and delete this line

	return &pb.GetGroupUserResponse{}, nil
}
