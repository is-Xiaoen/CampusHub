package logic

import (
	"context"

	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MarkAllReadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMarkAllReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkAllReadLogic {
	return &MarkAllReadLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// MarkAllRead 全部标记已读
func (l *MarkAllReadLogic) MarkAllRead(in *chat.MarkAllReadReq) (*chat.MarkAllReadResp, error) {
	// 1. 参数验证
	if in.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	// 2. 标记所有通知为已读
	affectedCount, err := l.svcCtx.NotificationModel.MarkAllAsRead(l.ctx, in.UserId)
	if err != nil {
		l.Errorf("全部标记已读失败: %v", err)
		return nil, status.Error(codes.Internal, "全部标记已读失败")
	}

	// 3. 返回结果
	return &chat.MarkAllReadResp{
		Success:       true,
		AffectedCount: int32(affectedCount),
	}, nil
}
