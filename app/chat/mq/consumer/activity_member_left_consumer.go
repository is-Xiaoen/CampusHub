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

// ActivityMemberLeftConsumer 用户取消报名事件消费者
type ActivityMemberLeftConsumer struct {
	chatRpc chat.ChatServiceClient
	logger  logx.Logger
}

func NewActivityMemberLeftConsumer(chatRpc chat.ChatServiceClient) *ActivityMemberLeftConsumer {
	return &ActivityMemberLeftConsumer{
		chatRpc: chatRpc,
		logger:  logx.WithContext(context.Background()),
	}
}

func (c *ActivityMemberLeftConsumer) Subscribe(msgClient *messaging.Client) {
	msgClient.Subscribe("activity.member.left", "chat-auto-remove-member", c.handleMemberLeft)
	c.logger.Info("已订阅 activity.member.left 事件")
}

func (c *ActivityMemberLeftConsumer) handleMemberLeft(msg *message.Message) error {
	ctx := msg.Context()

	var event ActivityMemberLeftEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		c.logger.Errorf("解析用户取消报名事件失败: %v", err)
		return messaging.NewNonRetryableError(fmt.Errorf("解析事件失败: %w", err))
	}

	c.logger.Infof("收到用户取消报名事件: activity_id=%d, user_id=%d", event.ActivityID, event.UserID)

	groupResp, err := c.chatRpc.GetGroupByActivityId(ctx, &chat.GetGroupByActivityIdReq{
		ActivityId: event.ActivityID,
	})
	if err != nil {
		c.logger.Errorf("查询群聊信息失败: %v, activity_id=%d", err, event.ActivityID)
		return messaging.NewRetryableError(fmt.Errorf("查询群聊失败: %w", err))
	}

	_, err = c.chatRpc.RemoveGroupMember(ctx, &chat.RemoveGroupMemberReq{
		GroupId:    groupResp.Group.GroupId,
		UserId:     event.UserID,
		OperatorId: 0, // 系统操作
	})
	if err != nil {
		c.logger.Errorf("自动移除群成员失败: %v, group_id=%s, user_id=%d",
			err, groupResp.Group.GroupId, event.UserID)
		return messaging.NewRetryableError(fmt.Errorf("移除群成员失败: %w", err))
	}

	c.logger.Infof("自动移除群成员成功: group_id=%s, user_id=%d", groupResp.Group.GroupId, event.UserID)

	_, err = c.chatRpc.CreateNotification(ctx, &chat.CreateNotificationReq{
		UserId:  event.UserID,
		Type:    "group_left",
		Title:   "已退出群聊",
		Content: fmt.Sprintf("您已退出活动群聊「%s」", groupResp.Group.Name),
	})
	if err != nil {
		c.logger.Errorf("发送退出群聊通知失败: %v", err)
	}

	return nil
}
