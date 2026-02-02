// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package admin

import (
	"context"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminListActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 管理员活动列表
func NewAdminListActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminListActivityLogic {
	return &AdminListActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminListActivityLogic) AdminListActivity(req *types.ListActivityReq) (resp *types.ListActivityResp, err error) {
	// todo: add your logic here and delete this line

	return
}
