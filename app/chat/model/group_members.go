package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// GroupMember 群成员模型
// 对应数据库表：group_members
type GroupMember struct {
	ID       uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"` // group_id 和 user_id 构成联合唯一索引 uk_group_user
	GroupID  string    `gorm:"uniqueIndex:uk_group_user;column:group_id;type:varchar(64);not null" json:"group_id"`
	UserID   string    `gorm:"uniqueIndex:uk_group_user;index:idx_user_id;column:user_id;type:varchar(64);not null" json:"user_id"`
	Role     int8      `gorm:"column:role;type:tinyint;not null;default:1" json:"role"`                      // 1-普通成员 2-群主
	Status   int8      `gorm:"index:idx_status;column:status;type:tinyint;not null;default:1" json:"status"` // 1-正常 2-已退出
	JoinedAt time.Time `gorm:"column:joined_at;type:datetime;not null;default:CURRENT_TIMESTAMP" json:"joined_at"`

	// LeftAt 在 SQL 中允许为 NULL，在 Go 中建议使用 *time.Time 或 sql.NullTime
	LeftAt *time.Time `gorm:"column:left_at;type:datetime" json:"left_at,omitempty"`
}

// TableName 指定表名
func (GroupMember) TableName() string {
	return "group_members"
}

// GroupMemberModel 群成员模型接口
type GroupMemberModel interface {
	Insert(ctx context.Context, data *GroupMember) error
	FindOne(ctx context.Context, groupID, userID string) (*GroupMember, error)
	FindByGroupID(ctx context.Context, groupID string, page, pageSize int32) ([]*GroupMember, int64, error)
	FindByUserID(ctx context.Context, userID string, page, pageSize int32) ([]*GroupMember, int64, error)
	Delete(ctx context.Context, groupID, userID string) error
	UpdateRole(ctx context.Context, groupID, userID string, role int8) error
	UpdateStatus(ctx context.Context, groupID, userID string, status int8) error
}

// defaultGroupMemberModel 群成员模型默认实现
type defaultGroupMemberModel struct {
	db *gorm.DB
}

// NewGroupMemberModel 创建群成员模型实例
func NewGroupMemberModel(db *gorm.DB) GroupMemberModel {
	return &defaultGroupMemberModel{db: db}
}

// Insert 插入群成员记录
func (m *defaultGroupMemberModel) Insert(ctx context.Context, data *GroupMember) error {
	return m.db.WithContext(ctx).Create(data).Error
}

// FindOne 查询单个群成员
func (m *defaultGroupMemberModel) FindOne(ctx context.Context, groupID, userID string) (*GroupMember, error) {
	var member GroupMember
	err := m.db.WithContext(ctx).
		Where("group_id = ? AND user_id = ? AND status = 1", groupID, userID).
		First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// FindByGroupID 根据群聊ID查询成员列表（分页）
func (m *defaultGroupMemberModel) FindByGroupID(ctx context.Context, groupID string, page, pageSize int32) ([]*GroupMember, int64, error) {
	var members []*GroupMember
	var total int64

	db := m.db.WithContext(ctx).
		Where("group_id = ? AND status = 1", groupID)

	// 查询总数
	if err := db.Model(&GroupMember{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := db.Offset(int(offset)).Limit(int(pageSize)).Find(&members).Error; err != nil {
		return nil, 0, err
	}

	return members, total, nil
}

// FindByUserID 根据用户ID查询加入的群列表（分页）
func (m *defaultGroupMemberModel) FindByUserID(ctx context.Context, userID string, page, pageSize int32) ([]*GroupMember, int64, error) {
	var members []*GroupMember
	var total int64

	db := m.db.WithContext(ctx).
		Where("user_id = ? AND status = 1", userID)

	// 查询总数
	if err := db.Model(&GroupMember{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := db.Offset(int(offset)).Limit(int(pageSize)).Find(&members).Error; err != nil {
		return nil, 0, err
	}

	return members, total, nil
}

// Delete 删除群成员（软删除）
func (m *defaultGroupMemberModel) Delete(ctx context.Context, groupID, userID string) error {
	now := time.Now()
	return m.db.WithContext(ctx).
		Model(&GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Updates(map[string]interface{}{
			"status":  2,
			"left_at": &now,
		}).Error
}

// UpdateRole 更新成员角色
func (m *defaultGroupMemberModel) UpdateRole(ctx context.Context, groupID, userID string, role int8) error {
	return m.db.WithContext(ctx).
		Model(&GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Update("role", role).Error
}

// UpdateStatus 更新成员状态
func (m *defaultGroupMemberModel) UpdateStatus(ctx context.Context, groupID, userID string, status int8) error {
	return m.db.WithContext(ctx).
		Model(&GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Update("status", status).Error
}
