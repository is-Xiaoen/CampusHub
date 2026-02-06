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
	// 调用 RPC 服务获取通知列表
	rpcResp, err := l.svcCtx.ChatRpc.GetNotifications(l.ctx, &chat.GetNotificationsReq{
		UserId:   strconv.FormatInt(req.UserId, 10),
		IsRead:   -1, // -1 表示查询全部
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		l.Errorf("调用 RPC 获取通知列表失败: %v", err)
		return &types.GetNotificationsResp{
			Code:    500,
			Message: fmt.Sprintf("获取通知列表失败: %v", err),
			Data: types.GetNotificationsData{
				Total:         0,
				UnreadCount:   0,
				Notifications: []types.NotificationInfo{},
				Page:          req.Page,
				PageSize:      req.PageSize,
			},
		}, nil
	}

	// 转换通知列表
	notifications := make([]types.NotificationInfo, 0, len(rpcResp.Notifications))
	for _, notif := range rpcResp.Notifications {
		notifications = append(notifications, types.NotificationInfo{
			NotificationId: notif.NotificationId,
			Type:           notif.Type,
			Title:          notif.Title,
			Content:        notif.Content,
			IsRead:         notif.IsRead == 1,
			CreatedAt:      formatTimestamp(notif.CreatedAt),
		})
	}

	return &types.GetNotificationsResp{
		Code:    0,
		Message: "success",
		Data: types.GetNotificationsData{
			Total:         rpcResp.Total,
			UnreadCount:   rpcResp.UnreadCount,
			Notifications: notifications,
			Page:          req.Page,
			PageSize:      req.PageSize,
		},
	}, nil
}

// formatTimestamp 将时间戳转换为字符串格式
func formatTimestamp(timestamp int64) string {
	if timestamp == 0 {
		return ""
	}
	return fmt.Sprintf("%d", timestamp)
}
