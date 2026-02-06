package search

import (
	"context"
	"fmt"
	"time"

	"activity-platform/app/activity/model"

	"github.com/olivere/elastic/v7"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// ==================== 数据同步服务 ====================

// SyncService 数据同步服务
//
// 职责：
// - 单个活动同步（创建/更新后）
// - 批量同步（全量同步）
// - 删除同步
//
// 同步策略：
// - 使用外部版本控制，防止旧数据覆盖新数据
// - 异步同步 + 重试机制
// - 失败记录（后续可扩展失败队列）
type SyncService struct {
	es       *ESClient
	db       *gorm.DB
	tagModel *model.TagCacheModel
	catModel *model.CategoryModel
}

// NewSyncService 创建同步服务
func NewSyncService(es *ESClient, db *gorm.DB, tagModel *model.TagCacheModel, catModel *model.CategoryModel) *SyncService {
	return &SyncService{
		es:       es,
		db:       db,
		tagModel: tagModel,
		catModel: catModel,
	}
}

// ==================== 单个文档同步 ====================

// IndexActivity 索引单个活动（创建/更新时调用）
//
// 使用外部版本控制：
// - 版本号 = UpdatedAt 时间戳
// - 只有版本更大时才会更新
// - 防止并发写入时旧数据覆盖新数据
func (s *SyncService) IndexActivity(ctx context.Context, activity *model.Activity) error {
	if s.es == nil {
		return nil // ES 未启用
	}

	// 1. 转换为 ES 文档
	doc, err := s.convertToDoc(ctx, activity)
	if err != nil {
		logx.Errorf("[ESSync] 转换文档失败 id=%d: %v", activity.ID, err)
		return err
	}

	// 2. 使用 UpdatedAt 作为外部版本号
	version := activity.UpdatedAt

	// 3. 索引文档
	_, err = s.es.client.Index().
		Index(s.es.indexName).
		Id(DocID(activity.ID)).
		BodyJson(doc).
		VersionType("external"). // 使用外部版本控制
		Version(version).        // 版本号 = 更新时间戳
		Refresh("false").        // 不立即刷新，提高写入性能
		Do(ctx)

	if err != nil {
		// 版本冲突不是错误，说明已有更新的数据
		if elastic.IsConflict(err) {
			logx.Infof("[ESSync] 跳过旧版本数据 id=%d", activity.ID)
			return nil
		}
		logx.Errorf("[ESSync] 索引活动失败 id=%d: %v", activity.ID, err)
		return err
	}

	logx.Infof("[ESSync] 索引活动成功 id=%d, version=%d", activity.ID, version)
	return nil
}

// IndexActivityAsync 异步索引活动（带重试）
//
// 用于 Create/Update Logic 中，不阻塞主流程
// 失败时记录日志，后续可扩展失败队列
func (s *SyncService) IndexActivityAsync(activity *model.Activity) {
	if s.es == nil {
		return
	}

	// 值拷贝，防止调用方后续修改导致 goroutine 读到不一致数据
	activityCopy := *activity
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 重试 3 次
		var err error
		for i := 0; i < 3; i++ {
			err = s.IndexActivity(ctx, &activityCopy)
			if err == nil {
				return
			}
			logx.Errorf("[ESSync] 异步索引重试 %d/3, id=%d: %v", i+1, activityCopy.ID, err)
			time.Sleep(time.Duration(i+1) * time.Second) // 1s, 2s, 3s
		}

		// 3 次重试失败，记录日志
		// TODO: 写入失败队列表，由定时任务重试
		logx.Errorf("[ESSync] 异步索引最终失败 id=%d: %v", activityCopy.ID, err)
	}()
}

// DeleteActivity 删除活动索引
func (s *SyncService) DeleteActivity(ctx context.Context, id uint64) error {
	if s.es == nil {
		return nil
	}

	_, err := s.es.client.Delete().
		Index(s.es.indexName).
		Id(DocID(id)).
		Refresh("false").
		Do(ctx)

	if err != nil {
		// 忽略文档不存在的错误
		if elastic.IsNotFound(err) {
			logx.Infof("[ESSync] 文档不存在，跳过删除 id=%d", id)
			return nil
		}
		logx.Errorf("[ESSync] 删除活动索引失败 id=%d: %v", id, err)
		return err
	}

	logx.Infof("[ESSync] 删除活动索引成功 id=%d", id)
	return nil
}

// DeleteActivityAsync 异步删除活动索引
func (s *SyncService) DeleteActivityAsync(id uint64) {
	if s.es == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.DeleteActivity(ctx, id); err != nil {
			logx.Errorf("[ESSync] 异步删除失败 id=%d: %v", id, err)
		}
	}()
}

// ==================== 批量同步 ====================

// FullSync 全量同步（定时任务调用）
//
// 同步策略：
// - 分批查询，每批 500 条，防止 OOM
// - 使用游标分页（基于 ID），避免深分页问题
// - 使用外部版本控制，防止旧数据覆盖
func (s *SyncService) FullSync(ctx context.Context) error {
	if s.es == nil {
		return nil
	}

	startTime := time.Now()
	logx.Info("[ESSync] 开始全量同步...")

	const batchSize = 500
	var (
		lastID     uint64 = 0
		totalCount int    = 0
		failCount  int    = 0
	)

	for {
		// 1. 分批查询（游标分页，基于 ID）
		var activities []model.Activity
		err := s.db.WithContext(ctx).
			Where("id > ?", lastID).
			Where("status IN ?", []int8{model.StatusPublished, model.StatusOngoing, model.StatusFinished}).
			Where("deleted_at IS NULL").
			Order("id ASC").
			Limit(batchSize).
			Find(&activities).Error

		if err != nil {
			return fmt.Errorf("查询活动失败: %w", err)
		}

		// 没有更多数据，退出循环
		if len(activities) == 0 {
			break
		}

		// 更新游标
		lastID = activities[len(activities)-1].ID

		// 2. 转换文档（绑定 doc 和对应的 activity 元数据，防止索引对齐问题）
		type docEntry struct {
			doc     ActivityDoc
			id      uint64
			version int64
		}
		entries := make([]docEntry, 0, len(activities))
		for i := range activities {
			doc, err := s.convertToDoc(ctx, &activities[i])
			if err != nil {
				logx.Errorf("[ESSync] 转换文档失败 id=%d: %v", activities[i].ID, err)
				failCount++
				continue
			}
			entries = append(entries, docEntry{
				doc:     doc,
				id:      activities[i].ID,
				version: activities[i].UpdatedAt,
			})
		}

		// 3. 批量索引
		bulkRequest := s.es.client.Bulk().Index(s.es.indexName)
		for _, entry := range entries {
			req := elastic.NewBulkIndexRequest().
				Id(DocID(entry.id)).
				VersionType("external").
				Version(entry.version).
				Doc(entry.doc)
			bulkRequest.Add(req)
		}

		// 4. 执行批量索引
		response, err := bulkRequest.Do(ctx)
		if err != nil {
			logx.Errorf("[ESSync] 批量索引失败 lastID=%d: %v", lastID, err)
			failCount += len(entries)
			continue
		}

		// 5. 统计
		totalCount += len(response.Succeeded())
		failCount += len(response.Failed())

		// 记录失败项
		for _, item := range response.Failed() {
			logx.Errorf("[ESSync] 索引失败 id=%s: %s", item.Id, item.Error.Reason)
		}

		logx.Infof("[ESSync] 全量同步进度: 已处理到 ID=%d, 本批=%d", lastID, len(activities))

		// 如果本批不足 batchSize，说明已到末尾
		if len(activities) < batchSize {
			break
		}

		// 短暂休眠，避免数据库压力
		time.Sleep(100 * time.Millisecond)
	}

	logx.Infof("[ESSync] 全量同步完成: 成功=%d, 失败=%d, 耗时=%v",
		totalCount, failCount, time.Since(startTime))

	return nil
}

// ==================== 文档转换 ====================

// convertToDoc 将 Activity 转换为 ES 文档
func (s *SyncService) convertToDoc(ctx context.Context, activity *model.Activity) (ActivityDoc, error) {
	doc := ActivityDoc{
		// 基础字段
		ID:          activity.ID,
		Title:       activity.Title,
		Description: activity.Description,
		CategoryID:  activity.CategoryID,
		Status:      activity.Status,

		// 组织者信息
		OrganizerID:     activity.OrganizerID,
		OrganizerName:   activity.OrganizerName,
		OrganizerAvatar: activity.OrganizerAvatar,

		// 地点信息
		Location:      activity.Location,
		AddressDetail: activity.AddressDetail,

		// 时间信息
		RegisterStartTime: activity.RegisterStartTime,
		RegisterEndTime:   activity.RegisterEndTime,
		ActivityStartTime: activity.ActivityStartTime,
		ActivityEndTime:   activity.ActivityEndTime,

		// 名额与统计
		MaxParticipants:     activity.MaxParticipants,
		CurrentParticipants: activity.CurrentParticipants,
		ViewCount:           activity.ViewCount,

		// 封面
		CoverURL:  activity.CoverURL,
		CoverType: activity.CoverType,

		// 时间戳
		CreatedAt: activity.CreatedAt,
		UpdatedAt: activity.UpdatedAt,
	}

	// 地理坐标（如果有）
	if activity.Latitude != 0 || activity.Longitude != 0 {
		doc.GeoLocation = &GeoPoint{
			Lat: activity.Latitude,
			Lon: activity.Longitude,
		}
	}

	// 获取分类名称（容错处理）
	if s.catModel != nil {
		if category, err := s.catModel.FindByID(ctx, activity.CategoryID); err == nil && category != nil {
			doc.CategoryName = category.Name
		}
	}

	// 获取标签（容错处理）
	if s.tagModel != nil {
		if tags, err := s.tagModel.FindByActivityID(ctx, activity.ID); err == nil {
			doc.Tags = make([]string, len(tags))
			for i, tag := range tags {
				doc.Tags[i] = tag.Name
			}
		}
	}

	// 添加搜索建议
	doc.Suggest = &Suggest{
		Input: []string{activity.Title},
		Contexts: map[string]string{
			"category": fmt.Sprintf("%d", activity.CategoryID),
		},
	}

	return doc, nil
}

// ==================== 增量同步 ====================

// IncrementalSync 增量同步（基于更新时间）
func (s *SyncService) IncrementalSync(ctx context.Context, since time.Time) error {
	if s.es == nil {
		return nil
	}

	logx.Infof("[ESSync] 开始增量同步，起始时间: %v", since)

	var activities []model.Activity
	err := s.db.WithContext(ctx).
		Where("updated_at >= ?", since.Unix()).
		Where("deleted_at IS NULL").
		Find(&activities).Error

	if err != nil {
		return fmt.Errorf("查询增量数据失败: %w", err)
	}

	successCount := 0
	for i := range activities {
		if err := s.IndexActivity(ctx, &activities[i]); err != nil {
			logx.Errorf("[ESSync] 增量同步失败 id=%d: %v", activities[i].ID, err)
		} else {
			successCount++
		}
	}

	logx.Infof("[ESSync] 增量同步完成，共处理 %d 条，成功 %d 条", len(activities), successCount)
	return nil
}
