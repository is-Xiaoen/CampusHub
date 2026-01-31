// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package activity

import (
	"context"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type MyCreatedActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 我创建的活动
func NewMyCreatedActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MyCreatedActivityLogic {
	return &MyCreatedActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MyCreatedActivityLogic) MyCreatedActivity(req *types.MyActivityReq) (resp *types.MyActivityResp, err error) {
	// todo: add your logic here and delete this line

	return
}
