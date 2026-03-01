package logic

import (
	"context"
	"encoding/json"
	"time"

	"activity-platform/app/chat/model"
	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/rpc/internal/svc"
	"activity-platform/common/messaging"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CreateNotificationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateNotificationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateNotificationLogic {
	return &CreateNotificationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CreateNotification 创建系统通知
func (l *CreateNotificationLogic) CreateNotification(in *chat.CreateNotificationReq) (*chat.CreateNotificationResp, error) {
	// 1. 参数验证
	if in.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}
	if in.Type == "" {
		return nil, status.Error(codes.InvalidArgument, "通知类型不能为空")
	}
	if in.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "通知标题不能为空")
	}
	if in.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "通知内容不能为空")
	}

	// 2. 生成通知ID
	notificationID := uuid.New().String()

	// 3. 创建通知记录
	notification := &model.Notification{
		NotificationID: notificationID,
		UserID:         in.UserId,
		Type:           in.Type,
		Title:          in.Title,
		Content:        in.Content,
		IsRead:         0, // 0-未读
		CreatedAt:      time.Now(),
	}

	if err := l.svcCtx.NotificationModel.Insert(l.ctx, notification); err != nil {
		l.Errorf("创建通知失败: %v", err)
		return nil, status.Error(codes.Internal, "创建通知失败")
	}

	// 5. 异步推送实时通知（best-effort，失败不影响主流程）
	go func() {
		event := messaging.NotificationPushEventData{
			UserID:         in.UserId,
			NotificationID: notificationID,
			Type:           in.Type,
			Title:          in.Title,
			Content:        in.Content,
			Timestamp:      time.Now().Unix(),
		}
		payload, err := json.Marshal(event)
		if err != nil {
			l.Errorf("序列化通知推送事件失败: %v", err)
			return
		}
		if err := l.svcCtx.MsgClient.Publish(context.Background(), messaging.TopicNotificationPush, payload); err != nil {
			l.Errorf("发布通知推送事件失败: %v", err)
		}
	}()

	return &chat.CreateNotificationResp{
		NotificationId: notificationID,
	}, nil
}
