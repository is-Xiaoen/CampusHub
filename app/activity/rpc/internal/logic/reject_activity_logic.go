package logic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/common/errorx"

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
	// MVP 版本无后台管理系统，拒绝功能暂不支持
	return nil, errorx.NewWithMessage(errorx.CodeServiceUnavailable, "拒绝功能暂未开放")
}
