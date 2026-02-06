package activityservicelogic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserPublishedActivitiesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserPublishedActivitiesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserPublishedActivitiesLogic {
	return &GetUserPublishedActivitiesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取用户已发布的活动列表（User 服务调用，用于展示用户主页）
func (l *GetUserPublishedActivitiesLogic) GetUserPublishedActivities(in *activity.GetUserPublishedActivitiesReq) (*activity.GetUserPublishedActivitiesResp, error) {
	// todo: add your logic here and delete this line

	return &activity.GetUserPublishedActivitiesResp{}, nil
}
