/**
 * @projectName: CampusHub
 * @package: handler
 * @className: Handler
 * @author: lijunqi
 * @description: MQ消息处理器定义（基于 Watermill Redis Stream）
 * @date: 2026-01-30
 * @version: 1.0
 */

package handler

import (
	"context"
	"encoding/json"

	"activity-platform/app/user/mq/internal/svc"
	"activity-platform/common/messaging"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/zeromicro/go-zero/core/logx"
)

// ==================== 消息类型常量 ====================
// 消息类型常量和 RawMessage 定义在 common/messaging/message.go
// 使用 messaging.MsgTypeCreditChange、messaging.RawMessage 等

// Message 业务消息结构（从 Watermill 消息解析而来）
// 这是消费者内部使用的封装结构，包含 Watermill 消息 ID
type Message struct {
	// ID 消息ID（Watermill UUID）
	ID string

	// Type 消息类型
	Type string

	// Data 消息数据（JSON 字符串）
	Data string
}

// Handler 业务消息处理函数签名
type Handler func(ctx context.Context, msg *Message) error

// Handlers 消息处理器管理器
type Handlers struct {
	svcCtx   *svc.ServiceContext
	handlers map[string]Handler
}

// NewHandlers 创建处理器集合
func NewHandlers(svcCtx *svc.ServiceContext) *Handlers {
	h := &Handlers{
		svcCtx:   svcCtx,
		handlers: make(map[string]Handler),
	}

	// 注册所有处理器
	h.handlers[messaging.MsgTypeCreditChange] = NewCreditChangeHandler(svcCtx)
	h.handlers[messaging.MsgTypeVerifyCallback] = NewVerifyCallbackHandler(svcCtx)

	return h
}

// WatermillHandler 返回 Watermill 兼容的处理函数
// 用于注册到 messaging.Client.Subscribe()
func (h *Handlers) WatermillHandler() message.NoPublishHandlerFunc {
	return func(wmMsg *message.Message) error {
		ctx := wmMsg.Context()
		logger := logx.WithContext(ctx)

		// 1. 解析原始消息
		var raw messaging.RawMessage
		if err := json.Unmarshal(wmMsg.Payload, &raw); err != nil {
			logger.Errorf("[Handler] 解析消息失败: %v, payload=%s", err, string(wmMsg.Payload))
			// 解析失败不重试，直接 ACK
			return nil
		}

		// 2. 构造业务消息
		msg := &Message{
			ID:   wmMsg.UUID,
			Type: raw.Type,
			Data: raw.Data,
		}

		// 3. 根据类型分发处理
		handler, ok := h.handlers[msg.Type]
		if !ok {
			logger.Infof("[Handler] [WARN] 未知消息类型，跳过: type=%s, id=%s", msg.Type, msg.ID)
			return nil
		}

		// 4. 调用业务处理器
		if err := handler(ctx, msg); err != nil {
			logger.Errorf("[Handler] 处理消息失败: type=%s, id=%s, err=%v", msg.Type, msg.ID, err)
			return err // 返回错误触发重试
		}

		logger.Infof("[Handler] 消息处理成功: type=%s, id=%s", msg.Type, msg.ID)
		return nil
	}
}

// Handle 处理消息（内部使用，保持向后兼容）
func (h *Handlers) Handle(ctx context.Context, msg *Message) error {
	handler, ok := h.handlers[msg.Type]
	if !ok {
		return nil
	}
	return handler(ctx, msg)
}
