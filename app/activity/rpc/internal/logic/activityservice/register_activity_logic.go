package activityservicelogic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterActivityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterActivityLogic {
	return &RegisterActivityLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// RegisterActivity 报名活动
func (l *RegisterActivityLogic) RegisterActivity(in *activity.RegisterActivityRequest) (*activity.RegisterActivityResponse, error) {
	// todo: add your logic here and delete this line

	return &activity.RegisterActivityResponse{}, nil
}
