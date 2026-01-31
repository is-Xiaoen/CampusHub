// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package public

import (
	"context"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type IncrViewCountLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 增加浏览量
func NewIncrViewCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IncrViewCountLogic {
	return &IncrViewCountLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *IncrViewCountLogic) IncrViewCount(req *types.IncrViewCountReq) (resp *types.IncrViewCountResp, err error) {
	// todo: add your logic here and delete this line

	return
}
