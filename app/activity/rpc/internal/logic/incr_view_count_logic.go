package logic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type IncrViewCountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewIncrViewCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IncrViewCountLogic {
	return &IncrViewCountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ==================== 浏览量接口 ====================
func (l *IncrViewCountLogic) IncrViewCount(in *activity.IncrViewCountReq) (*activity.IncrViewCountResp, error) {
	// todo: add your logic here and delete this line

	return &activity.IncrViewCountResp{}, nil
}
