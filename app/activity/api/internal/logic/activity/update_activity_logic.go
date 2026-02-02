// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package activity

import (
	"context"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新活动
func NewUpdateActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateActivityLogic {
	return &UpdateActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateActivityLogic) UpdateActivity(req *types.UpdateActivityReq) (resp *types.UpdateActivityResp, err error) {
	// todo: add your logic here and delete this line

	return
}
