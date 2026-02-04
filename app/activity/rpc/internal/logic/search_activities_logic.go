package logic

import (
	"context"
	"errors"
	"time"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchActivitiesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSearchActivitiesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchActivitiesLogic {
	return &SearchActivitiesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// SearchActivities 搜索活动（MySQL LIKE 版本）
//
// 业务逻辑：
//  1. 参数校验和规范化
//  2. 调用 Model 层搜索（MySQL LIKE）
//  3. 批量查询关联数据（分类名称、标签列表）
//  4. 构建响应（包含查询耗时）
//
// 搜索规则：
//   - 关键词匹配：标题、描述、地点（OR 关系）
//   - 只搜索公开状态：已发布(2)/进行中(3)/已结束(4)
//   - 支持分类筛选
//   - 支持多种排序：相关性/时间/热度
//
// 性能说明：
//   - 当前使用 MySQL LIKE '%keyword%'，数据量 < 5万条时可接受
//   - 后续可替换为 Elasticsearch 实现
func (l *SearchActivitiesLogic) SearchActivities(in *activity.SearchActivitiesReq) (*activity.SearchActivitiesResp, error) {
	startTime := time.Now()

	// 1. 参数校验
	if err := l.validateParams(in); err != nil {
		return nil, err
	}

	// 2. 构建搜索条件
	query := &model.SearchQuery{
		Keyword:    in.Keyword,
		CategoryID: uint64(in.CategoryId),
		Page:       int(in.Page),
		PageSize:   int(in.PageSize),
		Sort:       in.Sort,
	}

	// 3. 执行搜索
	result, err := l.svcCtx.ActivityModel.Search(l.ctx, query)
	if err != nil {
		if errors.Is(err, model.ErrPageTooDeep) {
			return nil, errorx.ErrInvalidParams("搜索结果不支持查看超过100页，请缩小搜索范围")
		}
		l.Errorf("搜索活动失败: keyword=%s, err=%v", in.Keyword, err)
		return nil, errorx.ErrDBError(err)
	}

	// 4. 空结果直接返回
	if len(result.List) == 0 {
		queryTimeMs := time.Since(startTime).Milliseconds()
		l.Infof("搜索活动完成（无结果）: keyword=%s, categoryId=%d, queryTimeMs=%d",
			in.Keyword, in.CategoryId, queryTimeMs)

		return &activity.SearchActivitiesResp{
			List:        []*activity.ActivityListItem{},
			Total:       0,
			QueryTimeMs: int32(queryTimeMs),
		}, nil
	}

	// 5. 批量查询关联数据
	// 5.1 收集活动 ID
	activityIDs := make([]uint64, len(result.List))
	for i, act := range result.List {
		activityIDs[i] = act.ID
	}

	// 5.2 批量查询标签
	tagsMap, err := l.svcCtx.TagCacheModel.FindByActivityIDs(l.ctx, activityIDs)
	if err != nil {
		l.Infof("[WARNING] 批量查询标签失败: %v", err)
		tagsMap = make(map[uint64][]model.TagCache)
	}

	// 5.3 加载分类映射
	categoryMap := l.loadCategoryMap()

	// 6. 构建响应列表
	list := make([]*activity.ActivityListItem, len(result.List))
	for i, act := range result.List {
		list[i] = l.buildActivityListItem(&act, categoryMap, tagsMap)
	}

	// 7. 计算查询耗时
	queryTimeMs := time.Since(startTime).Milliseconds()

	l.Infof("搜索活动成功: keyword=%s, categoryId=%d, page=%d, pageSize=%d, total=%d, returned=%d, queryTimeMs=%d",
		in.Keyword, in.CategoryId, result.Page, result.PageSize, result.Total, len(list), queryTimeMs)

	return &activity.SearchActivitiesResp{
		List:        list,
		Total:       result.Total,
		QueryTimeMs: int32(queryTimeMs),
	}, nil
}

// validateParams 参数校验
func (l *SearchActivitiesLogic) validateParams(in *activity.SearchActivitiesReq) error {
	// 1. 关键词校验（API 层已校验，这里做兜底）
	keywordLen := len([]rune(in.Keyword))
	if keywordLen < 2 {
		return errorx.ErrInvalidParams("搜索关键词至少2个字符")
	}
	if keywordLen > 50 {
		return errorx.ErrInvalidParams("搜索关键词不能超过50个字符")
	}

	// 2. 分页参数规范化（负数或 0 会在 Model 层处理）
	if in.Page < 0 {
		in.Page = 1
	}
	if in.PageSize < 0 {
		in.PageSize = 10
	}

	// 3. 排序字段校验
	validSorts := map[string]bool{
		"":          true, // 默认
		"relevance": true, // 相关性
		"time":      true, // 时间
		"hot":       true, // 热度
	}
	if !validSorts[in.Sort] {
		return errorx.ErrInvalidParams("无效的排序方式，可选值：relevance/time/hot")
	}

	return nil
}

// loadCategoryMap 加载分类映射表
func (l *SearchActivitiesLogic) loadCategoryMap() map[uint64]string {
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
func (l *SearchActivitiesLogic) buildActivityListItem(
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

// convertTagCaches 转换标签缓存列表为 Proto Tag
func (l *SearchActivitiesLogic) convertTagCaches(tags []model.TagCache) []*activity.Tag {
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
