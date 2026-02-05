package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"activity-platform/app/chat/rpc/chat"
	"activity-platform/common/messaging"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/zeromicro/go-zero/core/logx"
)

// ActivityCreatedConsumer 活动创建事件消费者
type ActivityCreatedConsumer struct {
	chatRpc chat.ChatServiceClient
	logger  logx.Logger
}

// NewActivityCreatedConsumer 创建活动创建事件消费者
func NewActivityCreatedConsumer(chatRpc chat.ChatServiceClient) *ActivityCreatedConsumer {
	return &ActivityCreatedConsumer{
		chatRpc: chatRpc,
		logger:  logx.WithContext(context.Background()),
	}
}

// Subscribe 订阅活动创建事件
func (c *ActivityCreatedConsumer) Subscribe(msgClient *messaging.Client) {
	msgClient.Subscribe("activity.created", "chat-auto-create-group", c.handleActivityCreated)
	c.logger.Info("已订阅 activity.created 事件")
}

// handleActivityCreated 处理活动创建事件
func (c *ActivityCreatedConsumer) handleActivityCreated(msg *message.Message) error {
	ctx := msg.Context()

	// 1. 解析事件数据
	var event ActivityCreatedEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		c.logger.Errorf("解析活动创建事件失败: %v, payload: %s", err, string(msg.Payload))
		// 数据格式错误，不可重试
		return messaging.NewNonRetryableError(fmt.Errorf("解析事件失败: %w", err))
	}

	c.logger.Infof("收到活动创建事件: activity_id=%s, creator_id=%s, title=%s",
		event.ActivityID, event.CreatorID, event.Title)

	// 2. 调用 RPC 创建群聊
	resp, err := c.chatRpc.CreateGroup(ctx, &chat.CreateGroupReq{
		ActivityId: event.ActivityID,
		Name:       event.Title, // 使用活动标题作为群聊名称
		OwnerId:    event.CreatorID,
		MaxMembers: 500, // 默认最大成员数
	})

	if err != nil {
		c.logger.Errorf("自动创建群聊失败: %v, activity_id=%s", err, event.ActivityID)
		// RPC 调用失败，可重试
		return messaging.NewRetryableError(fmt.Errorf("创建群聊失败: %w", err))
	}

	c.logger.Infof("自动创建群聊成功: group_id=%s, activity_id=%s", resp.GroupId, event.ActivityID)

	// 3. 可选：发送系统通知给创建者
	_, err = c.chatRpc.CreateNotification(ctx, &chat.CreateNotificationReq{
		UserId:  event.CreatorID,
		Type:    "group_created",
		Title:   "群聊创建成功",
		Content: fmt.Sprintf("您的活动「%s」已自动创建群聊", event.Title),
	})

	if err != nil {
		// 通知发送失败不影响主流程，只记录日志
		c.logger.Errorf("发送群聊创建通知失败: %v", err)
	}

	return nil
}
