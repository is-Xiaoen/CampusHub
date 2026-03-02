package types

import "encoding/json"

// MessageType 消息类型
type MessageType string

const (
	// 客户端 -> 服务端
	TypePing        MessageType = "ping"         // 心跳
	TypeAuth        MessageType = "auth"         // 认证
	TypeSendMessage MessageType = "send_message" // 发送消息

	// 服务端 -> 客户端
	TypePong           MessageType = "pong"            // 心跳响应
	TypeAuthSuccess    MessageType = "auth_success"    // 认证成功
	TypeAuthFailed     MessageType = "auth_failed"     // 认证失败
	TypeNewMessage     MessageType = "new_message"     // 新消息
	TypeNotification   MessageType = "notification"    // 系统通知
	TypeVerifyProgress MessageType = "verify_progress" // 认证进度更新
	TypeError          MessageType = "error"           // 错误消息
	TypeAck            MessageType = "ack"             // 消息确认
)

// WSMessage WebSocket 消息结构
type WSMessage struct {
	Type      MessageType     `json:"type"`           // 消息类型
	MessageID string          `json:"message_id"`     // 消息ID (用于去重和确认)
	Timestamp int64           `json:"timestamp"`      // 时间戳
	Data      json.RawMessage `json:"data,omitempty"` // 消息数据
}

// AuthData 认证数据
type AuthData struct {
	Token string `json:"token"` // JWT Token
}

// SendMessageData 发送消息数据
type SendMessageData struct {
	GroupID  string `json:"group_id"`            // 群聊ID
	MsgType  int32  `json:"msg_type"`            // 消息类型: 1-文字 2-图片
	Content  string `json:"content,omitempty"`   // 文本内容
	ImageURL string `json:"image_url,omitempty"` // 图片URL
}

// NewMessageData 新消息数据
type NewMessageData struct {
	MessageID    string `json:"message_id"`    // 消息ID
	GroupID      string `json:"group_id"`      // 群聊ID
	SenderID     uint64 `json:"sender_id"`     // 发送者ID
	SenderName   string `json:"sender_name"`   // 发送者名称
	SenderAvatar string `json:"sender_avatar"` // 发送者头像URL
	MsgType      int32  `json:"msg_type"`      // 消息类型
	Content      string `json:"content"`       // 内容
	ImageURL     string `json:"image_url"`     // 图片URL
	CreatedAt    int64  `json:"created_at"`    // 创建时间
}

// VerifyProgressData 认证进度通知数据
type VerifyProgressData struct {
	VerifyID int64 `json:"verify_id"` // 认证记录ID
	Status   int32 `json:"status"`    // 最新状态
	Refresh  bool  `json:"refresh"`   // 是否触发前端刷新
}

// ErrorData 错误数据
type ErrorData struct {
	Code    int    `json:"code"`    // 错误码
	Message string `json:"message"` // 错误信息
}

// NotificationData 系统通知数据
type NotificationData struct {
	NotificationID string `json:"notification_id"`
	Type           string `json:"type"`
	Title          string `json:"title"`
	Content        string `json:"content"`
}

// AckData 确认数据
type AckData struct {
	MessageID string `json:"message_id"` // 确认的消息ID
	Success   bool   `json:"success"`    // 是否成功
}
