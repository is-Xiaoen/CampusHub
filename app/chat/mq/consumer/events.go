package consumer

import "time"

// ActivityCreatedEvent 活动创建事件
type ActivityCreatedEvent struct {
	ActivityID uint64    `json:"activity_id"`
	CreatorID  uint64    `json:"creator_id"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"created_at"`
}

// ActivityMemberJoinedEvent 用户报名成功事件
type ActivityMemberJoinedEvent struct {
	ActivityID uint64    `json:"activity_id"`
	UserID     uint64    `json:"user_id"`
	JoinedAt   time.Time `json:"joined_at"`
}

// ActivityMemberLeftEvent 用户取消报名事件
type ActivityMemberLeftEvent struct {
	ActivityID uint64    `json:"activity_id"`
	UserID     uint64    `json:"user_id"`
	LeftAt     time.Time `json:"left_at"`
}
