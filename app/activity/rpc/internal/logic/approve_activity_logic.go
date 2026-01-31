package logic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ApproveActivityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewApproveActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApproveActivityLogic {
	return &ApproveActivityLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ApproveActivityLogic) ApproveActivity(in *activity.ApproveActivityReq) (*activity.ApproveActivityResp, error) {
	// todo: add your logic here and delete this line

	return &activity.ApproveActivityResp{}, nil
}
