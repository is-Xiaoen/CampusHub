// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package activity

import (
	"context"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 取消活动
func NewCancelActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelActivityLogic {
	return &CancelActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CancelActivityLogic) CancelActivity(req *types.CancelActivityReq) (resp *types.CancelActivityResp, err error) {
	// todo: add your logic here and delete this line

	return
}
