package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Group 群聊模型
// 对应数据库表：groups
type Group struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`                                               // 自增主键
	GroupID     string    `gorm:"uniqueIndex:uk_group_id;column:group_id;type:varchar(64);not null" json:"group_id"`          // 群聊唯一标识
	ActivityID  string    `gorm:"uniqueIndex:uk_activity_id;column:activity_id;type:varchar(64);not null" json:"activity_id"` // 关联活动ID
	Name        string    `gorm:"column:name;type:varchar(255);not null" json:"name"`                                         // 群聊名称
	OwnerID     string    `gorm:"index:idx_owner_id;column:owner_id;type:varchar(64);not null" json:"owner_id"`               // 群主用户ID
	Status      int8      `gorm:"index:idx_status;column:status;type:tinyint;not null;default:1" json:"status"`               // 状态: 1-正常 2-已解散
	MaxMembers  int32     `gorm:"column:max_members;type:int;not null" json:"max_members"`                                    // 最大成员数
	MemberCount int32     `gorm:"column:member_count;type:int;not null;default:0" json:"member_count"`                        // 成员数量
	CreatedAt   time.Time `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt   time.Time `gorm:"column:deleted_at;type:datetime;not null;default:NULL" json:"deleted_at"`
}

// TableName 指定 GORM 使用的表名
func (Group) TableName() string {
	return "groups"
}

// GroupModel 群聊模型接口
type GroupModel interface {
	Insert(ctx context.Context, data *Group) error
	FindOne(ctx context.Context, groupID string) (*Group, error)
	FindByActivityID(ctx context.Context, activityID string) (*Group, error)
	Update(ctx context.Context, data *Group) error
	Delete(ctx context.Context, groupID string) error
	UpdateStatus(ctx context.Context, groupID string, status int8) error
	IncrementMemberCount(ctx context.Context, groupID string, delta int32) error
}

// defaultGroupModel 群聊模型默认实现
type defaultGroupModel struct {
	db *gorm.DB
}

// NewGroupModel 创建群聊模型实例
func NewGroupModel(db *gorm.DB) GroupModel {
	return &defaultGroupModel{db: db}
}

// Insert 插入群聊记录
func (m *defaultGroupModel) Insert(ctx context.Context, data *Group) error {
	return m.db.WithContext(ctx).Create(data).Error
}

// FindOne 根据群聊ID查询群聊信息
func (m *defaultGroupModel) FindOne(ctx context.Context, groupID string) (*Group, error) {
	var group Group
	err := m.db.WithContext(ctx).
		Where("group_id = ? AND status = 1", groupID).
		First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// FindByActivityID 根据活动ID查询群聊信息
func (m *defaultGroupModel) FindByActivityID(ctx context.Context, activityID string) (*Group, error) {
	var group Group
	err := m.db.WithContext(ctx).
		Where("activity_id = ? AND status = 1", activityID).
		First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// Update 更新群聊信息
func (m *defaultGroupModel) Update(ctx context.Context, data *Group) error {
	return m.db.WithContext(ctx).
		Where("group_id = ?", data.GroupID).
		Updates(data).Error
}

// Delete 删除群聊（软删除）
func (m *defaultGroupModel) Delete(ctx context.Context, groupID string) error {
	return m.db.WithContext(ctx).
		Model(&Group{}).
		Where("group_id = ?", groupID).
		Update("deleted_at", time.Now()).Error
}

// UpdateStatus 更新群聊状态
func (m *defaultGroupModel) UpdateStatus(ctx context.Context, groupID string, status int8) error {
	return m.db.WithContext(ctx).
		Model(&Group{}).
		Where("group_id = ?", groupID).
		Update("status", status).Error
}

// IncrementMemberCount 增加或减少成员数量
func (m *defaultGroupModel) IncrementMemberCount(ctx context.Context, groupID string, delta int32) error {
	return m.db.WithContext(ctx).
		Model(&Group{}).
		Where("group_id = ?", groupID).
		UpdateColumn("member_count", gorm.Expr("member_count + ?", delta)).Error
}
