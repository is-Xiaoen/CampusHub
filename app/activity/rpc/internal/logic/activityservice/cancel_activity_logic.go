package activityservicelogic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelActivityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCancelActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelActivityLogic {
	return &CancelActivityLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CancelActivityLogic) CancelActivity(in *activity.CancelActivityReq) (*activity.CancelActivityResp, error) {
	// todo: add your logic here and delete this line

	return &activity.CancelActivityResp{}, nil
}
