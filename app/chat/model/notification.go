package model

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Notification 通知模型
// 对应数据库表：notifications
type Notification struct {
	ID             uint64 `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	NotificationID string `gorm:"uniqueIndex:uk_notification_id;column:notification_id;type:varchar(64);not null" json:"notification_id"`
	UserID         uint64 `gorm:"index:idx_user_id_created;column:user_id;type:bigint;not null" json:"user_id"`
	Type           string `gorm:"column:type;type:varchar(32);not null" json:"type"` // 通知类型，如 "system", "group_invite"
	Title          string `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Content        string `gorm:"column:content;type:text;not null" json:"content"`

	// Data 对应 MySQL 的 JSON 类型
	// 使用 datatypes.JSON (来自 "gorm.io/datatypes") 效果更好，或者简单的 []byte/string
	Data json.RawMessage `gorm:"column:data;type:json" json:"data"`

	IsRead    int8       `gorm:"index:idx_is_read;column:is_read;type:tinyint;not null;default:0" json:"is_read"` // 0-未读 1-已读
	ReadAt    *time.Time `gorm:"column:read_at;type:datetime" json:"read_at,omitempty"`                           // 使用指针处理 NULL
	CreatedAt time.Time  `gorm:"index:idx_user_id_created;column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (Notification) TableName() string {
	return "notifications"
}

// NotificationModel 通知模型接口
type NotificationModel interface {
	Insert(ctx context.Context, data *Notification) error
	FindOne(ctx context.Context, notificationID string) (*Notification, error)
	FindByUserID(ctx context.Context, userID uint64, isRead int32, page, pageSize int32) ([]*Notification, int64, error)
	GetUnreadCount(ctx context.Context, userID uint64) (int64, error)
	MarkAsRead(ctx context.Context, userID uint64, notificationIDs []string) (int64, error)
	MarkAllAsRead(ctx context.Context, userID uint64) (int64, error)
}

// defaultNotificationModel 通知模型默认实现
type defaultNotificationModel struct {
	db *gorm.DB
}

// NewNotificationModel 创建通知模型实例
func NewNotificationModel(db *gorm.DB) NotificationModel {
	return &defaultNotificationModel{db: db}
}

// Insert 插入通知记录
func (m *defaultNotificationModel) Insert(ctx context.Context, data *Notification) error {
	return m.db.WithContext(ctx).Create(data).Error
}

// FindOne 根据通知ID查询通知
func (m *defaultNotificationModel) FindOne(ctx context.Context, notificationID string) (*Notification, error) {
	var notification Notification
	err := m.db.WithContext(ctx).
		Where("notification_id = ?", notificationID).
		First(&notification).Error
	if err != nil {
		return nil, err
	}
	return &notification, nil
}

// FindByUserID 根据用户ID查询通知列表（分页）
func (m *defaultNotificationModel) FindByUserID(ctx context.Context, userID uint64, isRead int32, page, pageSize int32) ([]*Notification, int64, error) {
	var notifications []*Notification
	var total int64

	query := m.db.WithContext(ctx).Where("user_id = ?", userID)

	// 筛选已读/未读
	if isRead >= 0 {
		query = query.Where("is_read = ?", isRead)
	}

	// 查询总数
	if err := query.Model(&Notification{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").
		Offset(int(offset)).
		Limit(int(pageSize)).
		Find(&notifications).Error; err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

// GetUnreadCount 获取未读通知数量
func (m *defaultNotificationModel) GetUnreadCount(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&Notification{}).
		Where("user_id = ? AND is_read = 0", userID).
		Count(&count).Error
	return count, err
}

// MarkAsRead 标记指定通知为已读
func (m *defaultNotificationModel) MarkAsRead(ctx context.Context, userID uint64, notificationIDs []string) (int64, error) {
	now := time.Now()
	result := m.db.WithContext(ctx).
		Model(&Notification{}).
		Where("user_id = ? AND notification_id IN ?", userID, notificationIDs).
		Where("is_read = 0").
		Updates(map[string]interface{}{
			"is_read": 1,
			"read_at": &now,
		})

	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

// MarkAllAsRead 标记用户所有通知为已读
func (m *defaultNotificationModel) MarkAllAsRead(ctx context.Context, userID uint64) (int64, error) {
	now := time.Now()
	result := m.db.WithContext(ctx).
		Model(&Notification{}).
		Where("user_id = ? AND is_read = 0", userID).
		Updates(map[string]interface{}{
			"is_read": 1,
			"read_at": &now,
		})

	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}
