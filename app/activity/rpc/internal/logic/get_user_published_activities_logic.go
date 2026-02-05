package logic

import (
	"context"
	"errors"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserPublishedActivitiesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserPublishedActivitiesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserPublishedActivitiesLogic {
	return &GetUserPublishedActivitiesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取用户已发布的活动列表（User 服务调用，用于展示用户主页）
//
// 业务逻辑：
//  1. 参数校验（用户 ID 必须大于 0）
//  2. 根据 status 参数决定查询范围
//     - status = -1：公开状态（已发布/进行中/已结束）用于用户主页展示
//     - status = -2：全部状态（用户自己查看）
//     - status >= 0：指定状态
//  3. 调用 Model 层分页查询
//  4. 批量查询关联数据（分类名称、标签列表）
//  5. 构建响应
//
// 设计说明：
//   - 内部接口，供 User 服务调用
//   - 默认返回公开状态的活动（用于他人查看用户主页）
//   - 用户自己可查看全部状态的活动
func (l *GetUserPublishedActivitiesLogic) GetUserPublishedActivities(in *activity.GetUserPublishedActivitiesReq) (*activity.GetUserPublishedActivitiesResp, error) {
	// 1. 参数校验
	if in.GetUserId() <= 0 {
		return nil, errorx.ErrInvalidParams("用户ID无效")
	}

	// 2. 规范化分页参数
	page := int(in.GetPage())
	if page <= 0 {
		page = 1
	}
	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50 // 用户主页限制每页最多 50 条
	}

	// 3. 构建查询条件
	status := int(in.GetStatus())
	// 默认 status=0 时改为公开状态
	if status == 0 {
		status = -1 // 公开状态
	}

	query := &model.ListQuery{
		Pagination: model.Pagination{
			Page:     page,
			PageSize: pageSize,
		},
		OrganizerID: uint64(in.GetUserId()),
		Status:      status,
		Sort:        "created_at", // 按创建时间倒序
	}

	// 4. 调用 Model 层查询
	result, err := l.svcCtx.ActivityModel.List(l.ctx, query)
	if err != nil {
		if errors.Is(err, model.ErrPageTooDeep) {
			return nil, errorx.ErrInvalidParams("不支持查看超过100页的数据")
		}
		l.Errorf("查询用户发布的活动失败: user_id=%d, err=%v", in.GetUserId(), err)
		return nil, errorx.ErrDBError(err)
	}

	// 5. 空列表直接返回
	if len(result.List) == 0 {
		return &activity.GetUserPublishedActivitiesResp{
			List: []*activity.ActivityListItem{},
			Pagination: &activity.Pagination{
				Page:       int32(result.Page),
				PageSize:   int32(result.PageSize),
				Total:      result.Total,
				TotalPages: int32(result.TotalPages),
			},
		}, nil
	}

	// 6. 批量查询关联数据
	// 6.1 收集活动 ID
	activityIDs := make([]uint64, len(result.List))
	for i, act := range result.List {
		activityIDs[i] = act.ID
	}

	// 6.2 批量查询标签（从 tag_cache 表）
	tagsMap, err := l.svcCtx.TagCacheModel.FindByActivityIDs(l.ctx, activityIDs)
	if err != nil {
		l.Infof("[WARNING] 批量查询标签失败: %v", err)
		tagsMap = make(map[uint64][]model.TagCache)
	}

	// 6.3 加载分类映射表
	categoryMap := l.loadCategoryMap()

	// 7. 构建响应列表
	list := make([]*activity.ActivityListItem, len(result.List))
	for i, act := range result.List {
		list[i] = l.buildActivityListItem(&act, categoryMap, tagsMap)
	}

	l.Infof("获取用户发布的活动成功: user_id=%d, page=%d, total=%d, returned=%d",
		in.GetUserId(), result.Page, result.Total, len(list))

	return &activity.GetUserPublishedActivitiesResp{
		List: list,
		Pagination: &activity.Pagination{
			Page:       int32(result.Page),
			PageSize:   int32(result.PageSize),
			Total:      result.Total,
			TotalPages: int32(result.TotalPages),
		},
	}, nil
}

// loadCategoryMap 加载分类映射表
func (l *GetUserPublishedActivitiesLogic) loadCategoryMap() map[uint64]string {
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
func (l *GetUserPublishedActivitiesLogic) buildActivityListItem(
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
func (l *GetUserPublishedActivitiesLogic) convertTagCaches(tags []model.TagCache) []*activity.Tag {
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
