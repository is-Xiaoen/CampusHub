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
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func (l *GetNotificationsLogic) GetNotifications(req *types.GetNotificationsReq) (resp *types.GetNotificationsData, err error) {
	// 调用 RPC 服务获取通知列表
	rpcResp, err := l.svcCtx.ChatRpc.GetNotifications(l.ctx, &chat.GetNotificationsReq{
		UserId:   strconv.FormatInt(req.UserId, 10),
		IsRead:   -1, // -1 表示查询全部
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		l.Errorf("调用 RPC 获取通知列表失败: %v", err)
		// 处理 gRPC 错误
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return nil, errorx.New(errorx.CodeNotificationNotFound)
			default:
				return nil, errorx.NewWithMessage(errorx.CodeRPCError, "获取通知列表失败")
			}
		}
		return nil, errorx.NewWithMessage(errorx.CodeInternalError, "获取通知列表失败")
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

	return &types.GetNotificationsData{
		Total:         rpcResp.Total,
		UnreadCount:   rpcResp.UnreadCount,
		Notifications: notifications,
		Page:          req.Page,
		PageSize:      req.PageSize,
	}, nil
}

// formatTimestamp 将时间戳转换为字符串格式
func formatTimestamp(timestamp int64) string {
	if timestamp == 0 {
		return ""
	}
	return fmt.Sprintf("%d", timestamp)
}
