package activityservicelogic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTicketDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTicketDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTicketDetailLogic {
	return &GetTicketDetailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetTicketDetail 获取票券详情
func (l *GetTicketDetailLogic) GetTicketDetail(in *activity.GetTicketDetailRequest) (*activity.GetTicketDetailResponse, error) {
	// todo: add your logic here and delete this line

	return &activity.GetTicketDetailResponse{}, nil
}
