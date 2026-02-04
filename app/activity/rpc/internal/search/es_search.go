package search

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/zeromicro/go-zero/core/logx"
)

// ==================== 错误定义 ====================

var (
	ErrESNotEnabled = errors.New("ES 搜索未启用")
	ErrSearchFailed = errors.New("搜索失败")
)

// ==================== 搜索实现 ====================

// Search 搜索活动
//
// 搜索流程：
// 1. 构建查询条件（dis_max 优化）
// 2. 添加过滤条件（状态、分类、时间）
// 3. 添加排序和高亮
// 4. 执行搜索
// 5. 解析结果
//
// 性能优化：
// - 使用 dis_max 替代 multi_match，性能提升 30-50%
// - 使用 filter 替代 must（不计算评分，可被缓存）
// - 添加 id 作为 tie-breaker，保证分页一致性
func (c *ESClient) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	startTime := time.Now()

	// 1. 参数规范化
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 50 {
		req.PageSize = 50
	}

	// 2. 构建查询
	query := c.buildQuery(req)

	// 3. 构建搜索请求
	searchService := c.client.Search().
		Index(c.indexName).
		Query(query).
		From((req.Page - 1) * req.PageSize).
		Size(req.PageSize).
		TrackTotalHits(true) // 精确统计总数

	// 4. 添加高亮
	searchService = searchService.Highlight(
		elastic.NewHighlight().
			PreTags("<em>").
			PostTags("</em>").
			Fields(
				elastic.NewHighlighterField("title"),
				elastic.NewHighlighterField("description").NumOfFragments(2),
				elastic.NewHighlighterField("location"),
			),
	)

	// 5. 添加排序
	searchService = c.addSort(searchService, req.SortBy, req.Query)

	// 6. 添加聚合（分类统计，可选）
	searchService = searchService.Aggregation("category_count",
		elastic.NewTermsAggregation().Field("category_id").Size(10),
	)

	// 7. 执行搜索
	result, err := searchService.Do(ctx)
	if err != nil {
		logx.Errorf("[ESSearch] 搜索失败: query=%s, err=%v", req.Query, err)
		return nil, err
	}

	// 8. 解析结果
	response := &SearchResponse{
		Total:      result.TotalHits(),
		Activities: make([]ActivityDoc, 0, len(result.Hits.Hits)),
		TookMs:     time.Since(startTime).Milliseconds(),
	}

	for _, hit := range result.Hits.Hits {
		var doc ActivityDoc
		if err := json.Unmarshal(hit.Source, &doc); err != nil {
			logx.Errorf("[ESSearch] 解析文档失败: id=%s, err=%v", hit.Id, err)
			continue
		}

		// 处理高亮（替换原文）
		if hit.Highlight != nil {
			if titles, ok := hit.Highlight["title"]; ok && len(titles) > 0 {
				doc.Title = titles[0]
			}
			if descs, ok := hit.Highlight["description"]; ok && len(descs) > 0 {
				doc.Description = descs[0]
			}
			if locations, ok := hit.Highlight["location"]; ok && len(locations) > 0 {
				doc.Location = locations[0]
			}
		}

		response.Activities = append(response.Activities, doc)
	}

	// 9. 解析聚合
	if agg, found := result.Aggregations.Terms("category_count"); found {
		categoryCount := make(map[string]interface{})
		for _, bucket := range agg.Buckets {
			categoryCount[json.Number(bucket.KeyNumber.String()).String()] = bucket.DocCount
		}
		response.Aggregations = map[string]interface{}{
			"category_count": categoryCount,
		}
	}

	logx.Infof("[ESSearch] 搜索成功: query=%s, total=%d, returned=%d, took_ms=%d",
		req.Query, response.Total, len(response.Activities), response.TookMs)

	return response, nil
}

// buildQuery 构建查询条件
//
// 查询策略：
// - 关键词搜索：使用 dis_max 选择最佳匹配字段
// - 状态筛选：使用 filter（不计算评分，可被 ES 缓存）
// - 分类筛选：使用 filter
// - 时间范围：使用 filter
func (c *ESClient) buildQuery(req SearchRequest) elastic.Query {
	boolQuery := elastic.NewBoolQuery()

	// 1. 关键词搜索（使用 dis_max 优化）
	if req.Query != "" {
		// dis_max: 选择最佳匹配字段的分数，而非累加所有字段
		// 相比 multi_match + Fuzziness，性能提升显著
		disMaxQuery := elastic.NewDisMaxQuery().
			// 高优先级：标题短语匹配（完全匹配加分最多）
			Query(elastic.NewMatchPhraseQuery("title", req.Query).Boost(5)).
			// 中优先级：标题分词匹配
			Query(elastic.NewMatchQuery("title", req.Query).Boost(3)).
			// 中优先级：地点匹配
			Query(elastic.NewMatchQuery("location", req.Query).Boost(2)).
			// 低优先级：描述匹配
			Query(elastic.NewMatchQuery("description", req.Query).Boost(1)).
			TieBreaker(0.3) // 次优匹配也贡献 30% 分数

		boolQuery.Must(disMaxQuery)
	}

	// 2. 状态筛选（使用 filter，不计算评分，可被 ES 缓存）
	if len(req.Status) > 0 {
		statusValues := make([]interface{}, len(req.Status))
		for i, s := range req.Status {
			statusValues[i] = s
		}
		boolQuery.Filter(elastic.NewTermsQuery("status", statusValues...))
	} else {
		// 默认只搜索公开状态（已发布、进行中、已结束）
		// 注意：ES 的 TermsQuery 需要 interface{} 类型，int8 会自动转换
		boolQuery.Filter(elastic.NewTermsQuery("status",
			StatusPublished, StatusOngoing, StatusFinished))
	}

	// 3. 分类筛选
	if req.CategoryID > 0 {
		boolQuery.Filter(elastic.NewTermQuery("category_id", req.CategoryID))
	}

	// 4. 时间范围筛选
	// ⚠️ ES mapping 使用 epoch_second，直接传 int64 时间戳
	if req.StartTime != nil || req.EndTime != nil {
		rangeQuery := elastic.NewRangeQuery("activity_start_time")
		if req.StartTime != nil {
			rangeQuery.Gte(*req.StartTime)
		}
		if req.EndTime != nil {
			rangeQuery.Lte(*req.EndTime)
		}
		boolQuery.Filter(rangeQuery)
	}

	return boolQuery
}

// addSort 添加排序
//
// 排序策略：
// - relevance：按相关性分数（有关键词时）+ id 作为 tie-breaker
// - time：按活动开始时间
// - hot：按报名人数
// - newest：按创建时间
func (c *ESClient) addSort(s *elastic.SearchService, sortBy string, query string) *elastic.SearchService {
	switch sortBy {
	case "time":
		// 按活动开始时间排序（即将开始的优先）
		return s.Sort("activity_start_time", true).Sort("id", false)
	case "hot":
		// 按报名人数排序（热度）
		return s.Sort("current_participants", false).Sort("id", false)
	case "newest":
		// 按创建时间排序
		return s.Sort("created_at", false).Sort("id", false)
	case "relevance":
		fallthrough
	default:
		// 按相关性排序
		if query != "" {
			// 有关键词：按评分排序 + id 作为 tie-breaker（保证分页一致性）
			return s.Sort("_score", false).Sort("id", false)
		}
		// 无关键词：按热度排序
		return s.Sort("current_participants", false).Sort("id", false)
	}
}

// ==================== 高级搜索功能 ====================

// SearchByIDs 根据 ID 批量获取文档
func (c *ESClient) SearchByIDs(ctx context.Context, ids []uint64) ([]ActivityDoc, error) {
	if len(ids) == 0 {
		return []ActivityDoc{}, nil
	}

	// 构建 IDs 查询
	idsValues := make([]string, len(ids))
	for i, id := range ids {
		idsValues[i] = DocID(id)
	}

	result, err := c.client.Search().
		Index(c.indexName).
		Query(elastic.NewIdsQuery().Ids(idsValues...)).
		Size(len(ids)).
		Do(ctx)

	if err != nil {
		return nil, err
	}

	docs := make([]ActivityDoc, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		var doc ActivityDoc
		if err := json.Unmarshal(hit.Source, &doc); err != nil {
			continue
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

// CountByCategory 按分类统计活动数量
func (c *ESClient) CountByCategory(ctx context.Context) (map[uint64]int64, error) {
	result, err := c.client.Search().
		Index(c.indexName).
		Query(elastic.NewTermsQuery("status",
			int(StatusPublished), int(StatusOngoing))).
		Size(0). // 只要聚合结果
		Aggregation("category_count",
			elastic.NewTermsAggregation().Field("category_id").Size(100)).
		Do(ctx)

	if err != nil {
		return nil, err
	}

	counts := make(map[uint64]int64)
	if agg, found := result.Aggregations.Terms("category_count"); found {
		for _, bucket := range agg.Buckets {
			if categoryID, ok := bucket.Key.(float64); ok {
				counts[uint64(categoryID)] = bucket.DocCount
			}
		}
	}

	return counts, nil
}
