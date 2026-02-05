package logic

import (
	"context"
	"errors"
	"time"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/search"
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

// SearchActivities 搜索活动
//
// 搜索策略（优雅降级）：
//  1. 优先使用 ES 搜索（如果已启用且可用）
//  2. ES 不可用时，自动降级到 MySQL LIKE 搜索
//
// 搜索规则：
//   - 关键词匹配：标题、描述、地点（OR 关系）
//   - 只搜索公开状态：已发布(2)/进行中(3)/已结束(4)
//   - 支持分类筛选
//   - 支持多种排序：相关性/时间/热度
func (l *SearchActivitiesLogic) SearchActivities(in *activity.SearchActivitiesReq) (*activity.SearchActivitiesResp, error) {
	startTime := time.Now()

	// 1. 参数校验
	if err := l.validateParams(in); err != nil {
		return nil, err
	}

	// 2. 尝试使用 ES 搜索
	if l.svcCtx.ESClient != nil && l.svcCtx.ESClient.IsEnabled() {
		resp, err := l.searchWithES(in)
		if err == nil {
			return resp, nil
		}
		// ES 搜索失败，记录日志并降级到 MySQL
		l.Errorf("[SearchActivities] ES 搜索失败，降级到 MySQL: %v", err)
	}

	// 3. 降级到 MySQL LIKE 搜索
	return l.searchWithMySQL(in, startTime)
}

// ==================== ES 搜索 ====================

// searchWithES 使用 ES 搜索
func (l *SearchActivitiesLogic) searchWithES(in *activity.SearchActivitiesReq) (*activity.SearchActivitiesResp, error) {
	startTime := time.Now()

	// 1. 构建 ES 搜索请求
	req := search.SearchRequest{
		Query:      in.Keyword,
		CategoryID: uint64(in.CategoryId),
		SortBy:     in.Sort,
		Page:       int(in.Page),
		PageSize:   int(in.PageSize),
	}

	// 2. 执行搜索
	result, err := l.svcCtx.ESClient.SearchWithFallback(l.ctx, req)
	if err != nil {
		return nil, err
	}

	// 3. 转换响应
	list := make([]*activity.ActivityListItem, len(result.Activities))
	for i, doc := range result.Activities {
		list[i] = l.convertESDocToListItem(&doc)
	}

	queryTimeMs := time.Since(startTime).Milliseconds()

	l.Infof("[SearchActivities] ES 搜索成功: keyword=%s, total=%d, returned=%d, took_ms=%d",
		in.Keyword, result.Total, len(list), queryTimeMs)

	return &activity.SearchActivitiesResp{
		List:        list,
		Total:       result.Total,
		QueryTimeMs: int32(queryTimeMs),
	}, nil
}

// convertESDocToListItem 转换 ES 文档为列表项
func (l *SearchActivitiesLogic) convertESDocToListItem(doc *search.ActivityDoc) *activity.ActivityListItem {
	return &activity.ActivityListItem{
		Id:                  int64(doc.ID),
		Title:               doc.Title, // 可能包含高亮标签 <em>
		CoverUrl:            doc.CoverURL,
		CoverType:           int32(doc.CoverType),
		CategoryName:        doc.CategoryName,
		OrganizerName:       doc.OrganizerName,
		OrganizerAvatar:     doc.OrganizerAvatar,
		ActivityStartTime:   doc.ActivityStartTime,
		Location:            doc.Location, // 可能包含高亮标签 <em>
		MaxParticipants:     int32(doc.MaxParticipants),
		CurrentParticipants: int32(doc.CurrentParticipants),
		Status:              int32(doc.Status),
		StatusText:          l.getStatusText(doc.Status),
		Tags:                l.convertTagStrings(doc.Tags),
		ViewCount:           int64(doc.ViewCount),
		CreatedAt:           doc.CreatedAt,
	}
}

// convertTagStrings 转换标签字符串为 Proto Tag
func (l *SearchActivitiesLogic) convertTagStrings(tags []string) []*activity.Tag {
	if len(tags) == 0 {
		return []*activity.Tag{}
	}

	result := make([]*activity.Tag, len(tags))
	for i, name := range tags {
		result[i] = &activity.Tag{
			Id:   0, // ES 中只存了名称，ID 需要另外查询
			Name: name,
		}
	}
	return result
}

// getStatusText 获取状态文本
func (l *SearchActivitiesLogic) getStatusText(status int8) string {
	switch status {
	case model.StatusDraft:
		return "草稿"
	case model.StatusPending:
		return "待审核"
	case model.StatusPublished:
		return "已发布"
	case model.StatusOngoing:
		return "进行中"
	case model.StatusFinished:
		return "已结束"
	case model.StatusRejected:
		return "已拒绝"
	case model.StatusCancelled:
		return "已取消"
	default:
		return "未知"
	}
}

// ==================== MySQL 搜索（降级方案）====================

// searchWithMySQL 使用 MySQL LIKE 搜索（降级方案）
func (l *SearchActivitiesLogic) searchWithMySQL(in *activity.SearchActivitiesReq, startTime time.Time) (*activity.SearchActivitiesResp, error) {
	// 1. 构建搜索条件
	query := &model.SearchQuery{
		Keyword:    in.Keyword,
		CategoryID: uint64(in.CategoryId),
		Page:       int(in.Page),
		PageSize:   int(in.PageSize),
		Sort:       in.Sort,
	}

	// 2. 执行搜索
	result, err := l.svcCtx.ActivityModel.Search(l.ctx, query)
	if err != nil {
		if errors.Is(err, model.ErrPageTooDeep) {
			return nil, errorx.ErrInvalidParams("搜索结果不支持查看超过100页，请缩小搜索范围")
		}
		l.Errorf("MySQL 搜索失败: keyword=%s, err=%v", in.Keyword, err)
		return nil, errorx.ErrDBError(err)
	}

	// 3. 空结果直接返回
	if len(result.List) == 0 {
		queryTimeMs := time.Since(startTime).Milliseconds()
		l.Infof("[SearchActivities] MySQL 搜索完成（无结果）: keyword=%s, took_ms=%d",
			in.Keyword, queryTimeMs)

		return &activity.SearchActivitiesResp{
			List:        []*activity.ActivityListItem{},
			Total:       0,
			QueryTimeMs: int32(queryTimeMs),
		}, nil
	}

	// 4. 批量查询关联数据
	activityIDs := make([]uint64, len(result.List))
	for i, act := range result.List {
		activityIDs[i] = act.ID
	}

	// 4.1 批量查询标签
	tagsMap, err := l.svcCtx.TagCacheModel.FindByActivityIDs(l.ctx, activityIDs)
	if err != nil {
		l.Infof("[WARNING] 批量查询标签失败: %v", err)
		tagsMap = make(map[uint64][]model.TagCache)
	}

	// 4.2 加载分类映射
	categoryMap := l.loadCategoryMap()

	// 5. 构建响应列表
	list := make([]*activity.ActivityListItem, len(result.List))
	for i, act := range result.List {
		list[i] = l.buildActivityListItem(&act, categoryMap, tagsMap)
	}

	// 6. 计算查询耗时
	queryTimeMs := time.Since(startTime).Milliseconds()

	l.Infof("[SearchActivities] MySQL 搜索成功: keyword=%s, page=%d, pageSize=%d, total=%d, returned=%d, took_ms=%d",
		in.Keyword, result.Page, result.PageSize, result.Total, len(list), queryTimeMs)

	return &activity.SearchActivitiesResp{
		List:        list,
		Total:       result.Total,
		QueryTimeMs: int32(queryTimeMs),
	}, nil
}

// ==================== 公共方法 ====================

// validateParams 参数校验
func (l *SearchActivitiesLogic) validateParams(in *activity.SearchActivitiesReq) error {
	// 1. 关键词校验
	keywordLen := len([]rune(in.Keyword))
	if keywordLen < 2 {
		return errorx.ErrInvalidParams("搜索关键词至少2个字符")
	}
	if keywordLen > 50 {
		return errorx.ErrInvalidParams("搜索关键词不能超过50个字符")
	}

	// 2. 分页参数规范化
	if in.Page <= 0 {
		in.Page = 1
	}
	if in.PageSize <= 0 {
		in.PageSize = 10
	}
	if in.PageSize > 50 {
		in.PageSize = 50
	}

	// 3. 排序字段校验
	validSorts := map[string]bool{
		"":          true,
		"relevance": true,
		"time":      true,
		"hot":       true,
	}
	if !validSorts[in.Sort] {
		return errorx.ErrInvalidParams("无效的排序方式，可选值：relevance/time/hot")
	}

	return nil
}

// loadCategoryMap 加载分类映射表
func (l *SearchActivitiesLogic) loadCategoryMap() map[uint64]string {
	// 优先使用缓存
	if l.svcCtx.CategoryCache != nil {
		nameMap, err := l.svcCtx.CategoryCache.GetNameMap(l.ctx)
		if err == nil && len(nameMap) > 0 {
			return nameMap
		}
	}

	// 降级到数据库查询
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
