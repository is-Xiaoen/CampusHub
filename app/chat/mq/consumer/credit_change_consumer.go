/**
 * @projectName: CampusHub
 * @package: consumer
 * @className: CreditChangeConsumer
 * @description: 信用分变更消费者（薄层：解析 RawMessage → 调 UserRpc.UpdateScore）
 * @date: 2026-02-06
 * @version: 1.0
 *
 * 消息来源: Activity RPC（签到、爽约等事件）
 * Topic: credit:events（RawMessage 信封格式）
 */

package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/constants"
	"activity-platform/common/messaging"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// eventTypeToChangeType 事件类型 → 信用变更类型
var eventTypeToChangeType = map[string]int32{
	messaging.CreditEventCheckin:     constants.CreditChangeTypeCheckin,
	messaging.CreditEventCancelEarly: constants.CreditChangeTypeCancelEarly,
	messaging.CreditEventCancelLate:  constants.CreditChangeTypeCancelLate,
	messaging.CreditEventNoShow:      constants.CreditChangeTypeNoShow,
	messaging.CreditEventHostSuccess: constants.CreditChangeTypeHostSuccess,
	messaging.CreditEventHostDelete:  constants.CreditChangeTypeHostDelete,
}

// eventTypeToReason 事件类型 → 变更原因描述
var eventTypeToReason = map[string]string{
	messaging.CreditEventCheckin:     "活动签到成功",
	messaging.CreditEventCancelEarly: "提前取消活动报名",
	messaging.CreditEventCancelLate:  "临期取消活动报名（<24h）",
	messaging.CreditEventNoShow:      "活动爽约未签到",
	messaging.CreditEventHostSuccess: "成功举办活动",
	messaging.CreditEventHostDelete:  "删除已有报名的活动",
}

// CreditChangeConsumer 信用分变更消费者
type CreditChangeConsumer struct {
	creditRpc pb.CreditServiceClient
	logger    logx.Logger
}

// NewCreditChangeConsumer 创建信用分变更消费者
func NewCreditChangeConsumer(creditRpc pb.CreditServiceClient) *CreditChangeConsumer {
	return &CreditChangeConsumer{
		creditRpc: creditRpc,
		logger:    logx.WithContext(context.Background()),
	}
}

// Subscribe 订阅信用事件主题
func (c *CreditChangeConsumer) Subscribe(msgClient *messaging.Client) {
	msgClient.Subscribe("credit:events", "credit-event-handler", c.handleCreditEvent)
	c.logger.Info("已订阅 credit:events 事件")
}

// handleCreditEvent 处理信用事件（RawMessage 信封格式）
func (c *CreditChangeConsumer) handleCreditEvent(msg *message.Message) error {
	ctx := msg.Context()
	logger := logx.WithContext(ctx)

	// 1. 解析 RawMessage 信封
	var raw messaging.RawMessage
	if err := json.Unmarshal(msg.Payload, &raw); err != nil {
		logger.Errorf("[CreditConsumer] 解析信封失败: %v", err)
		return nil
	}

	// 2. 解析内层 CreditEventData
	var event messaging.CreditEventData
	if err := json.Unmarshal([]byte(raw.Data), &event); err != nil {
		logger.Errorf("[CreditConsumer] 解析事件数据失败: %v", err)
		return nil
	}

	// 3. 参数校验
	if event.UserID <= 0 || event.ActivityID <= 0 {
		logger.Infof("[CreditConsumer] 无效参数: userId=%d, activityId=%d", event.UserID, event.ActivityID)
		return nil
	}

	// 4. 映射事件类型
	changeType, ok := eventTypeToChangeType[event.Type]
	if !ok {
		logger.Infof("[CreditConsumer] 未知事件类型: %s", event.Type)
		return nil
	}

	reason := eventTypeToReason[event.Type]
	if reason == "" {
		reason = fmt.Sprintf("活动事件: %s", event.Type)
	}
	sourceID := fmt.Sprintf("%s:%d:%d", event.Type, event.ActivityID, event.UserID)

	// 5. 调 UserRpc.UpdateScore
	_, err := c.creditRpc.UpdateScore(ctx, &pb.UpdateScoreReq{
		UserId:     event.UserID,
		ChangeType: changeType,
		SourceId:   sourceID,
		Reason:     reason,
	})
	if err != nil {
		// 幂等错误 = 成功
		if st, ok := status.FromError(err); ok && st.Code() == codes.AlreadyExists {
			logger.Infof("[CreditConsumer] 幂等拦截: sourceId=%s", sourceID)
			return nil
		}
		logger.Errorf("[CreditConsumer] RPC调用失败: userId=%d, type=%s, err=%v",
			event.UserID, event.Type, err)
		return err // 触发重试
	}

	logger.Infof("[CreditConsumer] 处理成功: userId=%d, type=%s, sourceId=%s",
		event.UserID, event.Type, sourceID)
	return nil
}
