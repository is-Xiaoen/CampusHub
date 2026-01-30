package model

import (
	"context"

	"gorm.io/gorm"
)

// ActivityStatusLog 活动状态变更日志
type ActivityStatusLog struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement"          json:"id"`
	ActivityID   uint64 `gorm:"index:idx_activity_created,priority:1;not null;comment:活动ID" json:"activity_id"`
	FromStatus   int8   `gorm:"not null;comment:变更前状态"       json:"from_status"`
	ToStatus     int8   `gorm:"not null;comment:变更后状态"       json:"to_status"`
	OperatorID   uint64 `gorm:"not null;comment:操作人ID"         json:"operator_id"`
	OperatorType int8   `gorm:"default:1;comment:操作人类型: 1用户 2管理员 3系统" json:"operator_type"`
	Reason       string `gorm:"type:varchar(500);default:'';comment:变更原因" json:"reason"`
	CreatedAt    int64  `gorm:"autoCreateTime;index:idx_activity_created,priority:2"  json:"created_at"`
}

func (ActivityStatusLog) TableName() string {
	return "activity_status_logs"
}

// ==================== ActivityStatusLogModel 数据访问层

type ActivityStatusLogModel struct {
	db *gorm.DB
}

func NewActivityStatusLogModel(db *gorm.DB) *ActivityStatusLogModel {
	return &ActivityStatusLogModel{db: db}
}

// Create 创建日志（通常在事务内调用）
func (m *ActivityStatusLogModel) Create(ctx context.Context, tx *gorm.DB, log *ActivityStatusLog) error {
	if tx == nil {
		tx = m.db
	}
	return tx.WithContext(ctx).Create(log).Error
}

// FindByActivityID 获取活动的状态变更历史
func (m *ActivityStatusLogModel) FindByActivityID(ctx context.Context, activityID uint64) ([]ActivityStatusLog, error) {
	var logs []ActivityStatusLog
	err := m.db.WithContext(ctx).
		Where("activity_id = ?", activityID).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}
