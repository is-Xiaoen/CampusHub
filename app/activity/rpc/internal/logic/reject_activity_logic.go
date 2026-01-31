package logic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type RejectActivityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRejectActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RejectActivityLogic {
	return &RejectActivityLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RejectActivityLogic) RejectActivity(in *activity.RejectActivityReq) (*activity.RejectActivityResp, error) {
	// todo: add your logic here and delete this line

	return &activity.RejectActivityResp{}, nil
}
