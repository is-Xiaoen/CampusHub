// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package ticket

import (
	"context"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelActivitiesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 取消报名活动
func NewCancelActivitiesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelActivitiesLogic {
	return &CancelActivitiesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CancelActivitiesLogic) CancelActivities(req *types.CancelActivityRequest) (resp *types.CancelActivityResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
