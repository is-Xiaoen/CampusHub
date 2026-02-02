// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package public

import (
	"context"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetHotActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 热门活动
func NewGetHotActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetHotActivityLogic {
	return &GetHotActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetHotActivityLogic) GetHotActivity(req *types.GetHotActivityReq) (resp *types.GetHotActivityResp, err error) {
	// todo: add your logic here and delete this line

	return
}
