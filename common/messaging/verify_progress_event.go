package messaging

const (
	// TopicVerifyProgress publishes verify status changes for websocket push.
	TopicVerifyProgress = "verify:progress"
)

// VerifyProgressEventData is the payload sent to TopicVerifyProgress.
type VerifyProgressEventData struct {
	UserID    int64  `json:"user_id"`
	VerifyID  int64  `json:"verify_id"`
	Status    int32  `json:"status"`
	Operator  string `json:"operator"`
	Refresh   bool   `json:"refresh"`
	Timestamp int64  `json:"timestamp"`
}
