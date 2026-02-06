package logic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/common/errorx"

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
	// MVP 版本无后台管理系统，审批功能暂不支持
	return nil, errorx.NewWithMessage(errorx.CodeServiceUnavailable, "审批功能暂未开放")
}
