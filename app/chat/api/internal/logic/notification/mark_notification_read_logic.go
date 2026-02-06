// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package notification

import (
	"context"
	"fmt"
	"strconv"

	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/app/chat/api/internal/types"
	"activity-platform/app/chat/rpc/chat"

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
	// 调用 RPC 服务标记通知已读
	rpcResp, err := l.svcCtx.ChatRpc.MarkNotificationRead(l.ctx, &chat.MarkNotificationReadReq{
		UserId:          strconv.FormatInt(req.UserId, 10),
		NotificationIds: req.NotificationIds,
	})
	if err != nil {
		l.Errorf("调用 RPC 标记通知已读失败: %v", err)
		return &types.MarkNotificationReadResp{
			Code:    500,
			Message: fmt.Sprintf("标记已读失败: %v", err),
			Data: types.MarkNotificationReadData{
				UpdatedCount: 0,
			},
		}, nil
	}

	return &types.MarkNotificationReadResp{
		Code:    0,
		Message: "标记成功",
		Data: types.MarkNotificationReadData{
			UpdatedCount: rpcResp.UpdatedCount,
		},
	}, nil
}
