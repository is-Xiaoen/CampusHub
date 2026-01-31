package logic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelActivitiesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCancelActivitiesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelActivitiesLogic {
	return &CancelActivitiesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CancelActivities 取消报名活动
func (l *CancelActivitiesLogic) CancelActivities(in *activity.CancelActivityRequest) (*activity.CancelActivityResponse, error) {
	// todo: add your logic here and delete this line

	return &activity.CancelActivityResponse{}, nil
}
