package activityservicelogic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetHotActivitiesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetHotActivitiesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetHotActivitiesLogic {
	return &GetHotActivitiesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetHotActivitiesLogic) GetHotActivities(in *activity.GetHotActivitiesReq) (*activity.GetHotActivitiesResp, error) {
	// todo: add your logic here and delete this line

	return &activity.GetHotActivitiesResp{}, nil
}
