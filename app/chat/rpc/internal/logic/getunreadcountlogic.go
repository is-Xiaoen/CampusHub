package logic

import (
	"context"

	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetUnreadCountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUnreadCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUnreadCountLogic {
	return &GetUnreadCountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetUnreadCount 获取未读通知数量
func (l *GetUnreadCountLogic) GetUnreadCount(in *chat.GetUnreadCountReq) (*chat.GetUnreadCountResp, error) {
	// 1. 参数验证
	if in.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	// 2. 查询未读数量
	count, err := l.svcCtx.NotificationModel.GetUnreadCount(l.ctx, in.UserId)
	if err != nil {
		l.Errorf("查询未读通知数量失败: %v", err)
		return nil, status.Error(codes.Internal, "查询未读通知数量失败")
	}

	// 3. 返回结果
	return &chat.GetUnreadCountResp{
		UnreadCount: int32(count),
	}, nil
}
