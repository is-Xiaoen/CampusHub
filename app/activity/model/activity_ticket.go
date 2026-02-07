package model

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// ==================== 票据状态 ====================

const (
	TicketStatusUnused  int8 = 0 // 未使用
	TicketStatusUsed    int8 = 1 // 已使用
	TicketStatusExpired int8 = 2 // 已过期
	TicketStatusVoid    int8 = 3 // 已作废
)

// ==================== 错误定义 ====================

var (
	ErrTicketNotFound = errors.New("票据不存在")
)

const (
	emptyCheckInSnapshotJSON = "{}"
)

// ==================== ActivityTicket 票据模型 ====================

type ActivityTicket struct {
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	TicketCode string `gorm:"type:varchar(32);uniqueIndex:uk_ticket_code;not null;comment:票据短码" json:"ticket_code"`
	TicketUUID string `gorm:"type:char(36);uniqueIndex:uk_ticket_uuid;not null;comment:票据UUID" json:"ticket_uuid"`

	ActivityID     uint64 `gorm:"index:idx_activity_user,priority:1;not null;comment:活动ID" json:"activity_id"`
	UserID         uint64 `gorm:"index:idx_activity_user,priority:2;not null;comment:用户ID" json:"user_id"`
	RegistrationID uint64 `gorm:"uniqueIndex:uk_registration_id;not null;comment:关联报名记录ID" json:"registration_id"`

	TotpSecret  string `gorm:"type:varchar(64);comment:TOTP密钥" json:"-"`
	TotpEnabled bool   `gorm:"default:true;comment:是否启用TOTP" json:"totp_enabled"`

	ValidStartTime int64 `gorm:"default:0;comment:可核销开始时间" json:"valid_start_time"`
	ValidEndTime   int64 `gorm:"default:0;comment:可核销截止时间" json:"valid_end_time"`

	Status          int8   `gorm:"default:0;index;comment:票据状态" json:"status"`
	UsedTime        int64  `gorm:"default:0;comment:核销时间" json:"used_time"`
	UsedLocation    string `gorm:"type:varchar(200);default:'';comment:核销地点" json:"used_location"`
	CheckInSnapshot string `gorm:"type:json;comment:核销快照" json:"check_in_snapshot"`

	CreatedAt int64          `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt int64          `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ActivityTicket) TableName() string {
	return "activity_tickets"
}

// TicketDetail 票券详情（查询用）
type TicketDetail struct {
	TicketID     uint64 `json:"ticket_id"`
	TicketCode   string `json:"ticket_code"`
	ActivityID   uint64 `json:"activity_id"`
	ActivityTime int64  `gorm:"column:activity_time" json:"activity_time"`
	QrCodeURL    string `gorm:"column:qr_code_url" json:"qr_code_url"`
}

// ==================== ActivityTicketModel 数据访问层 ====================

type ActivityTicketModel struct {
	db *gorm.DB
}

func NewActivityTicketModel(db *gorm.DB) *ActivityTicketModel {
	return &ActivityTicketModel{db: db}
}

// Create 创建票据
func (m *ActivityTicketModel) Create(ctx context.Context, ticket *ActivityTicket) error {
	return m.db.WithContext(ctx).Create(ticket).Error
}

// FindByID 根据ID查询
func (m *ActivityTicketModel) FindByID(ctx context.Context, id uint64) (*ActivityTicket, error) {
	var ticket ActivityTicket
	err := m.db.WithContext(ctx).Where("id = ?", id).First(&ticket).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}
	return &ticket, nil
}

// FindByCode 根据票据码查询
func (m *ActivityTicketModel) FindByCode(ctx context.Context, ticketCode string) (*ActivityTicket, error) {
	var ticket ActivityTicket
	err := m.db.WithContext(ctx).
		Where("ticket_code = ?", ticketCode).
		First(&ticket).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}
	return &ticket, nil
}

// FindByUUID 根据UUID查询
func (m *ActivityTicketModel) FindByUUID(ctx context.Context, ticketUUID string) (*ActivityTicket, error) {
	var ticket ActivityTicket
	err := m.db.WithContext(ctx).
		Where("ticket_uuid = ?", ticketUUID).
		First(&ticket).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}
	return &ticket, nil
}

// FindByRegistrationID 根据报名记录ID查询
func (m *ActivityTicketModel) FindByRegistrationID(ctx context.Context, registrationID uint64) (*ActivityTicket, error) {
	var ticket ActivityTicket
	err := m.db.WithContext(ctx).
		Where("registration_id = ?", registrationID).
		First(&ticket).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}
	return &ticket, nil
}

// FindByRegistrationIDTx 根据报名记录ID查询（事务内）
func (m *ActivityTicketModel) FindByRegistrationIDTx(ctx context.Context, tx *gorm.DB, registrationID uint64) (*ActivityTicket, error) {
	if tx == nil {
		return nil, errors.New("tx is nil")
	}
	var ticket ActivityTicket
	err := tx.WithContext(ctx).
		Where("registration_id = ?", registrationID).
		First(&ticket).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}
	return &ticket, nil
}

// FindDetailByTicketID 根据票据ID获取票据与活动详情
func (m *ActivityTicketModel) FindDetailByTicketID(ctx context.Context, ticketID uint64) (*TicketDetail, error) {
	var detail TicketDetail
	err := m.db.WithContext(ctx).
		Table("activity_tickets t").
		Select(
			"t.id AS ticket_id, t.ticket_code, t.activity_id, "+
				"a.activity_start_time AS activity_time, t.ticket_uuid AS qr_code_url",
		).
		Joins("INNER JOIN activities a ON a.id = t.activity_id").
		Where("t.id = ?", ticketID).
		Take(&detail).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}
	return &detail, nil
}

// ListByUserID 获取用户票据列表
func (m *ActivityTicketModel) ListByUserID(ctx context.Context, userID uint64, offset, limit int) ([]ActivityTicket, error) {
	var tickets []ActivityTicket
	err := m.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&tickets).Error
	return tickets, err
}

// CountByUserID 统计用户票据数量
func (m *ActivityTicketModel) CountByUserID(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&ActivityTicket{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

// MarkUsed 核销票据
func (m *ActivityTicketModel) MarkUsed(ctx context.Context, id uint64, usedTime int64, usedLocation, snapshot string) error {
	result := m.db.WithContext(ctx).
		Model(&ActivityTicket{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":            TicketStatusUsed,
			"used_time":         usedTime,
			"used_location":     usedLocation,
			"check_in_snapshot": snapshot,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTicketNotFound
	}
	return nil
}

// UpdateStatus 更新票据状态
func (m *ActivityTicketModel) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	result := m.db.WithContext(ctx).
		Model(&ActivityTicket{}).
		Where("id = ?", id).
		Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTicketNotFound
	}
	return nil
}

// ResetForReuse 重置票据为可用状态（事务内）
func (m *ActivityTicketModel) ResetForReuse(ctx context.Context, tx *gorm.DB, id uint64) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	result := tx.WithContext(ctx).
		Model(&ActivityTicket{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":            TicketStatusUnused,
			"used_time":         int64(0),
			"used_location":     "",
			"check_in_snapshot": emptyCheckInSnapshotJSON,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTicketNotFound
	}
	return nil
}
