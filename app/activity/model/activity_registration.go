package model

import (
	"context"
	"errors"

	mysqlerr "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// ==================== 报名状态 ====================

const (
	RegistrationStatusSuccess  int8 = 1 // 报名成功
	RegistrationStatusCanceled int8 = 2 // 取消报名
	RegistrationStatusFailed   int8 = 3 // 报名失败
)

// ==================== 参加状态（前端筛选） ====================

const (
	AttendStatusNotJoined int8 = 0 // 报名未参加
	AttendStatusJoined    int8 = 1 // 报名已参加
)

// ==================== 错误定义 ====================

var (
	ErrRegistrationNotFound = errors.New("报名记录不存在")
	ErrAttendStatusInvalid  = errors.New("参加状态无效")
	ErrActivityQuotaFull    = errors.New("活动名额已满")
)

// TicketPayload 票券生成入参
type TicketPayload struct {
	TicketCode string
	TicketUUID string
	TotpSecret string
}

// RegisterWithTicketResult 报名结果
type RegisterWithTicketResult struct {
	AlreadyRegistered bool
}

// ==================== ActivityRegistration 报名记录模型 ====================

type ActivityRegistration struct {
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	ActivityID uint64 `gorm:"uniqueIndex:uk_activity_user,priority:1;index:idx_activity_id;not null;comment:活动ID" json:"activity_id"`
	UserID     uint64 `gorm:"uniqueIndex:uk_activity_user,priority:2;index:idx_user_id;not null;comment:用户ID" json:"user_id"`

	Status     int8  `gorm:"default:1;comment:报名状态: 1成功 2取消 3失败" json:"status"`
	CancelTime int64 `gorm:"default:0;comment:取消时间" json:"cancel_time"`

	CreatedAt int64 `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt int64 `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ActivityRegistration) TableName() string {
	return "activity_registrations"
}

// ==================== ActivityRegistrationModel 数据访问层 ====================

type ActivityRegistrationModel struct {
	db *gorm.DB
}

func NewActivityRegistrationModel(db *gorm.DB) *ActivityRegistrationModel {
	return &ActivityRegistrationModel{db: db}
}

// Create 创建报名记录
func (m *ActivityRegistrationModel) Create(ctx context.Context, reg *ActivityRegistration) error {
	return m.db.WithContext(ctx).Create(reg).Error
}

// FindByID 根据ID查询
func (m *ActivityRegistrationModel) FindByID(ctx context.Context, id uint64) (*ActivityRegistration, error) {
	var reg ActivityRegistration
	err := m.db.WithContext(ctx).Where("id = ?", id).First(&reg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRegistrationNotFound
		}
		return nil, err
	}
	return &reg, nil
}

// FindByActivityUser 根据活动ID和用户ID查询
func (m *ActivityRegistrationModel) FindByActivityUser(ctx context.Context, activityID, userID uint64) (*ActivityRegistration, error) {
	var reg ActivityRegistration
	err := m.db.WithContext(ctx).
		Where("activity_id = ? AND user_id = ?", activityID, userID).
		First(&reg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRegistrationNotFound
		}
		return nil, err
	}
	return &reg, nil
}

// FindByActivityUserTx 根据活动ID和用户ID查询（事务内）
func (m *ActivityRegistrationModel) FindByActivityUserTx(ctx context.Context, tx *gorm.DB, activityID, userID uint64) (*ActivityRegistration, error) {
	if tx == nil {
		return nil, errors.New("tx is nil")
	}
	var reg ActivityRegistration
	err := tx.WithContext(ctx).
		Where("activity_id = ? AND user_id = ?", activityID, userID).
		First(&reg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRegistrationNotFound
		}
		return nil, err
	}
	return &reg, nil
}

// ExistsByActivityUser 判断是否已报名
func (m *ActivityRegistrationModel) ExistsByActivityUser(ctx context.Context, activityID, userID uint64) (bool, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&ActivityRegistration{}).
		Where("activity_id = ? AND user_id = ?", activityID, userID).
		Count(&count).Error
	return count > 0, err
}

// ListByUserID 获取用户报名记录列表
func (m *ActivityRegistrationModel) ListByUserID(ctx context.Context, userID uint64, offset, limit int) ([]ActivityRegistration, error) {
	var regs []ActivityRegistration
	err := m.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&regs).Error
	return regs, err
}

// ListByUserAttendStatus 获取用户报名但未参加/已参加的活动列表
func (m *ActivityRegistrationModel) ListByUserAttendStatus(ctx context.Context, userID uint64, attendStatus int, offset, limit int) ([]ActivityRegistration, error) {
	var ticketStatus int8
	switch attendStatus {
	case int(AttendStatusNotJoined):
		ticketStatus = TicketStatusUnused
	case int(AttendStatusJoined):
		ticketStatus = TicketStatusUsed
	default:
		return nil, ErrAttendStatusInvalid
	}

	var regs []ActivityRegistration
	err := m.db.WithContext(ctx).
		Table("activity_registrations r").
		Select("r.*").
		Joins("INNER JOIN activity_tickets t ON t.registration_id = r.id").
		Where("r.user_id = ? AND r.status = ? AND t.status = ?", userID, RegistrationStatusSuccess, ticketStatus).
		Order("r.created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&regs).Error
	return regs, err
}

// ListByActivityID 获取活动报名记录列表
func (m *ActivityRegistrationModel) ListByActivityID(ctx context.Context, activityID uint64, offset, limit int) ([]ActivityRegistration, error) {
	var regs []ActivityRegistration
	err := m.db.WithContext(ctx).
		Where("activity_id = ?", activityID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&regs).Error
	return regs, err
}

// CountByUserID 统计用户报名记录数量
func (m *ActivityRegistrationModel) CountByUserID(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&ActivityRegistration{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

// CountByUserAttendStatus 统计用户报名但未参加/已参加数量
func (m *ActivityRegistrationModel) CountByUserAttendStatus(ctx context.Context, userID uint64, attendStatus int) (int64, error) {
	var ticketStatus int8
	switch attendStatus {
	case int(AttendStatusNotJoined):
		ticketStatus = TicketStatusUnused
	case int(AttendStatusJoined):
		ticketStatus = TicketStatusUsed
	default:
		return 0, ErrAttendStatusInvalid
	}

	var count int64
	err := m.db.WithContext(ctx).
		Table("activity_registrations r").
		Joins("INNER JOIN activity_tickets t ON t.registration_id = r.id").
		Where("r.user_id = ? AND r.status = ? AND t.status = ?", userID, RegistrationStatusSuccess, ticketStatus).
		Count(&count).Error
	return count, err
}

// CountByActivityID 统计活动报名记录数量
func (m *ActivityRegistrationModel) CountByActivityID(ctx context.Context, activityID uint64) (int64, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&ActivityRegistration{}).
		Where("activity_id = ?", activityID).
		Count(&count).Error
	return count, err
}

// ReopenIfCanceledOrFailed 将取消/失败报名恢复为成功（事务内）
func (m *ActivityRegistrationModel) ReopenIfCanceledOrFailed(ctx context.Context, tx *gorm.DB, id uint64) (bool, error) {
	if tx == nil {
		return false, errors.New("tx is nil")
	}
	result := tx.WithContext(ctx).
		Model(&ActivityRegistration{}).
		Where("id = ? AND status IN ?", id, []int8{RegistrationStatusCanceled, RegistrationStatusFailed}).
		Updates(map[string]interface{}{
			"status":      RegistrationStatusSuccess,
			"cancel_time": int64(0),
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// OccupyActivityQuota 报名成功占用名额（当前报名人数+1）
func (m *ActivityRegistrationModel) OccupyActivityQuota(ctx context.Context, tx *gorm.DB, activityID uint64) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	result := tx.WithContext(ctx).
		Model(&Activity{}).
		Where("id = ? AND (max_participants = 0 OR current_participants < max_participants)", activityID).
		Update("current_participants", gorm.Expr("current_participants + 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrActivityQuotaFull
	}
	return nil
}

// RegisterWithTicket 创建或恢复报名并生成票券（事务内）
func (m *ActivityRegistrationModel) RegisterWithTicket(
	ctx context.Context,
	activityID,
	userID uint64,
	gen func() (*TicketPayload, error),
) (*RegisterWithTicketResult, error) {
	if gen == nil {
		return nil, errors.New("ticket generator is nil")
	}
	result := &RegisterWithTicketResult{}
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		reg := &ActivityRegistration{
			ActivityID: activityID,
			UserID:     userID,
			Status:     RegistrationStatusSuccess,
		}
		if err := tx.Create(reg).Error; err != nil {
			if !isDuplicateKeyErr(err) {
				return err
			}
			existing, err := m.FindByActivityUserTx(ctx, tx, activityID, userID)
			if err != nil {
				return err
			}
			if existing.Status == RegistrationStatusSuccess {
				result.AlreadyRegistered = true
				return nil
			}
			if existing.Status != RegistrationStatusCanceled && existing.Status != RegistrationStatusFailed {
				result.AlreadyRegistered = true
				return nil
			}
			reopened, err := m.ReopenIfCanceledOrFailed(ctx, tx, existing.ID)
			if err != nil {
				return err
			}
			if !reopened {
				result.AlreadyRegistered = true
				return nil
			}
			reg = existing
		}

		if result.AlreadyRegistered {
			return nil
		}

		if err := m.OccupyActivityQuota(ctx, tx, activityID); err != nil {
			return err
		}

		// 先尝试复用已有票券（按报名记录ID更新）
		reuse := tx.WithContext(ctx).
			Model(&ActivityTicket{}).
			Where("registration_id = ?", reg.ID).
			Updates(map[string]interface{}{
				"status":            TicketStatusUnused,
				"used_time":         int64(0),
				"used_location":     "",
				"check_in_snapshot": "",
			})
		if reuse.Error != nil {
			return reuse.Error
		}
		if reuse.RowsAffected > 0 {
			return nil
		}

		for i := 0; i < 3; i++ {
			payload, err := gen()
			if err != nil {
				return err
			}
			ticket := &ActivityTicket{
				ActivityID:     activityID,
				UserID:         userID,
				RegistrationID: reg.ID,
				TicketCode:     payload.TicketCode,
				TicketUUID:     payload.TicketUUID,
				TotpSecret:     payload.TotpSecret,
				Status:         TicketStatusUnused,
			}
			if err := tx.Create(ticket).Error; err != nil {
				if isDuplicateKeyErr(err) {
					continue
				}
				return err
			}
			return nil
		}

		return errors.New("生成票据失败")
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateStatus 更新报名状态
func (m *ActivityRegistrationModel) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	result := m.db.WithContext(ctx).
		Model(&ActivityRegistration{}).
		Where("id = ?", id).
		Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRegistrationNotFound
	}
	return nil
}

func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var mysqlErr *mysqlerr.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}
	return false
}

// Cancel 取消报名（更新状态和取消时间）
func (m *ActivityRegistrationModel) Cancel(ctx context.Context, activityID, userID uint64, cancelTime int64) error {
	result := m.db.WithContext(ctx).
		Model(&ActivityRegistration{}).
		Where("activity_id = ? AND user_id = ?", activityID, userID).
		Updates(map[string]interface{}{
			"status":      RegistrationStatusCanceled,
			"cancel_time": cancelTime,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRegistrationNotFound
	}
	return nil
}
