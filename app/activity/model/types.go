package model

import (
	"time"

	"gorm.io/gorm"
)

//  活动状态常量

const (
	StatusDraft     int8 = 0 // 草稿
	StatusPending   int8 = 1 // 待审核
	StatusPublished int8 = 2 // 已发布(报名中)
	StatusOngoing   int8 = 3 // 进行中
	StatusFinished  int8 = 4 // 已结束
	StatusRejected  int8 = 5 // 已拒绝
	StatusCancelled int8 = 6 // 已取消
)

// StatusText 状态文本映射
//
// 注意：StatusPublished 是活动生命周期状态（"已发布"），不是报名状态
// 报名状态由 ComputeRegistrationStatus 根据时间动态计算
var StatusText = map[int8]string{
	StatusDraft:     "草稿",
	StatusPending:   "待审核",
	StatusPublished: "已发布",
	StatusOngoing:   "进行中",
	StatusFinished:  "已结束",
	StatusRejected:  "已拒绝",
	StatusCancelled: "已取消",
}

// 报名状态常量（动态计算，不持久化）

const (
	RegStatusNotApplicable int32 = 0 // 不适用（草稿/待审核/已拒绝/已取消）
	RegStatusNotStarted    int32 = 1 // 未开始报名
	RegStatusOpen          int32 = 2 // 报名中
	RegStatusClosed        int32 = 3 // 报名已截止
)

// regStatusText 报名状态文本映射
var regStatusText = map[int32]string{
	RegStatusNotApplicable: "",
	RegStatusNotStarted:    "未开始报名",
	RegStatusOpen:          "报名中",
	RegStatusClosed:        "报名已截止",
}

// ComputeRegistrationStatus 根据活动状态和报名时间计算报名状态
//
// 规则：
//   - 非公开状态（草稿/待审核/已拒绝/已取消）：返回 0（不适用）
//   - 已结束：返回 3（报名已截止）
//   - 已发布/进行中：根据当前时间与报名时间窗口判断
func ComputeRegistrationStatus(status int8, registerStartTime, registerEndTime, now int64) (int32, string) {
	// 非公开状态：不适用
	if status == StatusDraft || status == StatusPending ||
		status == StatusRejected || status == StatusCancelled {
		return RegStatusNotApplicable, regStatusText[RegStatusNotApplicable]
	}

	// 已结束：报名已截止
	if status == StatusFinished {
		return RegStatusClosed, regStatusText[RegStatusClosed]
	}

	// 已发布/进行中：根据时间判断
	if now < registerStartTime {
		return RegStatusNotStarted, regStatusText[RegStatusNotStarted]
	}
	if now <= registerEndTime {
		return RegStatusOpen, regStatusText[RegStatusOpen]
	}
	return RegStatusClosed, regStatusText[RegStatusClosed]
}

// 操作人类型

const (
	OperatorTypeUser   int8 = 1 // 用户
	OperatorTypeAdmin  int8 = 2 // 管理员
	OperatorTypeSystem int8 = 3 // 系统自动
)

// 封面类型

const (
	CoverTypeImage int8 = 1 // 图片
	CoverTypeVideo int8 = 2 // 视频
)

// 公共基础模型

// BaseModel 基础模型（所有表共用）
type BaseModel struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// 分页参数

const (
	DefaultPage       = 1
	DefaultPageSize   = 10
	MaxPageSize       = 50
	MaxPage           = 100 // 禁止超过100页
	DeepPageThreshold = 20  // 深分页优化阈值
)

// Pagination 分页请求
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// Normalize 规范化分页参数
func (p *Pagination) Normalize() {
	if p.Page <= 0 {
		p.Page = DefaultPage
	}
	if p.PageSize <= 0 {
		p.PageSize = DefaultPageSize
	}
	if p.PageSize > MaxPageSize {
		p.PageSize = MaxPageSize
	}
}

// Offset 计算偏移量
func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}
