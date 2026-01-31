// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package activity

import (
	"context"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除活动
func NewDeleteActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteActivityLogic {
	return &DeleteActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteActivityLogic) DeleteActivity(req *types.DeleteActivityReq) (resp *types.DeleteActivityResp, err error) {
	// todo: add your logic here and delete this line

	return
}
