/**
 * @projectName: CampusHub
 * @package: messaging
 * @className: credit_event
 * @author: lijunqi
 * @description: 信用事件消息协议定义
 * @date: 2026-02-06
 * @version: 1.0
 *
 * 本文件定义了 Activity 服务与 User/MQ 服务之间的信用事件消息协议。
 *
 * 使用方:
 *   - Activity 服务（生产者）：发布信用事件消息
 *   - User/MQ 服务（消费者）：消费并处理信用事件
 *
 * 消息流向:
 *   Activity RPC -> Redis Stream (credit:events) -> User MQ Handler
 */

package messaging

// ==================== Topic 定义 ====================

const (
	// TopicCreditEvent 信用事件消息队列 Topic
	// 用于 Activity 服务发布信用变更事件，User/MQ 服务消费处理
	TopicCreditEvent = "credit:events"
)

// ==================== 事件类型常量 ====================
// 定义所有支持的信用事件类型，供生产者和消费者共同使用

const (
	// CreditEventCheckin 签到成功
	// 场景: 用户在活动开始时完成签到
	// 分值变动: +2
	CreditEventCheckin = "checkin"

	// CreditEventCancelEarly 提前取消
	// 场景: 用户在活动开始前 24 小时之前取消报名
	// 分值变动: 0（无责取消，但记录日志）
	CreditEventCancelEarly = "cancel_early"

	// CreditEventCancelLate 临期取消
	// 场景: 用户在活动开始前 24 小时内取消报名
	// 分值变动: -5
	CreditEventCancelLate = "cancel_late"

	// CreditEventNoShow 爽约
	// 场景: 用户报名后未签到，活动已结束
	// 分值变动: -10
	CreditEventNoShow = "noshow"

	// CreditEventHostSuccess 成功举办
	// 场景: 组织者的活动圆满结束（有人参与且正常签到）
	// 分值变动: +5
	CreditEventHostSuccess = "host_success"

	// CreditEventHostDelete 删除活动
	// 场景: 组织者删除已有报名的活动
	// 分值变动: -10
	CreditEventHostDelete = "host_delete"
)

// ==================== 消息数据结构 ====================

// CreditEventData 信用事件消息数据
// 由 Activity 服务发布到 Redis Stream，User/MQ 服务消费处理
//
// 消息示例:
//
//	{
//	  "type": "checkin",
//	  "activity_id": 123456,
//	  "user_id": 789,
//	  "timestamp": 1706745600,
//	  "trace_id": "abc123"
//	}
type CreditEventData struct {
	// Type 事件类型
	// 可选值: checkin | cancel_early | cancel_late | noshow | host_success | host_delete
	// 建议使用 CreditEvent* 常量，避免硬编码字符串
	Type string `json:"type"`

	// ActivityID 活动ID（用于构造幂等键）
	ActivityID int64 `json:"activity_id"`

	// UserID 用户ID
	// 对于参与者事件(checkin/cancel/noshow): 为报名用户ID
	// 对于组织者事件(host_success/host_delete): 为活动创建者ID
	UserID int64 `json:"user_id"`

	// Timestamp 事件发生时间（Unix 秒级时间戳）
	Timestamp int64 `json:"timestamp"`

	// TraceID 链路追踪ID（可选，用于日志关联）
	TraceID string `json:"trace_id,omitempty"`
}

// ==================== 辅助函数 ====================

// ValidCreditEventTypes 所有有效的信用事件类型
var ValidCreditEventTypes = map[string]bool{
	CreditEventCheckin:     true,
	CreditEventCancelEarly: true,
	CreditEventCancelLate:  true,
	CreditEventNoShow:      true,
	CreditEventHostSuccess: true,
	CreditEventHostDelete:  true,
}

// IsValidCreditEventType 检查事件类型是否有效
func IsValidCreditEventType(eventType string) bool {
	return ValidCreditEventTypes[eventType]
}
