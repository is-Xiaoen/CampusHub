// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package ticket

import (
	"context"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 报名活动
func NewRegisterActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterActivityLogic {
	return &RegisterActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterActivityLogic) RegisterActivity(req *types.RegisterActivityRequest) (resp *types.RegisterActivityResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
