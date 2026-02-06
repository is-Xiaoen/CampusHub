/**
 * @projectName: CampusHub
 * @package: consumer
 * @className: VerifyOcrConsumer
 * @description: OCR 认证消费者（薄层：解析 RawMessage → 调 UserRpc.ProcessOcrVerify）
 * @date: 2026-02-06
 * @version: 1.0
 *
 * 消息来源: User RPC（用户提交认证申请后异步触发）
 * Topic: verify:events（RawMessage 信封格式）
 */

package consumer

import (
	"context"
	"encoding/json"

	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/messaging"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/zeromicro/go-zero/core/logx"
)

// VerifyOcrConsumer OCR 认证消费者
type VerifyOcrConsumer struct {
	verifyRpc pb.VerifyServiceClient
	logger    logx.Logger
}

// NewVerifyOcrConsumer 创建 OCR 认证消费者
func NewVerifyOcrConsumer(verifyRpc pb.VerifyServiceClient) *VerifyOcrConsumer {
	return &VerifyOcrConsumer{
		verifyRpc: verifyRpc,
		logger:    logx.WithContext(context.Background()),
	}
}

// Subscribe 订阅认证事件主题
func (c *VerifyOcrConsumer) Subscribe(msgClient *messaging.Client) {
	msgClient.Subscribe("verify:events", "verify-event-handler", c.handleVerifyEvent)
	c.logger.Info("已订阅 verify:events 事件")
}

// handleVerifyEvent 处理认证事件（RawMessage 信封格式）
func (c *VerifyOcrConsumer) handleVerifyEvent(msg *message.Message) error {
	ctx := msg.Context()
	logger := logx.WithContext(ctx)

	// 1. 解析 RawMessage 信封
	var raw messaging.RawMessage
	if err := json.Unmarshal(msg.Payload, &raw); err != nil {
		logger.Errorf("[VerifyConsumer] 解析信封失败: %v", err)
		return nil
	}

	// 2. 解析内层 VerifyApplyEventData
	var event messaging.VerifyApplyEventData
	if err := json.Unmarshal([]byte(raw.Data), &event); err != nil {
		logger.Errorf("[VerifyConsumer] 解析事件数据失败: %v", err)
		return nil
	}

	// 3. 参数校验
	if event.VerifyID <= 0 || event.UserID <= 0 {
		logger.Infof("[VerifyConsumer] 无效参数: verifyId=%d, userId=%d", event.VerifyID, event.UserID)
		return nil
	}

	logger.Infof("[VerifyConsumer] 开始处理: verifyId=%d, userId=%d", event.VerifyID, event.UserID)

	// 4. 调 UserRpc.ProcessOcrVerify
	resp, err := c.verifyRpc.ProcessOcrVerify(ctx, &pb.ProcessOcrVerifyReq{
		VerifyId:      event.VerifyID,
		UserId:        event.UserID,
		FrontImageUrl: event.FrontImageURL,
		BackImageUrl:  event.BackImageURL,
	})
	if err != nil {
		logger.Errorf("[VerifyConsumer] RPC调用失败: verifyId=%d, err=%v", event.VerifyID, err)
		return err // 触发重试
	}

	logger.Infof("[VerifyConsumer] 处理完成: verifyId=%d, status=%d, msg=%s",
		event.VerifyID, resp.ResultStatus, resp.Message)
	return nil
}
