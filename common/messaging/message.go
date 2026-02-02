package messaging

import (
	"fmt"
	"regexp"
	"time"
)

// Message 表示要传输的消息
type Message struct {
	// ID 消息唯一标识符
	ID string `json:"id"`

	// Topic 消息所属主题
	Topic string `json:"topic"`

	// Payload 消息负载（业务数据）
	Payload []byte `json:"payload"`

	// Metadata 消息元数据
	Metadata Metadata `json:"metadata"`

	// CreatedAt 消息创建时间
	CreatedAt time.Time `json:"created_at"`

	// ReceivedAt 消息接收时间（消费者侧）
	ReceivedAt time.Time `json:"received_at"`

	// Ack 确认函数（消费者调用以确认消息已处理）
	Ack func() error `json:"-"`

	// Nack 拒绝函数（消费者调用以拒绝消息）
	Nack func() error `json:"-"`
}

// Metadata 消息元数据（键值对）
type Metadata map[string]string

// 常用元数据键
const (
	MetadataKeyTraceID       = "trace_id"        // 链路追踪ID
	MetadataKeyCorrelationID = "correlation_id"  // 关联ID
	MetadataKeyContentType   = "content_type"    // 内容类型
	MetadataKeyRetryCount    = "retry_count"     // 重试次数
	MetadataKeyFirstFailedAt = "first_failed_at" // 首次失败时间
	MetadataKeyLastError     = "last_error"      // 最后错误
	MetadataKeySourceService = "source_service"  // 来源服务
	MetadataKeyEventType     = "event_type"      // 事件类型
)

// Get 获取元数据值
func (m Metadata) Get(key string) string {
	return m[key]
}

// Set 设置元数据值
func (m Metadata) Set(key, value string) {
	m[key] = value
}

// Has 检查元数据键是否存在
func (m Metadata) Has(key string) bool {
	_, ok := m[key]
	return ok
}

// Delete 删除元数据键
func (m Metadata) Delete(key string) {
	delete(m, key)
}

// Clone 克隆元数据
func (m Metadata) Clone() Metadata {
	clone := make(Metadata, len(m))
	for k, v := range m {
		clone[k] = v
	}
	return clone
}

// 消息验证常量
const (
	MaxMessageSize  = 1 * 1024 * 1024 // 1MB
	MaxMetadataSize = 10 * 1024       // 10KB
	MinTopicLength  = 3
	MaxTopicLength  = 128
)

// 主题名称正则表达式：小写字母、数字、点号、下划线、连字符
var topicNameRegex = regexp.MustCompile(`^[a-z0-9-_.]+$`)

// Validate 验证消息
func (m *Message) Validate() error {
	// 验证 ID
	if m.ID == "" {
		return fmt.Errorf("消息ID不能为空")
	}

	// 验证主题
	if err := ValidateTopicName(m.Topic); err != nil {
		return fmt.Errorf("无效主题: %w", err)
	}

	// 验证负载大小
	if len(m.Payload) == 0 {
		return fmt.Errorf("消息负载不能为空")
	}
	if len(m.Payload) > MaxMessageSize {
		return fmt.Errorf("消息负载大小 (%d 字节) 超过最大限制 (%d 字节)", len(m.Payload), MaxMessageSize)
	}

	// 验证元数据大小
	metadataSize := 0
	for k, v := range m.Metadata {
		metadataSize += len(k) + len(v)
	}
	if metadataSize > MaxMetadataSize {
		return fmt.Errorf("元数据大小 (%d 字节) 超过最大限制 (%d 字节)", metadataSize, MaxMetadataSize)
	}

	return nil
}

// ValidateTopicName 验证主题名称
func ValidateTopicName(topic string) error {
	if len(topic) < MinTopicLength {
		return fmt.Errorf("主题名称过短 (最少 %d 个字符)", MinTopicLength)
	}
	if len(topic) > MaxTopicLength {
		return fmt.Errorf("主题名称过长 (最多 %d 个字符)", MaxTopicLength)
	}
	if !topicNameRegex.MatchString(topic) {
		return fmt.Errorf("主题名称必须匹配模式: ^[a-z0-9-_.]+$")
	}
	return nil
}

// NewMessage 创建新消息
func NewMessage(topic string, payload []byte) *Message {
	return &Message{
		Topic:     topic,
		Payload:   payload,
		Metadata:  make(Metadata),
		CreatedAt: time.Now(),
	}
}
