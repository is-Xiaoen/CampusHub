package logic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTicketListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTicketListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTicketListLogic {
	return &GetTicketListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetTicketList 获取个人票券列表
func (l *GetTicketListLogic) GetTicketList(in *activity.GetTicketListRequest) (*activity.GetTicketListResponse, error) {
	// todo: add your logic here and delete this line

	return &activity.GetTicketListResponse{}, nil
}
