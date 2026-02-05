package tagservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateUserTagLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateUserTagLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserTagLogic {
	return &UpdateUserTagLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 修改用户兴趣
func (l *UpdateUserTagLogic) UpdateUserTag(in *pb.UpdateUserTagReq) (*pb.UpdateUserTagResponse, error) {
	// todo: add your logic here and delete this line

	return &pb.UpdateUserTagResponse{}, nil
}
