package model

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ==================== 错误定义 ====================

var (
	ErrActivityNotFound         = errors.New("活动不存在")
	ErrActivityStatusInvalid    = errors.New("活动状态不允许此操作")
	ErrActivityConcurrentUpdate = errors.New("并发更新冲突，请重试")
	ErrPageTooDeep              = errors.New("不支持查看超过100页的数据，请使用搜索功能")
)

// ==================== Activity 活动模型 ====================

type Activity struct {
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 基本信息
	Title       string `gorm:"type:varchar(100);not null;comment:活动标题" json:"title"`
	CoverURL    string `gorm:"type:varchar(500);not null;comment:封面URL" json:"cover_url"`
	CoverType   int8   `gorm:"default:1;comment:封面类型: 1图片 2视频"  json:"cover_type"`
	Description string `gorm:"type:text;comment:活动详情(富文本)" json:"description"`
	CategoryID  uint64 `gorm:"index:idx_category_status,priority:1;not null;comment:分类ID" json:"category_id"`

	// 组织者信息（冗余存储，避免联表查询）
	OrganizerID     uint64 `gorm:"index;not null;comment:组织者用户ID" json:"organizer_id"`
	OrganizerName   string `gorm:"type:varchar(50);not null;comment:组织者名称" json:"organizer_name"`
	OrganizerAvatar string `gorm:"type:varchar(500);default:'';comment:组织者头像" json:"organizer_avatar"`
	ContactPhone    string `gorm:"type:varchar(20);default:'';comment:联系电话" json:"contact_phone"`

	// 时间信息
	RegisterStartTime int64 `gorm:"not null;comment:报名开始时间" json:"register_start_time"`
	RegisterEndTime   int64 `gorm:"not null;comment:报名截止时间" json:"register_end_time"`
	ActivityStartTime int64 `gorm:"index:idx_status_start,priority:2;not null;comment:活动开始时间" json:"activity_start_time"`
	ActivityEndTime   int64 `gorm:"not null;comment:活动结束时间" json:"activity_end_time"`

	// 地点信息
	Location      string  `gorm:"type:varchar(200);not null;comment:活动地点" json:"location"`
	AddressDetail string  `gorm:"type:varchar(500);default:'';comment:详细地址" json:"address_detail"`
	Longitude     float64 `gorm:"type:decimal(10,7);comment:经度" json:"longitude"`
	Latitude      float64 `gorm:"type:decimal(10,7);comment:纬度" json:"latitude"`

	// 名额与报名规则
	MaxParticipants      uint32 `gorm:"default:0;comment:最大参与人数(0=不限)" json:"max_participants"`
	CurrentParticipants  uint32 `gorm:"default:0;comment:当前报名人数" json:"current_participants"`
	RequireApproval      bool   `gorm:"default:false;comment:是否需要审批" json:"require_approval"`
	RequireStudentVerify bool   `gorm:"default:false;comment:是否需要学生认证" json:"require_student_verify"`
	MinCreditScore       int    `gorm:"default:0;comment:最低信用分要求" json:"min_credit_score"`
	// 状态
	Status       int8   `gorm:"default:0;index:idx_category_status,priority:2;index:idx_status_start,priority:1;comment:状态" json:"status"`
	RejectReason string `gorm:"type:varchar(500);default:'';comment:拒绝原因"  json:"reject_reason"`
	// 统计（异步更新）
	ViewCount uint32 `gorm:"default:0;comment:浏览量" json:"view_count"`
	LikeCount uint32 `gorm:"default:0;comment:点赞数" json:"like_count"`
	// 乐观锁
	Version uint32 `gorm:"default:0;comment:乐观锁版本号" json:"version"`
	// 时间戳
	CreatedAt int64          `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt int64          `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联数据（非数据库字段）
	Tags         []TagCache `gorm:"-" json:"tags,omitempty"`
	CategoryName string     `gorm:"-" json:"category_name,omitempty"`
}

func (Activity) TableName() string {
	return "activities"
}

// StatusText 获取状态文本
func (a *Activity) StatusText() string {
	if text, ok := StatusText[a.Status]; ok {
		return text
	}
	return "未知"
}

// CanEdit 判断是否可编辑
func (a *Activity) CanEdit() bool {
	return a.Status == StatusDraft || a.Status == StatusPending || a.Status == StatusRejected
}

// IsPublic 判断是否公开可见
func (a *Activity) IsPublic() bool {
	return a.Status == StatusPublished || a.Status ==
		StatusOngoing || a.Status == StatusFinished
}

// ==================== ActivityModel 数据访问层

type ActivityModel struct {
	db *gorm.DB
}

func NewActivityModel(db *gorm.DB) *ActivityModel {
	return &ActivityModel{db: db}
}

// Create 创建活动
func (m *ActivityModel) Create(ctx context.Context, activity *Activity) error {
	return m.db.WithContext(ctx).Create(activity).Error
}

// FindByID 根据ID查询（包含软删除检查）
func (m *ActivityModel) FindByID(ctx context.Context, id uint64) (*Activity, error) {
	var activity Activity
	err := m.db.WithContext(ctx).
		Where("id = ?", id).
		First(&activity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrActivityNotFound
		}
		return nil, err
	}
	return &activity, nil
}

// FindByIDForUpdate 查询并加行锁（用于状态机）
func (m *ActivityModel) FindByIDForUpdate(ctx context.Context, tx *gorm.DB, id uint64) (*Activity, error) {
	var activity Activity
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&activity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrActivityNotFound
		}
		return nil, err
	}
	return &activity, nil
}

// Update 更新活动（带乐观锁）
func (m *ActivityModel) Update(ctx context.Context, activity *Activity) error {
	result := m.db.WithContext(ctx).
		Model(&Activity{}).
		Where("id = ? AND version = ?", activity.ID,
			activity.Version).
		Updates(map[string]interface{}{
			"title":                  activity.Title,
			"cover_url":              activity.CoverURL,
			"cover_type":             activity.CoverType,
			"description":            activity.Description,
			"category_id":            activity.CategoryID,
			"register_start_time":    activity.RegisterStartTime,
			"register_end_time":      activity.RegisterEndTime,
			"activity_start_time":    activity.ActivityStartTime,
			"activity_end_time":      activity.ActivityEndTime,
			"location":               activity.Location,
			"address_detail":         activity.AddressDetail,
			"longitude":              activity.Longitude,
			"latitude":               activity.Latitude,
			"max_participants":       activity.MaxParticipants,
			"require_approval":       activity.RequireApproval,
			"require_student_verify": activity.RequireStudentVerify,
			"min_credit_score":       activity.MinCreditScore,
			"contact_phone":          activity.ContactPhone,
			"version":                gorm.Expr("version + 1"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrActivityConcurrentUpdate
	}
	return nil
}

// UpdateStatus 更新状态（在事务内使用）
func (m *ActivityModel) UpdateStatus(ctx context.Context, tx *gorm.DB, id uint64, oldVersion uint32, newStatus int8, reason string) error {
	result := tx.WithContext(ctx).
		Model(&Activity{}).
		Where("id = ? AND version = ?", id, oldVersion).
		Updates(map[string]interface{}{
			"status":        newStatus,
			"reject_reason": reason,
			"version":       gorm.Expr("version + 1"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrActivityConcurrentUpdate
	}
	return nil
}

// SoftDelete 软删除
func (m *ActivityModel) SoftDelete(ctx context.Context, id uint64) error {
	return m.db.WithContext(ctx).Delete(&Activity{}, id).Error
}

// ==================== 列表查询 ====================

// ListQuery 列表查询条件
type ListQuery struct {
	Pagination
	CategoryID  uint64 // 0 = 全部
	Status      int    // -1 = 公开状态(2,3,4), -2 = 全部(需organizer_id), 具体值 = 筛选该状态
	OrganizerID uint64 // 0 = 全部
	Sort        string // created_at(默认) / hot / start_time
}

// ListResult 列表查询结果
type ListResult struct {
	List       []Activity
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

// List 分页列表查询
func (m *ActivityModel) List(ctx context.Context, query *ListQuery) (*ListResult, error) {
	query.Pagination.Normalize()

	// 禁止超深分页
	if query.Page > MaxPage {
		return nil, ErrPageTooDeep
	}

	db := m.db.WithContext(ctx).Model(&Activity{})

	// 构建查询条件
	db = m.buildListConditions(db, query)

	// 统计总数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// 深分页优化
	var activities []Activity
	if query.Page > DeepPageThreshold {
		activities, _ = m.listWithDeepPageOptimize(ctx, query, db)
	} else {
		// 普通分页
		db = m.buildListOrder(db, query.Sort)
		if err := db.Offset(query.Offset()).Limit(query.PageSize).Find(&activities).Error; err != nil {
			return nil, err
		}
	}

	totalPages := int(total) / query.PageSize
	if int(total)%query.PageSize > 0 {
		totalPages++
	}

	return &ListResult{
		List:       activities,
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: totalPages,
	}, nil
}

// buildListConditions 构建查询条件
func (m *ActivityModel) buildListConditions(db *gorm.DB, query *ListQuery) *gorm.DB {
	// 分类筛选
	if query.CategoryID > 0 {
		db = db.Where("category_id = ?", query.CategoryID)
	}

	// 状态筛选
	switch query.Status {
	case -1: // 公开状态
		db = db.Where("status IN ?", []int8{StatusPublished,
			StatusOngoing, StatusFinished})
	case -2: // 全部（需要 organizer_id）
		// 不加状态条件
	default:
		if query.Status >= 0 {
			db = db.Where("status = ?", query.Status)
		}
	}

	// 组织者筛选
	if query.OrganizerID > 0 {
		db = db.Where("organizer_id = ?", query.OrganizerID)
	}

	return db
}

// buildListOrder 构建排序
func (m *ActivityModel) buildListOrder(db *gorm.DB, sort string) *gorm.DB {
	switch sort {
	case "hot":
		return db.Order("current_participants DESC, created_at DESC")
	case "start_time":
		return db.Order("activity_start_time ASC")
	default: // created_at
		return db.Order("created_at DESC")
	}
}

// listWithDeepPageOptimize 深分页优化（延迟关联）
func (m *ActivityModel) listWithDeepPageOptimize(ctx context.Context, query *ListQuery, db *gorm.DB) ([]Activity,
	error) {
	// 1. 先只查 ID（利用覆盖索引）
	var ids []uint64
	subQuery := db.Session(&gorm.Session{})
	subQuery = m.buildListOrder(subQuery, query.Sort)
	if err :=
		subQuery.Offset(query.Offset()).Limit(query.PageSize).Pluck("id",
			&ids).Error; err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return []Activity{}, nil
	}

	// 2. 根据 ID 批量获取完整数据
	var activities []Activity
	if err := m.db.WithContext(ctx).Where("id IN ?",
		ids).Find(&activities).Error; err != nil {
		return nil, err
	}

	// 3. 按原顺序排序
	activityMap := make(map[uint64]*Activity, len(activities))
	for i := range activities {
		activityMap[activities[i].ID] = &activities[i]
	}
	result := make([]Activity, 0, len(ids))
	for _, id := range ids {
		if a, ok := activityMap[id]; ok {
			result = append(result, *a)
		}
	}

	return result, nil
}

// ==================== 热门活动 ====================

// FindHot 获取热门活动（按报名人数）
func (m *ActivityModel) FindHot(ctx context.Context, limit int) ([]Activity, error) {
	if limit <= 0 || limit > 20 {
		limit = 10
	}

	now := time.Now().Unix()
	var activities []Activity
	err := m.db.WithContext(ctx).
		Where("status IN ? AND activity_end_time > ?",
			[]int8{StatusPublished, StatusOngoing}, now).
		Order("current_participants DESC, created_at DESC").
		Limit(limit).
		Find(&activities).Error
	return activities, err
}

// ==================== 内部服务方法 ====================

// UpdateParticipantCount 更新报名人数（原子操作，供报名模块调用）
func (m *ActivityModel) UpdateParticipantCount(ctx context.Context, id uint64, delta int) (uint32, error) {
	var activity Activity
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 原子更新
		result := tx.Model(&Activity{}).
			Where("id = ?", id).
			Update("current_participants",
				gorm.Expr("current_participants + ?", delta))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrActivityNotFound
		}

		// 2. 查询更新后的值
		if err := tx.Where("id = ?", id).First(&activity).Error; err != nil {
			return err
		}

		// 3. 边界检查
		if activity.CurrentParticipants < 0 {
			return errors.New("报名人数不能为负")
		}
		if activity.MaxParticipants > 0 &&
			activity.CurrentParticipants > activity.MaxParticipants {
			return errors.New("超过人数上限")
		}

		return nil
	})

	return activity.CurrentParticipants, err
}

// IncrViewCount 增加浏览量（原子操作）
func (m *ActivityModel) IncrViewCount(ctx context.Context, id uint64, delta int) error {
	return m.db.WithContext(ctx).
		Model(&Activity{}).
		Where("id = ?", id).
		Update("view_count", gorm.Expr("view_count + ?",
			delta)).Error
}

// ==================== 定时任务方法 ====================

// BatchUpdateStatusByTime 批量更新状态（定时任务用）
func (m *ActivityModel) BatchUpdateStatusByTime(ctx context.Context, fromStatus, toStatus int8, timeField string,
	beforeTime int64, batchSize int) (int64, error) {
	var totalAffected int64

	for {
		result := m.db.WithContext(ctx).
			Model(&Activity{}).
			Where("status = ? AND "+timeField+" <= ?",
				fromStatus, beforeTime).
			Limit(batchSize).
			Update("status", toStatus)

		if result.Error != nil {
			return totalAffected, result.Error
		}

		totalAffected += result.RowsAffected

		if result.RowsAffected == 0 {
			break
		}

		// 短暂休眠，避免数据库压力
		time.Sleep(100 * time.Millisecond)
	}

	return totalAffected, nil
}
