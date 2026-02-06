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

// NewActivityMemberLeftConsumer 创建用户取消报名事件消费者
func NewActivityMemberLeftConsumer(chatRpc chat.ChatServiceClient) *ActivityMemberLeftConsumer {
	return &ActivityMemberLeftConsumer{
		chatRpc: chatRpc,
		logger:  logx.WithContext(context.Background()),
	}
}

// Subscribe 订阅用户取消报名事件
func (c *ActivityMemberLeftConsumer) Subscribe(msgClient *messaging.Client) {
	msgClient.Subscribe("activity.member.left", "chat-auto-remove-member", c.handleMemberLeft)
	c.logger.Info("已订阅 activity.member.left 事件")
}

// handleMemberLeft 处理用户取消报名事件
func (c *ActivityMemberLeftConsumer) handleMemberLeft(msg *message.Message) error {
	ctx := msg.Context()

	// 1. 解析事件数据
	var event ActivityMemberLeftEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		c.logger.Errorf("解析用户取消报名事件失败: %v, payload: %s", err, string(msg.Payload))
		// 数据格式错误，不可重试
		return messaging.NewNonRetryableError(fmt.Errorf("解析事件失败: %w", err))
	}

	c.logger.Infof("收到用户取消报名事件: activity_id=%s, user_id=%s",
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

	// 3. 从群聊中移除用户
	_, err = c.chatRpc.RemoveGroupMember(ctx, &chat.RemoveGroupMemberReq{
		GroupId:    groupResp.Group.GroupId,
		UserId:     event.UserID,
		OperatorId: "system", // 系统操作
	})

	if err != nil {
		c.logger.Errorf("自动移除群成员失败: %v, group_id=%s, user_id=%s",
			err, groupResp.Group.GroupId, event.UserID)
		// 移除失败，可重试
		return messaging.NewRetryableError(fmt.Errorf("移除群成员失败: %w", err))
	}

	c.logger.Infof("自动移除群成员成功: group_id=%s, user_id=%s",
		groupResp.Group.GroupId, event.UserID)

	// 4. 发送系统通知给用户
	_, err = c.chatRpc.CreateNotification(ctx, &chat.CreateNotificationReq{
		UserId:  event.UserID,
		Type:    "group_left",
		Title:   "已退出群聊",
		Content: fmt.Sprintf("您已退出活动群聊「%s」", groupResp.Group.Name),
	})

	if err != nil {
		// 通知发送失败不影响主流程，只记录日志
		c.logger.Errorf("发送退出群聊通知失败: %v", err)
	}

	return nil
}
