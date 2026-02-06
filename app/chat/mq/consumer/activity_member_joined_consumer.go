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

func NewActivityMemberJoinedConsumer(chatRpc chat.ChatServiceClient) *ActivityMemberJoinedConsumer {
	return &ActivityMemberJoinedConsumer{
		chatRpc: chatRpc,
		logger:  logx.WithContext(context.Background()),
	}
}

func (c *ActivityMemberJoinedConsumer) Subscribe(msgClient *messaging.Client) {
	msgClient.Subscribe("activity.member.joined", "chat-auto-add-member", c.handleMemberJoined)
	c.logger.Info("已订阅 activity.member.joined 事件")
}

func (c *ActivityMemberJoinedConsumer) handleMemberJoined(msg *message.Message) error {
	ctx := msg.Context()

	var event ActivityMemberJoinedEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		c.logger.Errorf("解析用户报名事件失败: %v", err)
		return messaging.NewNonRetryableError(fmt.Errorf("解析事件失败: %w", err))
	}

	c.logger.Infof("收到用户报名事件: activity_id=%s, user_id=%s", event.ActivityID, event.UserID)

	groupResp, err := c.chatRpc.GetGroupByActivityId(ctx, &chat.GetGroupByActivityIdReq{
		ActivityId: event.ActivityID,
	})
	if err != nil {
		c.logger.Errorf("查询群聊信息失败: %v, activity_id=%s", err, event.ActivityID)
		return messaging.NewRetryableError(fmt.Errorf("查询群聊失败: %w", err))
	}

	_, err = c.chatRpc.AddGroupMember(ctx, &chat.AddGroupMemberReq{
		GroupId: groupResp.Group.GroupId,
		UserId:  event.UserID,
		Role:    1,
	})
	if err != nil {
		c.logger.Errorf("自动添加群成员失败: %v, group_id=%s, user_id=%s",
			err, groupResp.Group.GroupId, event.UserID)
		return messaging.NewRetryableError(fmt.Errorf("添加群成员失败: %w", err))
	}

	c.logger.Infof("自动添加群成员成功: group_id=%s, user_id=%s", groupResp.Group.GroupId, event.UserID)

	_, err = c.chatRpc.CreateNotification(ctx, &chat.CreateNotificationReq{
		UserId:  event.UserID,
		Type:    "group_joined",
		Title:   "加入群聊成功",
		Content: fmt.Sprintf("您已成功加入活动群聊「%s」", groupResp.Group.Name),
	})
	if err != nil {
		c.logger.Errorf("发送加入群聊通知失败: %v", err)
	}

	return nil
}
