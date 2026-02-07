// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package notification

import (
	"context"

	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/app/chat/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type MarkNotificationReadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 标记已读
func NewMarkNotificationReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkNotificationReadLogic {
	return &MarkNotificationReadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MarkNotificationReadLogic) MarkNotificationRead(req *types.MarkNotificationReadReq) (resp *types.MarkNotificationReadResp, err error) {
	// todo: add your logic here and delete this line

	return
}
