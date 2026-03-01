package messaging

const (
	TopicNotificationPush = "notification:push"
)

type NotificationPushEventData struct {
	UserID         uint64 `json:"user_id"`
	NotificationID string `json:"notification_id"`
	Type           string `json:"type"`
	Title          string `json:"title"`
	Content        string `json:"content"`
	Timestamp      int64  `json:"timestamp"`
}
