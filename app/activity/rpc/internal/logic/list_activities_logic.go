package logic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListActivitiesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListActivitiesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListActivitiesLogic {
	return &ListActivitiesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListActivitiesLogic) ListActivities(in *activity.ListActivitiesReq) (*activity.ListActivitiesResp, error) {
	// todo: add your logic here and delete this line

	return &activity.ListActivitiesResp{}, nil
}
