// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package notification

import (
	"context"

	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/app/chat/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetNotificationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询通知列表
func NewGetNotificationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetNotificationsLogic {
	return &GetNotificationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetNotificationsLogic) GetNotifications(req *types.GetNotificationsReq) (resp *types.GetNotificationsResp, err error) {
	// todo: add your logic here and delete this line

	return
}
