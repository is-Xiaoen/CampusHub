/**
 * @projectName: CampusHub
 * @package: messaging
 * @className: verify_event
 * @author: lijunqi
 * @description: 学生认证事件消息协议定义
 * @date: 2026-02-06
 * @version: 1.0
 *
 * 本文件定义了学生认证流程中的事件消息协议。
 *
 * 使用方:
 *   - User RPC 服务（生产者）：用户申请认证时发布事件
 *   - User MQ 服务（消费者）：消费事件并执行 OCR 识别
 *
 * 消息流向:
 *   User RPC -> Redis Stream (verify:events) -> User MQ Handler -> OCR -> Update DB
 */

package messaging

// ==================== Topic 定义 ====================

const (
	// TopicVerifyEvent 认证事件消息队列 Topic
	// 用于 User RPC 发布认证申请事件，User MQ 消费并处理 OCR
	TopicVerifyEvent = "verify:events"
)

// ==================== 事件类型常量 ====================

const (
	// VerifyEventApplyOcr 认证申请 - 触发 OCR 识别
	// 场景: 用户提交学生认证申请后，异步触发 OCR 识别
	VerifyEventApplyOcr = "apply_ocr"
)

// ==================== 消息数据结构 ====================

// VerifyApplyEventData 认证申请事件数据
// 由 User RPC 发布到 Redis Stream，User MQ 消费并执行 OCR 识别
//
// 消息示例:
//
//	{
//	  "verify_id": 123,
//	  "user_id": 456,
//	  "front_image_url": "https://...",
//	  "back_image_url": "https://...",
//	  "timestamp": 1706745600,
//	  "trace_id": "abc123"
//	}
type VerifyApplyEventData struct {
	// VerifyID 认证记录ID
	VerifyID int64 `json:"verify_id"`

	// UserID 用户ID
	UserID int64 `json:"user_id"`

	// FrontImageURL 学生证正面图片URL
	FrontImageURL string `json:"front_image_url"`

	// BackImageURL 学生证详情面图片URL
	BackImageURL string `json:"back_image_url"`

	// Timestamp 事件发生时间（Unix 秒级时间戳）
	Timestamp int64 `json:"timestamp"`

	// TraceID 链路追踪ID（可选）
	TraceID string `json:"trace_id,omitempty"`
}

// ==================== 辅助函数 ====================

// ValidVerifyEventTypes 所有有效的认证事件类型
var ValidVerifyEventTypes = map[string]bool{
	VerifyEventApplyOcr: true,
}

// IsValidVerifyEventType 检查事件类型是否有效
func IsValidVerifyEventType(eventType string) bool {
	return ValidVerifyEventTypes[eventType]
}
