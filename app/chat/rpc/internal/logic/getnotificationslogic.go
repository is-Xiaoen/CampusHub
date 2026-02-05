package logic

import (
	"context"

	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetNotificationsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetNotificationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetNotificationsLogic {
	return &GetNotificationsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetNotifications 获取通知列表
func (l *GetNotificationsLogic) GetNotifications(in *chat.GetNotificationsReq) (*chat.GetNotificationsResp, error) {
	// 1. 参数验证
	if in.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	// 设置默认分页参数
	page := in.Page
	if page <= 0 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// 2. 查询通知列表
	notifications, total, err := l.svcCtx.NotificationModel.FindByUserID(l.ctx, in.UserId, in.IsRead, page, pageSize)
	if err != nil {
		l.Errorf("查询通知列表失败: %v", err)
		return nil, status.Error(codes.Internal, "查询通知列表失败")
	}

	// 3. 查询未读数量
	unreadCount, err := l.svcCtx.NotificationModel.GetUnreadCount(l.ctx, in.UserId)
	if err != nil {
		l.Errorf("查询未读数量失败: %v", err)
		// 未读数量查询失败不影响主流程，设置为0
		unreadCount = 0
	}

	// 4. 构造响应
	notificationList := make([]*chat.Notification, 0, len(notifications))
	for _, notification := range notifications {
		notificationList = append(notificationList, &chat.Notification{
			NotificationId: notification.NotificationID,
			UserId:         notification.UserID,
			Type:           notification.Type,
			Title:          notification.Title,
			Content:        notification.Content,
			IsRead:         int32(notification.IsRead),
			CreatedAt:      notification.CreatedAt.Unix(),
		})
	}

	return &chat.GetNotificationsResp{
		Notifications: notificationList,
		Total:         int32(total),
		UnreadCount:   int32(unreadCount),
	}, nil
}
