package model

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// ==================== 错误定义 ====================

var (
	ErrCheckInRecordNotFound = errors.New("核销记录不存在")
)

// ==================== CheckInRecord 核销记录模型 ====================

type CheckInRecord struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	CheckInNo string `gorm:"type:varchar(64);uniqueIndex:uk_check_in_no;not null;comment:核销流水号" json:"check_in_no"`

	TicketID   uint64 `gorm:"index:idx_ticket_id;not null;comment:票据ID" json:"ticket_id"`
	TicketCode string `gorm:"type:varchar(32);not null;comment:票据短码" json:"ticket_code"`
	ActivityID uint64 `gorm:"index:idx_activity_id;not null;comment:活动ID" json:"activity_id"`
	UserID     uint64 `gorm:"index:idx_user_id;not null;comment:用户ID" json:"user_id"`

	CheckInTime int64 `gorm:"default:0;comment:核销时间" json:"check_in_time"`

	Longitude float64 `gorm:"type:decimal(10,7);comment:经度" json:"longitude"`
	Latitude  float64 `gorm:"type:decimal(10,7);comment:纬度" json:"latitude"`

	ClientRequestID string `gorm:"type:varchar(64);uniqueIndex:uk_client_request_id;not null;comment:请求ID(幂等)" json:"client_request_id"`

	CheckInSnapshot string `gorm:"type:text;comment:核销快照" json:"check_in_snapshot"`

	CreatedAt int64          `gorm:"autoCreateTime;index" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (CheckInRecord) TableName() string {
	return "check_in_records"
}

// ==================== CheckInRecordModel 数据访问层 ====================

type CheckInRecordModel struct {
	db *gorm.DB
}

func NewCheckInRecordModel(db *gorm.DB) *CheckInRecordModel {
	return &CheckInRecordModel{db: db}
}

// Create 创建核销记录
func (m *CheckInRecordModel) Create(ctx context.Context, record *CheckInRecord) error {
	return m.db.WithContext(ctx).Create(record).Error
}

// FindByID 根据ID查询
func (m *CheckInRecordModel) FindByID(ctx context.Context, id uint64) (*CheckInRecord, error) {
	var record CheckInRecord
	err := m.db.WithContext(ctx).Where("id = ?", id).First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCheckInRecordNotFound
		}
		return nil, err
	}
	return &record, nil
}

// FindByCheckInNo 根据核销流水号查询
func (m *CheckInRecordModel) FindByCheckInNo(ctx context.Context, checkInNo string) (*CheckInRecord, error) {
	var record CheckInRecord
	err := m.db.WithContext(ctx).
		Where("check_in_no = ?", checkInNo).
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCheckInRecordNotFound
		}
		return nil, err
	}
	return &record, nil
}

// FindByClientRequestID 根据请求ID查询（幂等）
func (m *CheckInRecordModel) FindByClientRequestID(ctx context.Context, clientRequestID string) (*CheckInRecord, error) {
	var record CheckInRecord
	err := m.db.WithContext(ctx).
		Where("client_request_id = ?", clientRequestID).
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCheckInRecordNotFound
		}
		return nil, err
	}
	return &record, nil
}

// ExistsByClientRequestID 判断请求是否已处理（幂等检查）
func (m *CheckInRecordModel) ExistsByClientRequestID(ctx context.Context, clientRequestID string) (bool, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&CheckInRecord{}).
		Where("client_request_id = ?", clientRequestID).
		Count(&count).Error
	return count > 0, err
}

// ListByActivityID 获取活动的核销记录列表
func (m *CheckInRecordModel) ListByActivityID(ctx context.Context, activityID uint64, offset, limit int) ([]CheckInRecord, error) {
	var records []CheckInRecord
	err := m.db.WithContext(ctx).
		Where("activity_id = ?", activityID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&records).Error
	return records, err
}

// ListByTicketID 获取票据的核销记录列表
func (m *CheckInRecordModel) ListByTicketID(ctx context.Context, ticketID uint64, offset, limit int) ([]CheckInRecord, error) {
	var records []CheckInRecord
	err := m.db.WithContext(ctx).
		Where("ticket_id = ?", ticketID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&records).Error
	return records, err
}

// CountByActivityID 统计活动核销记录数量
func (m *CheckInRecordModel) CountByActivityID(ctx context.Context, activityID uint64) (int64, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&CheckInRecord{}).
		Where("activity_id = ?", activityID).
		Count(&count).Error
	return count, err
}
