package logic

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/cron"
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

	// 推荐模式：从 Redis 预计算缓存读取
	if in.Recommend {
		return l.listByRecommend(in)
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

	// 5.2 批量查询标签（从 tag_cache 表）
	tagsMap, err := l.svcCtx.TagCacheModel.FindByActivityIDs(l.ctx, activityIDs)
	if err != nil {
		l.Infof("[WARNING] 批量查询标签失败: %v", err)
		tagsMap = make(map[uint64][]model.TagCache)
	}

	// 5.3 查询分类（分类数量通常较少，一次性查询所有）
	categoryMap := l.loadCategoryMap()

	// 5.4 获取组织者最新信息（头像/昵称可能已更新）
	organizerIDs := make([]uint64, len(result.List))
	for i, act := range result.List {
		organizerIDs[i] = act.OrganizerID
	}
	organizerMap := fetchOrganizerMap(l.ctx, l.svcCtx, organizerIDs)
	for i := range result.List {
		if info, ok := organizerMap[result.List[i].OrganizerID]; ok {
			result.List[i].OrganizerName = info.Name
			result.List[i].OrganizerAvatar = info.Avatar
		}
	}

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

	// 3. 排序字段校验（推荐模式下排序由缓存决定，跳过校验）
	if !in.Recommend {
		validSorts := map[string]bool{
			"":           true, // 默认
			"created_at": true,
			"hot":        true,
			"start_time": true,
		}
		if !validSorts[in.Sort] {
			return errorx.ErrInvalidParams("无效的排序字段")
		}
	}

	return nil
}

// loadCategoryMap 加载分类映射表（优先从缓存获取）
//
// 由于分类数量通常较少（一般不超过 20 个），
// 直接加载所有分类比按需查询更高效
func (l *ListActivitiesLogic) loadCategoryMap() map[uint64]string {
	// 优先使用缓存
	if l.svcCtx.CategoryCache != nil {
		categoryMap, err := l.svcCtx.CategoryCache.GetNameMap(l.ctx)
		if err == nil {
			return categoryMap
		}
		l.Infof("[WARNING] 从缓存加载分类失败，降级查 DB: %v", err)
	}

	// 降级查 DB
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
	tagsMap map[uint64][]model.TagCache,
) *activity.ActivityListItem {
	// 获取分类名称
	categoryName := categoryMap[act.CategoryID]
	if categoryName == "" {
		categoryName = "未知分类"
	}

	// 获取标签列表
	tags := tagsMap[act.ID]

	// 计算报名状态
	now := time.Now().Unix()
	regStatus, regStatusText := model.ComputeRegistrationStatus(
		act.Status, act.RegisterStartTime, act.RegisterEndTime, now,
	)

	return &activity.ActivityListItem{
		Id:                     int64(act.ID),
		Title:                  act.Title,
		CoverUrl:               act.CoverURL,
		CoverType:              int32(act.CoverType),
		CategoryName:           categoryName,
		OrganizerName:          act.OrganizerName,
		OrganizerAvatar:        act.OrganizerAvatar,
		ActivityStartTime:      act.ActivityStartTime,
		Location:               act.Location,
		MaxParticipants:        int32(act.MaxParticipants),
		CurrentParticipants:    int32(act.CurrentParticipants),
		Status:                 int32(act.Status),
		StatusText:             act.StatusText(),
		Tags:                   convertTagCachesForList(tags),
		ViewCount:              int64(act.ViewCount),
		CreatedAt:              act.CreatedAt,
		RegistrationStatus:     regStatus,
		RegistrationStatusText: regStatusText,
	}
}

// recommendCacheKey Redis 推荐列表缓存键（与 RecommendCron 保持一致）
const recommendCacheKey = "activity:recommend:list_cache:global"

// listByRecommend 推荐模式：从 Redis 预计算缓存读取排序后的活动列表
//
// 数据流：Redis 缓存 → 分页切片 → 按 ID 批量查 DB → 补充关联数据 → 返回
// 降级策略：Redis 不可用或缓存缺失时，自动降级为 sort=hot 的普通 DB 查询
func (l *ListActivitiesLogic) listByRecommend(in *activity.ListActivitiesReq) (*activity.ListActivitiesResp, error) {
	// 1. 从 Redis 读取推荐缓存
	cached, err := l.svcCtx.Redis.GetCtx(l.ctx, recommendCacheKey)
	if err != nil || cached == "" {
		l.Infof("[listByRecommend] 推荐缓存未命中（err=%v），降级为热度排序", err)
		return l.fallbackToHotSort(in)
	}

	// 2. 反序列化
	var scoredList []cron.ActivityScoreDTO
	if err := json.Unmarshal([]byte(cached), &scoredList); err != nil {
		l.Errorf("[listByRecommend] 反序列化推荐缓存失败: %v", err)
		return l.fallbackToHotSort(in)
	}

	if len(scoredList) == 0 {
		return &activity.ListActivitiesResp{
			List: []*activity.ActivityListItem{},
			Pagination: &activity.Pagination{
				Page:       in.Page,
				PageSize:   in.PageSize,
				Total:      0,
				TotalPages: 0,
			},
		}, nil
	}

	// 3. 分页计算
	total := int64(len(scoredList))
	page := int(in.Page)
	pageSize := int(in.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	startIdx := (page - 1) * pageSize
	if startIdx >= int(total) {
		// 超出范围，返回空
		totalPages := int32(total) / in.PageSize
		if int64(totalPages)*int64(in.PageSize) < total {
			totalPages++
		}
		return &activity.ListActivitiesResp{
			List: []*activity.ActivityListItem{},
			Pagination: &activity.Pagination{
				Page:       in.Page,
				PageSize:   in.PageSize,
				Total:      total,
				TotalPages: totalPages,
			},
		}, nil
	}

	endIdx := startIdx + pageSize
	if endIdx > int(total) {
		endIdx = int(total)
	}

	// 4. 提取当前页的活动 ID（保持推荐分数顺序）
	pageSlice := scoredList[startIdx:endIdx]
	ids := make([]uint64, len(pageSlice))
	for i, dto := range pageSlice {
		ids[i] = dto.ActivityID
	}

	// 5. 批量查询完整活动数据
	activities, err := l.svcCtx.ActivityModel.FindByIDs(l.ctx, ids)
	if err != nil {
		l.Errorf("[listByRecommend] 批量查询活动失败: %v", err)
		return nil, errorx.ErrDBError(err)
	}

	// 6. 按推荐顺序重排（FindByIDs 不保证顺序）
	actMap := make(map[uint64]*model.Activity, len(activities))
	for i := range activities {
		actMap[activities[i].ID] = &activities[i]
	}
	ordered := make([]model.Activity, 0, len(ids))
	for _, id := range ids {
		if act, ok := actMap[id]; ok {
			ordered = append(ordered, *act)
		}
	}

	if len(ordered) == 0 {
		totalPages := int32(total) / in.PageSize
		if int64(totalPages)*int64(in.PageSize) < total {
			totalPages++
		}
		return &activity.ListActivitiesResp{
			List: []*activity.ActivityListItem{},
			Pagination: &activity.Pagination{
				Page:       in.Page,
				PageSize:   in.PageSize,
				Total:      total,
				TotalPages: totalPages,
			},
		}, nil
	}

	// 7. 批量查询关联数据（标签、分类、组织者信息）
	orderedIDs := make([]uint64, len(ordered))
	organizerIDs := make([]uint64, len(ordered))
	for i, act := range ordered {
		orderedIDs[i] = act.ID
		organizerIDs[i] = act.OrganizerID
	}

	tagsMap, err := l.svcCtx.TagCacheModel.FindByActivityIDs(l.ctx, orderedIDs)
	if err != nil {
		l.Infof("[listByRecommend] 批量查询标签失败: %v", err)
		tagsMap = make(map[uint64][]model.TagCache)
	}

	categoryMap := l.loadCategoryMap()

	organizerMap := fetchOrganizerMap(l.ctx, l.svcCtx, organizerIDs)
	for i := range ordered {
		if info, ok := organizerMap[ordered[i].OrganizerID]; ok {
			ordered[i].OrganizerName = info.Name
			ordered[i].OrganizerAvatar = info.Avatar
		}
	}

	// 8. 构建响应
	list := make([]*activity.ActivityListItem, len(ordered))
	for i, act := range ordered {
		list[i] = l.buildActivityListItem(&act, categoryMap, tagsMap)
	}

	totalPages := int32(total) / in.PageSize
	if int64(totalPages)*int64(in.PageSize) < total {
		totalPages++
	}

	l.Infof("[listByRecommend] 推荐列表查询成功: page=%d, pageSize=%d, total=%d, returned=%d",
		page, pageSize, total, len(list))

	return &activity.ListActivitiesResp{
		List: list,
		Pagination: &activity.Pagination{
			Page:       in.Page,
			PageSize:   in.PageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// fallbackToHotSort 降级：推荐缓存不可用时，走 sort=hot 的普通 DB 查询
func (l *ListActivitiesLogic) fallbackToHotSort(in *activity.ListActivitiesReq) (*activity.ListActivitiesResp, error) {
	return l.ListActivities(&activity.ListActivitiesReq{
		Page:        in.Page,
		PageSize:    in.PageSize,
		CategoryId:  in.CategoryId,
		Status:      in.Status,
		OrganizerId: in.OrganizerId,
		Sort:        "hot",
		ViewerId:    in.ViewerId,
		IsAdmin:     in.IsAdmin,
		Recommend:   false,
	})
}

// convertTagCachesForList 转换标签缓存列表为 Proto Tag（用于列表项）
func convertTagCachesForList(tags []model.TagCache) []*activity.Tag {
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
