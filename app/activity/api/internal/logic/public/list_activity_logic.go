// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package public

import (
	"context"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 活动列表
func NewListActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListActivityLogic {
	return &ListActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListActivityLogic) ListActivity(req *types.ListActivityReq) (resp *types.ListActivityResp, err error) {
	// todo: add your logic here and delete this line

	return
}
