/**
 * @projectName: CampusHub
 * @package: handler
 * @className: Handler
 * @author: lijunqi
 * @description: MQ消息处理器定义（基于Redis Stream）
 * @date: 2026-01-30
 * @version: 1.0
 */

package handler

import (
	"context"

	"activity-platform/app/user/mq/internal/svc"
)

// ==================== 消息类型常量 ====================
// [待确认] 消息类型的定义方式需要和队友统一
// 可能是：1. 不同的 Stream Key  2. 同一 Stream 内的 type 字段

const (
	// MsgTypeCreditChange 信用分变更
	// 消息来源: Activity服务（签到、爽约等事件）
	MsgTypeCreditChange = "credit_change"

	// MsgTypeVerifyCallback 认证回调
	// 消息来源: OCR服务回调、人工审核结果
	MsgTypeVerifyCallback = "verify_callback"
)

// Message Redis Stream 消息结构
// [待确认] 需要和队友确认消息结构
type Message struct {
	// ID Redis Stream 消息ID（如 "1234567890-0"）
	ID string

	// Type 消息类型
	Type string

	// Data 消息数据（JSON 字符串）
	Data string
}

// Handler 消息处理函数签名
type Handler func(ctx context.Context, msg *Message) error

// Handlers 消息处理器映射
// key: 消息类型, value: 处理函数
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
	h.handlers[MsgTypeCreditChange] = NewCreditChangeHandler(svcCtx)
	h.handlers[MsgTypeVerifyCallback] = NewVerifyCallbackHandler(svcCtx)

	return h
}

// GetHandler 根据消息类型获取处理器
func (h *Handlers) GetHandler(msgType string) (Handler, bool) {
	handler, ok := h.handlers[msgType]
	return handler, ok
}

// Handle 处理消息（根据类型分发）
func (h *Handlers) Handle(ctx context.Context, msg *Message) error {
	handler, ok := h.GetHandler(msg.Type)
	if !ok {
		// 未知消息类型，记录日志但不报错
		return nil
	}
	return handler(ctx, msg)
}
