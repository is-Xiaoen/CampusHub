// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package notification

import (
	"context"

	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/app/chat/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type MarkAllReadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 标记全部已读
func NewMarkAllReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkAllReadLogic {
	return &MarkAllReadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MarkAllReadLogic) MarkAllRead(req *types.MarkAllReadReq) (resp *types.MarkAllReadResp, err error) {
	// todo: add your logic here and delete this line

	return
}
