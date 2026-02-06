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

// ActivityMemberJoinedConsumer 用户报名成功事件消费者
type ActivityMemberJoinedConsumer struct {
	chatRpc chat.ChatServiceClient
	logger  logx.Logger
}

// NewActivityMemberJoinedConsumer 创建用户报名成功事件消费者
func NewActivityMemberJoinedConsumer(chatRpc chat.ChatServiceClient) *ActivityMemberJoinedConsumer {
	return &ActivityMemberJoinedConsumer{
		chatRpc: chatRpc,
		logger:  logx.WithContext(context.Background()),
	}
}

// Subscribe 订阅用户报名成功事件
func (c *ActivityMemberJoinedConsumer) Subscribe(msgClient *messaging.Client) {
	msgClient.Subscribe("activity.member.joined", "chat-auto-add-member", c.handleMemberJoined)
	c.logger.Info("已订阅 activity.member.joined 事件")
}

// handleMemberJoined 处理用户报名成功事件
func (c *ActivityMemberJoinedConsumer) handleMemberJoined(msg *message.Message) error {
	ctx := msg.Context()

	// 1. 解析事件数据
	var event ActivityMemberJoinedEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		c.logger.Errorf("解析用户报名事件失败: %v, payload: %s", err, string(msg.Payload))
		// 数据格式错误，不可重试
		return messaging.NewNonRetryableError(fmt.Errorf("解析事件失败: %w", err))
	}

	c.logger.Infof("收到用户报名事件: activity_id=%s, user_id=%s",
		event.ActivityID, event.UserID)

	// 2. 查询活动对应的群聊
	groupResp, err := c.chatRpc.GetGroupByActivityId(ctx, &chat.GetGroupByActivityIdReq{
		ActivityId: event.ActivityID,
	})

	if err != nil {
		c.logger.Errorf("查询群聊信息失败: %v, activity_id=%s", err, event.ActivityID)
		// 查询失败，可重试
		return messaging.NewRetryableError(fmt.Errorf("查询群聊失败: %w", err))
	}

	// 3. 添加用户到群聊
	_, err = c.chatRpc.AddGroupMember(ctx, &chat.AddGroupMemberReq{
		GroupId: groupResp.Group.GroupId,
		UserId:  event.UserID,
		Role:    1, // 1-普通成员
	})

	if err != nil {
		c.logger.Errorf("自动添加群成员失败: %v, group_id=%s, user_id=%s",
			err, groupResp.Group.GroupId, event.UserID)
		// 添加失败，可重试
		return messaging.NewRetryableError(fmt.Errorf("添加群成员失败: %w", err))
	}

	c.logger.Infof("自动添加群成员成功: group_id=%s, user_id=%s",
		groupResp.Group.GroupId, event.UserID)

	// 4. 发送系统通知给用户
	_, err = c.chatRpc.CreateNotification(ctx, &chat.CreateNotificationReq{
		UserId:  event.UserID,
		Type:    "group_joined",
		Title:   "加入群聊成功",
		Content: fmt.Sprintf("您已成功加入活动群聊「%s」", groupResp.Group.Name),
	})

	if err != nil {
		// 通知发送失败不影响主流程，只记录日志
		c.logger.Errorf("发送加入群聊通知失败: %v", err)
	}

	return nil
}
