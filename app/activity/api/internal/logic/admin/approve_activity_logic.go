package admin

import (
	"context"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ApproveActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 审核通过
func NewApproveActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApproveActivityLogic {
	return &ApproveActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ApproveActivityLogic) ApproveActivity(req *types.ApproveActivityReq) (resp *types.ApproveActivityResp, err error) {
	// MVP 版本：无后台管理系统，审核功能暂未开放
	// 活动创建后直接发布，无需审核
	return nil, errorx.ErrInvalidParams("MVP 版本暂未开放审核功能，活动创建后自动发布")
}
