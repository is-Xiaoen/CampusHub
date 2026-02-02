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

type ListActivitiesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListActivitiesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListActivitiesLogic {
	return &ListActivitiesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ListActivities 活动列表查询
//
// 业务逻辑：
//  1. 参数校验和规范化
//  2. 权限校验（status=-2 需要 organizer_id 或管理员）
//  3. 调用 Model 层分页查询
//  4. 批量查询关联数据（分类名称、标签列表）
//  5. 构建列表响应
//
// 状态筛选规则：
//   - status=-1：公开状态（已发布/进行中/已结束）
//   - status=-2：全部状态（需要 organizer_id 或 is_admin）
//   - status=0-6：具体状态值
func (l *ListActivitiesLogic) ListActivities(in *activity.ListActivitiesReq) (*activity.ListActivitiesResp, error) {
	// 1. 参数校验
	if err := l.validateParams(in); err != nil {
		return nil, err
	}

	// 2. 构建查询条件
	query := &model.ListQuery{
		Pagination: model.Pagination{
			Page:     int(in.Page),
			PageSize: int(in.PageSize),
		},
		CategoryID:  uint64(in.CategoryId),
		Status:      int(in.Status),
		OrganizerID: uint64(in.OrganizerId),
		Sort:        in.Sort,
	}

	// 3. 调用 Model 层查询
	result, err := l.svcCtx.ActivityModel.List(l.ctx, query)
	if err != nil {
		if errors.Is(err, model.ErrPageTooDeep) {
			return nil, errorx.ErrInvalidParams("不支持查看超过100页的数据，请使用搜索功能")
		}
		l.Errorf("查询活动列表失败: %v", err)
		return nil, errorx.ErrDBError(err)
	}

	// 4. 空列表直接返回
	if len(result.List) == 0 {
		return &activity.ListActivitiesResp{
			List: []*activity.ActivityListItem{},
			Pagination: &activity.Pagination{
				Page:       int32(result.Page),
				PageSize:   int32(result.PageSize),
				Total:      result.Total,
				TotalPages: int32(result.TotalPages),
			},
		}, nil
	}

	// 5. 批量查询关联数据
	// 5.1 收集活动 ID 和分类 ID
	activityIDs := make([]uint64, len(result.List))
	categoryIDSet := make(map[uint64]bool)
	for i, act := range result.List {
		activityIDs[i] = act.ID
		categoryIDSet[act.CategoryID] = true
	}

	// 5.2 批量查询标签
	tagsMap, err := l.svcCtx.TagModel.FindByActivityIDs(l.ctx, activityIDs)
	if err != nil {
		l.Infof("[WARNING] 批量查询标签失败: %v", err)
		tagsMap = make(map[uint64][]model.Tag)
	}

	// 5.3 查询分类（分类数量通常较少，一次性查询所有）
	categoryMap := l.loadCategoryMap()

	// 6. 构建响应列表
	list := make([]*activity.ActivityListItem, len(result.List))
	for i, act := range result.List {
		list[i] = l.buildActivityListItem(&act, categoryMap, tagsMap)
	}

	l.Infof("查询活动列表成功: page=%d, pageSize=%d, total=%d, returned=%d",
		result.Page, result.PageSize, result.Total, len(list))

	return &activity.ListActivitiesResp{
		List: list,
		Pagination: &activity.Pagination{
			Page:       int32(result.Page),
			PageSize:   int32(result.PageSize),
			Total:      result.Total,
			TotalPages: int32(result.TotalPages),
		},
	}, nil
}

// validateParams 参数校验
func (l *ListActivitiesLogic) validateParams(in *activity.ListActivitiesReq) error {
	// 1. 分页参数校验（Model 层会规范化，这里只做基本校验）
	if in.Page < 0 {
		return errorx.ErrInvalidParams("页码不能为负数")
	}
	if in.PageSize < 0 {
		return errorx.ErrInvalidParams("每页数量不能为负数")
	}

	// 2. 状态校验
	// status=-2（全部状态）需要指定 organizer_id 或 is_admin
	// 防止普通用户查看其他人的草稿/待审核活动
	if in.Status == -2 {
		if in.OrganizerId <= 0 && !in.IsAdmin {
			return errorx.ErrInvalidParams("查看全部状态需要指定组织者或管理员权限")
		}
	}

	// 3. 排序字段校验
	validSorts := map[string]bool{
		"":           true, // 默认
		"created_at": true,
		"hot":        true,
		"start_time": true,
	}
	if !validSorts[in.Sort] {
		return errorx.ErrInvalidParams("无效的排序字段")
	}

	return nil
}

// loadCategoryMap 加载分类映射表
//
// 由于分类数量通常较少（一般不超过 20 个），
// 直接加载所有分类比按需查询更高效
func (l *ListActivitiesLogic) loadCategoryMap() map[uint64]string {
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
func (l *ListActivitiesLogic) buildActivityListItem(
	act *model.Activity,
	categoryMap map[uint64]string,
	tagsMap map[uint64][]model.Tag,
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
		Tags:                convertTagsForList(tags),
		ViewCount:           int64(act.ViewCount),
		CreatedAt:           act.CreatedAt,
	}
}

// convertTagsForList 转换标签列表（用于列表项）
func convertTagsForList(tags []model.Tag) []*activity.Tag {
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
