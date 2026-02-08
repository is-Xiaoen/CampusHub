package logic

import (
	"context"

	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MarkNotificationReadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMarkNotificationReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkNotificationReadLogic {
	return &MarkNotificationReadLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// MarkNotificationRead 标记通知已读
func (l *MarkNotificationReadLogic) MarkNotificationRead(in *chat.MarkNotificationReadReq) (*chat.MarkNotificationReadResp, error) {
	// 1. 参数验证
	if in.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}
	if len(in.NotificationIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "通知ID列表不能为空")
	}

	// 2. 批量标记已读
	updatedCount, err := l.svcCtx.NotificationModel.MarkAsRead(l.ctx, in.UserId, in.NotificationIds)
	if err != nil {
		l.Errorf("标记通知已读失败: %v", err)
		return nil, status.Error(codes.Internal, "标记通知已读失败")
	}

	// 3. 返回结果
	return &chat.MarkNotificationReadResp{
		Success:      true,
		UpdatedCount: int32(updatedCount),
	}, nil
}
