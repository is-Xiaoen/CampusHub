/**
 * @projectName: CampusHub
 * @package: messaging
 * @className: message
 * @author: lijunqi
 * @description: 通用消息格式定义
 * @date: 2026-02-06
 * @version: 1.0
 *
 * 本文件定义了服务间消息通信的通用格式和类型常量。
 * 所有通过消息队列通信的服务都应使用这些定义。
 */

package messaging

// ==================== 消息类型常量 ====================
// 用于消息路由，区分不同业务的处理器

const (
	// MsgTypeCreditChange 信用分变更
	// 消息来源: Activity 服务（签到、爽约等事件）
	// 消费者: User/MQ 服务
	// 内层数据: CreditEventData
	MsgTypeCreditChange = "credit_change"

	// MsgTypeVerifyCallback 认证回调
	// 消息来源: OCR 服务回调、人工审核结果
	// 消费者: User/MQ 服务
	MsgTypeVerifyCallback = "verify_callback"
)

// ==================== 通用消息结构 ====================

// RawMessage 通用消息格式
// 所有服务发送消息时应使用此格式，便于消费者路由分发
//
// 消息示例:
//
//	{
//	  "type": "credit_change",
//	  "data": "{\"type\":\"checkin\",\"activity_id\":123,\"user_id\":456}"
//	}
type RawMessage struct {
	// Type 消息类型，用于路由到不同处理器
	// 可选值: credit_change | verify_callback
	Type string `json:"type"`

	// Data 消息数据（JSON 字符串）
	// 具体格式由 Type 决定:
	//   - credit_change: CreditEventData
	//   - verify_callback: VerifyCallbackData（待定义）
	Data string `json:"data"`
}

// ==================== 辅助函数 ====================

// NewRawMessage 创建通用消息
func NewRawMessage(msgType string, data string) *RawMessage {
	return &RawMessage{
		Type: msgType,
		Data: data,
	}
}
