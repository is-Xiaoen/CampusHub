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
var StatusText = map[int8]string{
	StatusDraft:     "草稿",
	StatusPending:   "待审核",
	StatusPublished: "报名中",
	StatusOngoing:   "进行中",
	StatusFinished:  "已结束",
	StatusRejected:  "已拒绝",
	StatusCancelled: "已取消",
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
	ID uint64 `gorm:"primaryKey;autoIncrement"    
  json:"id"`
	CreatedAt time.Time `gorm:"autoCreateTime"              
  json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"              
  json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
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
