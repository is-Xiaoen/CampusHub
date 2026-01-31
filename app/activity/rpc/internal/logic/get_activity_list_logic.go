package logic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetActivityListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetActivityListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetActivityListLogic {
	return &GetActivityListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetActivityList 获取待参加/已参加活动信息列表
func (l *GetActivityListLogic) GetActivityList(in *activity.GetActivityListRequest) (*activity.GetActivityListResponse, error) {
	// todo: add your logic here and delete this line

	return &activity.GetActivityListResponse{}, nil
}
