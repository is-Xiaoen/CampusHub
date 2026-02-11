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

// ActivityCancelledConsumer 活动取消事件消费者
type ActivityCancelledConsumer struct {
	chatRpc chat.ChatServiceClient
	logger  logx.Logger
}

func NewActivityCancelledConsumer(chatRpc chat.ChatServiceClient) *ActivityCancelledConsumer {
	return &ActivityCancelledConsumer{
		chatRpc: chatRpc,
		logger:  logx.WithContext(context.Background()),
	}
}

func (c *ActivityCancelledConsumer) Subscribe(msgClient *messaging.Client) {
	msgClient.Subscribe("activity.cancelled", "chat-auto-disband-group", c.handleActivityCancelled)
	c.logger.Info("已订阅 activity.cancelled 事件")
}

func (c *ActivityCancelledConsumer) handleActivityCancelled(msg *message.Message) error {
	ctx := msg.Context()

	var event ActivityCancelledEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		c.logger.Errorf("解析活动取消事件失败: %v", err)
		return messaging.NewNonRetryableError(fmt.Errorf("解析事件失败: %w", err))
	}

	c.logger.Infof("收到活动取消事件: activity_id=%d, cancelled_by=%d, reason=%s",
		event.ActivityID, event.CancelledBy, event.Reason)

	// 1. 查询活动对应的群聊
	groupResp, err := c.chatRpc.GetGroupByActivityId(ctx, &chat.GetGroupByActivityIdReq{
		ActivityId: event.ActivityID,
	})
	if err != nil {
		c.logger.Errorf("查询群聊信息失败: %v, activity_id=%d", err, event.ActivityID)
		return messaging.NewRetryableError(fmt.Errorf("查询群聊失败: %w", err))
	}

	// 2. 解散群聊
	_, err = c.chatRpc.DisbandGroup(ctx, &chat.DisbandGroupReq{
		GroupId:    groupResp.Group.GroupId,
		OperatorId: event.CancelledBy,
	})
	if err != nil {
		c.logger.Errorf("自动解散群聊失败: %v, group_id=%s, activity_id=%d",
			err, groupResp.Group.GroupId, event.ActivityID)
		return messaging.NewRetryableError(fmt.Errorf("解散群聊失败: %w", err))
	}

	c.logger.Infof("自动解散群聊成功: group_id=%s, activity_id=%d", groupResp.Group.GroupId, event.ActivityID)

	// 3. 给所有群成员发送通知
	c.notifyGroupMembers(ctx, groupResp.Group.GroupId, groupResp.Group.Name, event.Reason)

	return nil
}

// notifyGroupMembers 给所有群成员发送活动取消通知
func (c *ActivityCancelledConsumer) notifyGroupMembers(ctx context.Context, groupId, groupName, reason string) {
	// 1. 查询所有群成员（分页查询）
	page := int32(1)
	pageSize := int32(100) // 每次查询100个成员
	totalNotified := 0
	failedCount := 0

	for {
		// 查询当前页的群成员
		membersResp, err := c.chatRpc.GetGroupMembers(ctx, &chat.GetGroupMembersReq{
			GroupId:  groupId,
			Page:     page,
			PageSize: pageSize,
		})
		if err != nil {
			c.logger.Errorf("查询群成员失败: %v, group_id=%s, page=%d", err, groupId, page)
			break
		}

		// 如果没有成员了，退出循环
		if len(membersResp.Members) == 0 {
			break
		}

		// 2. 给每个成员发送通知
		for _, member := range membersResp.Members {
			// 构造通知内容
			title := "活动已取消"
			content := fmt.Sprintf("您参与的活动群聊「%s」已解散", groupName)
			if reason != "" {
				content = fmt.Sprintf("%s。取消原因：%s", content, reason)
			}

			// 发送通知
			_, err := c.chatRpc.CreateNotification(ctx, &chat.CreateNotificationReq{
				UserId:  member.UserId,
				Type:    "activity_cancelled",
				Title:   title,
				Content: content,
			})
			if err != nil {
				c.logger.Errorf("发送活动取消通知失败: user_id=%d, err=%v", member.UserId, err)
				failedCount++
			} else {
				totalNotified++
			}
		}

		// 3. 检查是否还有更多成员
		if len(membersResp.Members) < int(pageSize) {
			// 最后一页，退出循环
			break
		}

		// 继续查询下一页
		page++
	}

	c.logger.Infof("活动取消通知发送完成: group_id=%s, 成功=%d, 失败=%d", groupId, totalNotified, failedCount)
}
