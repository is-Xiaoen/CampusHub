// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package admin

import (
	"context"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RejectActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 审核拒绝
func NewRejectActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RejectActivityLogic {
	return &RejectActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RejectActivityLogic) RejectActivity(req *types.RejectActivityReq) (resp *types.RejectActivityResp, err error) {
	// todo: add your logic here and delete this line

	return
}
