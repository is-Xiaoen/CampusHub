package consumer

import "time"

// ActivityCreatedEvent 活动创建事件
// 由 Activity 服务发布，Chat 服务订阅
type ActivityCreatedEvent struct {
	ActivityID string    `json:"activity_id"` // 活动ID
	CreatorID  string    `json:"creator_id"`  // 创建者ID
	Title      string    `json:"title"`       // 活动标题
	CreatedAt  time.Time `json:"created_at"`  // 创建时间
}

// ActivityMemberJoinedEvent 用户报名成功事件
// 由 Activity 服务发布，Chat 服务订阅
type ActivityMemberJoinedEvent struct {
	ActivityID string    `json:"activity_id"` // 活动ID
	UserID     string    `json:"user_id"`     // 用户ID
	JoinedAt   time.Time `json:"joined_at"`   // 报名时间
}

// ActivityMemberLeftEvent 用户取消报名事件
// 由 Activity 服务发布，Chat 服务订阅
type ActivityMemberLeftEvent struct {
	ActivityID string    `json:"activity_id"` // 活动ID
	UserID     string    `json:"user_id"`     // 用户ID
	LeftAt     time.Time `json:"left_at"`     // 离开时间
}
