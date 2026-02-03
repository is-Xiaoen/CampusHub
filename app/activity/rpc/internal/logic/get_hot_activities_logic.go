package logic

import (
	"context"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetHotActivitiesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetHotActivitiesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetHotActivitiesLogic {
	return &GetHotActivitiesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetHotActivities 获取热门活动列表
//
// 业务逻辑：
//  1. 参数校验和规范化（limit 范围 1-20，默认 10）
//  2. 调用 Model 层查询热门活动（按报名人数降序）
//  3. 批量查询关联数据（分类名称、标签列表）
//  4. 构建响应
//
// 设计说明：
//   - 热门活动定义：已发布或进行中，且活动未结束
//   - 排序规则：current_participants DESC, created_at DESC
//   - 最大返回 20 条，防止数据量过大
func (l *GetHotActivitiesLogic) GetHotActivities(in *activity.GetHotActivitiesReq) (*activity.GetHotActivitiesResp, error) {
	// 1. 参数规范化
	limit := int(in.GetLimit())
	if limit <= 0 {
		limit = 10 // 默认值
	}
	if limit > 20 {
		limit = 20 // 上限
	}

	// 2. 查询热门活动
	activities, err := l.svcCtx.ActivityModel.FindHot(l.ctx, limit)
	if err != nil {
		l.Errorf("查询热门活动失败: err=%v", err)
		return nil, errorx.ErrDBError(err)
	}

	// 3. 空列表直接返回
	if len(activities) == 0 {
		return &activity.GetHotActivitiesResp{
			List: []*activity.ActivityListItem{},
		}, nil
	}

	// 4. 批量查询关联数据
	// 4.1 收集活动 ID
	activityIDs := make([]uint64, len(activities))
	for i, act := range activities {
		activityIDs[i] = act.ID
	}

	// 4.2 批量查询标签（从 tag_cache 表）
	tagsMap, err := l.svcCtx.TagCacheModel.FindByActivityIDs(l.ctx, activityIDs)
	if err != nil {
		l.Infof("[WARNING] 批量查询标签失败: %v", err)
		tagsMap = make(map[uint64][]model.TagCache)
	}

	// 4.3 加载分类映射表
	categoryMap := l.loadCategoryMap()

	// 5. 构建响应列表
	list := make([]*activity.ActivityListItem, len(activities))
	for i, act := range activities {
		list[i] = l.buildActivityListItem(&act, categoryMap, tagsMap)
	}

	l.Infof("获取热门活动成功: limit=%d, returned=%d", limit, len(list))

	return &activity.GetHotActivitiesResp{
		List: list,
	}, nil
}

// loadCategoryMap 加载分类映射表
func (l *GetHotActivitiesLogic) loadCategoryMap() map[uint64]string {
	categoryMap := make(map[uint64]string)

	categories, err := l.svcCtx.CategoryModel.FindAll(l.ctx)
	if err != nil {
		l.Infof("[WARNING] 加载分类列表失败: %v", err)
		return categoryMap
	}

	for _, cat := range categories {
		categoryMap[cat.ID] = cat.Name
	}
	return categoryMap
}

// buildActivityListItem 构建活动列表项
func (l *GetHotActivitiesLogic) buildActivityListItem(
	act *model.Activity,
	categoryMap map[uint64]string,
	tagsMap map[uint64][]model.TagCache,
) *activity.ActivityListItem {
	// 获取分类名称
	categoryName := categoryMap[act.CategoryID]
	if categoryName == "" {
		categoryName = "未知分类"
	}

	// 获取标签列表
	tags := tagsMap[act.ID]

	return &activity.ActivityListItem{
		Id:                  int64(act.ID),
		Title:               act.Title,
		CoverUrl:            act.CoverURL,
		CoverType:           int32(act.CoverType),
		CategoryName:        categoryName,
		OrganizerName:       act.OrganizerName,
		OrganizerAvatar:     act.OrganizerAvatar,
		ActivityStartTime:   act.ActivityStartTime,
		Location:            act.Location,
		MaxParticipants:     int32(act.MaxParticipants),
		CurrentParticipants: int32(act.CurrentParticipants),
		Status:              int32(act.Status),
		StatusText:          act.StatusText(),
		Tags:                l.convertTagCaches(tags),
		ViewCount:           int64(act.ViewCount),
		CreatedAt:           act.CreatedAt,
	}
}

// convertTagCaches 将 model.TagCache 转换为 proto Tag
func (l *GetHotActivitiesLogic) convertTagCaches(tags []model.TagCache) []*activity.Tag {
	if len(tags) == 0 {
		return []*activity.Tag{}
	}

	result := make([]*activity.Tag, len(tags))
	for i, tag := range tags {
		result[i] = &activity.Tag{
			Id:    int64(tag.ID),
			Name:  tag.Name,
			Color: tag.Color,
			Icon:  tag.Icon,
		}
	}
	return result
}
