/**
 * @projectName: CampusHub
 * @package: model
 * @className: SysImage
 * @author: lijunqi
 * @description: 图片资源中心实体及数据访问层
 * @date: 2026-02-10
 * @version: 1.0
 */

package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// SysImageStatus 图片状态
const (
	// SysImageStatusAuditing 审核中
	SysImageStatusAuditing int64 = 0
	// SysImageStatusNormal 正常
	SysImageStatusNormal int64 = 1
	// SysImageStatusBanned 封禁
	SysImageStatusBanned int64 = 2
)

// SysImageBizType 业务类型
const (
	// SysImageBizTypeAvatar 头像
	SysImageBizTypeAvatar = "avatar"
	// SysImageBizTypeActivityCover 活动封面
	SysImageBizTypeActivityCover = "activity_cover"
	// SysImageBizTypeIdentityAuth 身份认证
	SysImageBizTypeIdentityAuth = "identity_auth"
)

// SysImage 图片资源中心表
type SysImage struct {
	// 主键，自增ID
	ID int64 `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	// 图片存储相对路径或完整URL
	URL string `gorm:"column:url;size:500;not null" json:"url"`
	// 原始文件名
	OriginName string `gorm:"column:origin_name;size:255" json:"origin_name"`
	// 业务类型: avatar, activity_cover, identity_auth
	BizType string `gorm:"column:biz_type;size:32;not null" json:"biz_type"`
	// 文件大小(字节)
	FileSize int64 `gorm:"column:file_size;default:0;not null" json:"file_size"`
	// 图片格式: image/jpeg, image/png等
	MimeType string `gorm:"column:mime_type;size:64" json:"mime_type"`
	// 后缀名: jpg, png
	Extension string `gorm:"column:extension;size:10" json:"extension"`
	// 核心字段：引用计数，默认为0
	RefCount int64 `gorm:"column:ref_count;default:0;not null" json:"ref_count"`
	// 上传者用户ID，用于权限校验
	UploaderID int64 `gorm:"column:uploader_id;not null;index:idx_uploader" json:"uploader_id"`
	// 状态: 0-审核中, 1-正常, 2-封禁, 3-待清理
	Status int64 `gorm:"column:status;default:0;not null;index:idx_biz_status" json:"status"`
	// 上传时间
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;not null" json:"created_at"`
	// 最后更新时间
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime;not null" json:"updated_at"`
}

// TableName 指定表名
func (SysImage) TableName() string {
	return "sys_images"
}

// ISysImageModel 图片资源数据访问层接口
type ISysImageModel interface {
	// Create 创建图片记录
	Create(ctx context.Context, image *SysImage) error
	// FindByID 根据ID查询
	FindByID(ctx context.Context, id int64) (*SysImage, error)
	// FindByUploaderID 根据上传者ID查询
	FindByUploaderID(ctx context.Context, uploaderID int64) ([]*SysImage, error)
	// Update 更新图片信息
	Update(ctx context.Context, image *SysImage) error
	// Delete 删除图片记录
	Delete(ctx context.Context, id int64) error
}

// 确保 SysImageModel 实现 ISysImageModel 接口
var _ ISysImageModel = (*SysImageModel)(nil)

// SysImageModel 图片资源数据访问层
type SysImageModel struct {
	db *gorm.DB
}

// NewSysImageModel 创建SysImageModel实例
func NewSysImageModel(db *gorm.DB) ISysImageModel {
	return &SysImageModel{db: db}
}

// Create 创建图片记录
func (m *SysImageModel) Create(ctx context.Context, image *SysImage) error {
	return m.db.WithContext(ctx).Create(image).Error
}

// FindByID 根据ID查询
func (m *SysImageModel) FindByID(ctx context.Context, id int64) (*SysImage, error) {
	var image SysImage
	err := m.db.WithContext(ctx).First(&image, id).Error
	if err != nil {
		return nil, err
	}
	return &image, nil
}

// FindByUploaderID 根据上传者ID查询
func (m *SysImageModel) FindByUploaderID(ctx context.Context, uploaderID int64) ([]*SysImage, error) {
	var images []*SysImage
	err := m.db.WithContext(ctx).Where("uploader_id = ?", uploaderID).Find(&images).Error
	if err != nil {
		return nil, err
	}
	return images, nil
}

// Update 更新图片信息
func (m *SysImageModel) Update(ctx context.Context, image *SysImage) error {
	return m.db.WithContext(ctx).Save(image).Error
}

// Delete 删除图片记录
func (m *SysImageModel) Delete(ctx context.Context, id int64) error {
	return m.db.WithContext(ctx).Delete(&SysImage{}, id).Error
}
