package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Message 消息模型
// 对应数据库表：messages（已应用按月范围分区）
type Message struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement:false;column:id" json:"id"` // 复合主键的一部分，禁用自动增长由业务或DB控制
	MessageID string `gorm:"uniqueIndex:uk_message_id;column:message_id;type:varchar(64);not null" json:"message_id"`
	GroupID   string `gorm:"index:idx_group_id_created;column:group_id;type:varchar(64);not null" json:"group_id"`
	SenderID  string `gorm:"index:idx_sender_id;column:sender_id;type:varchar(64);not null" json:"sender_id"`
	MsgType   int8   `gorm:"column:msg_type;type:tinyint;not null" json:"msg_type"`       // 1-文字 2-图片
	Content   string `gorm:"column:content;type:text" json:"content"`                     // 文本内容
	ImageURL  string `gorm:"column:image_url;type:varchar(512)" json:"image_url"`         // 图片URL
	Status    int8   `gorm:"column:status;type:tinyint;not null;default:1" json:"status"` // 1-正常 2-已撤回

	// CreatedAt 是复合主键和分区键
	CreatedAt time.Time `gorm:"primaryKey;column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

// TableName 指定表名
func (Message) TableName() string {
	return "messages"
}

// MessageModel 消息模型接口
type MessageModel interface {
	Insert(ctx context.Context, data *Message) error
	FindOne(ctx context.Context, messageID string) (*Message, error)
	FindByGroupID(ctx context.Context, groupID, beforeID string, limit int32) ([]*Message, error)
	FindOfflineMessages(ctx context.Context, userID string, afterTime int64) ([]*Message, error)
	UpdateStatus(ctx context.Context, messageID string, status int8) error
}

// defaultMessageModel 消息模型默认实现
type defaultMessageModel struct {
	db *gorm.DB
}

// NewMessageModel 创建消息模型实例
func NewMessageModel(db *gorm.DB) MessageModel {
	return &defaultMessageModel{db: db}
}

// Insert 插入消息记录
func (m *defaultMessageModel) Insert(ctx context.Context, data *Message) error {
	return m.db.WithContext(ctx).Create(data).Error
}

// FindOne 根据消息ID查询消息
func (m *defaultMessageModel) FindOne(ctx context.Context, messageID string) (*Message, error) {
	var message Message
	err := m.db.WithContext(ctx).
		Where("message_id = ?", messageID).
		First(&message).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

// FindByGroupID 根据群聊ID查询历史消息（分页）
func (m *defaultMessageModel) FindByGroupID(ctx context.Context, groupID, beforeID string, limit int32) ([]*Message, error) {
	var messages []*Message

	query := m.db.WithContext(ctx).
		Where("group_id = ? AND status = 1", groupID).
		Order("created_at DESC")

	// 如果指定了 beforeID，则查询该消息之前的消息
	if beforeID != "" {
		var beforeMsg Message
		if err := m.db.WithContext(ctx).Where("message_id = ?", beforeID).First(&beforeMsg).Error; err == nil {
			query = query.Where("created_at < ?", beforeMsg.CreatedAt)
		}
	}

	if err := query.Limit(int(limit)).Find(&messages).Error; err != nil {
		return nil, err
	}

	return messages, nil
}

// FindOfflineMessages 查询离线消息
func (m *defaultMessageModel) FindOfflineMessages(ctx context.Context, userID string, afterTime int64) ([]*Message, error) {
	var messages []*Message

	// 查询用户加入的所有群的消息
	// 这里需要关联 group_members 表
	err := m.db.WithContext(ctx).
		Table("messages").
		Joins("INNER JOIN group_members ON messages.group_id = group_members.group_id").
		Where("group_members.user_id = ? AND group_members.status = 1", userID).
		Where("messages.created_at > FROM_UNIXTIME(?)", afterTime).
		Where("messages.status = 1").
		Order("messages.created_at ASC").
		Find(&messages).Error

	if err != nil {
		return nil, err
	}

	return messages, nil
}

// UpdateStatus 更新消息状态（如撤回消息）
func (m *defaultMessageModel) UpdateStatus(ctx context.Context, messageID string, status int8) error {
	return m.db.WithContext(ctx).
		Model(&Message{}).
		Where("message_id = ?", messageID).
		Update("status", status).Error
}
