package messaging

import "time"

// ==================== Topic 定义 ====================

const (
	TopicActivityCreated      = "activity.created"
	TopicActivityMemberJoined = "activity.member.joined"
	TopicActivityMemberLeft   = "activity.member.left"
	TopicActivityCancelled    = "activity.cancelled"
)

// ==================== 事件结构体 ====================
// 字段类型必须与 Chat MQ 消费者完全匹配（uint64 ID + time.Time）

// ActivityCreatedEvent 活动创建事件
// 消费者：Chat MQ（自动创建活动群聊）
type ActivityCreatedEvent struct {
	ActivityID uint64    `json:"activity_id"`
	CreatorID  uint64    `json:"creator_id"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"created_at"`
}

// ActivityMemberJoinedEvent 用户报名事件
// 消费者：Chat MQ（自动加入活动群聊）
type ActivityMemberJoinedEvent struct {
	ActivityID uint64    `json:"activity_id"`
	UserID     uint64    `json:"user_id"`
	JoinedAt   time.Time `json:"joined_at"`
}

// ActivityMemberLeftEvent 用户取消报名事件
// 消费者：Chat MQ（自动退出活动群聊）
type ActivityMemberLeftEvent struct {
	ActivityID uint64    `json:"activity_id"`
	UserID     uint64    `json:"user_id"`
	LeftAt     time.Time `json:"left_at"`
}

// ActivityCancelledEvent 活动取消事件
// 消费者：Chat MQ（通知所有已报名参与者）
type ActivityCancelledEvent struct {
	ActivityID  uint64    `json:"activity_id"`
	CancelledBy uint64    `json:"cancelled_by"`
	Reason      string    `json:"reason"`
	CancelledAt time.Time `json:"cancelled_at"`
}
