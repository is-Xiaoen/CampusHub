package model

import (
	"context"

	"gorm.io/gorm"
)

// TagModel 活动标签查询兼容层
// 说明：对外保持 TagModel.FindByActivityID 的旧接口
type TagModel struct {
	cache *TagCacheModel
}

func NewTagModel(db *gorm.DB) *TagModel {
	return &TagModel{
		cache: NewTagCacheModel(db),
	}
}

// FindByActivityID 获取活动关联的标签信息
func (m *TagModel) FindByActivityID(ctx context.Context, activityID uint64) ([]TagCache, error) {
	if m == nil || m.cache == nil {
		return []TagCache{}, nil
	}
	return m.cache.FindByActivityID(ctx, activityID)
}
