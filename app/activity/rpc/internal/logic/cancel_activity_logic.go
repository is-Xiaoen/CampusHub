package logic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// cancellableStatuses 可取消的活动状态集合
// 设计说明：
// - 草稿/待审核/已发布/进行中 可以取消
// - 已结束(4)：保留历史记录，不允许取消
// - 已拒绝(5)：应该重新编辑提交，不需要取消
// - 已取消(6)：已经是取消状态
var cancellableStatuses = map[int8]bool{
	model.StatusDraft:     true,
	model.StatusPending:   true,
	model.StatusPublished: true,
	model.StatusOngoing:   true,
}

type CancelActivityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCancelActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelActivityLogic {
	return &CancelActivityLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CancelActivityLogic) CancelActivity(in *activity.CancelActivityReq) (*activity.CancelActivityResp, error) {
	// todo: add your logic here and delete this line

	return &activity.CancelActivityResp{}, nil
}
