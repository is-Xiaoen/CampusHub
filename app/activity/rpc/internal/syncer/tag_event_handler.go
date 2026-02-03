package syncer

import (
	"context"
	"encoding/json"
	"time"

	"activity-platform/app/activity/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// ==================== 标签事件处理器 ====================
//
// 用途：处理用户服务发送的标签变更事件（MQ 消费）
// 设计要点：
//   1. 幂等性：通过 event_id + 时间戳去重
//   2. 顺序性：通过 updated_at 时间戳判断是否过期事件
//   3. 补偿机制：处理标签删除时的关联清理
//   4. 反向通知：将活动服务的统计数据同步回用户服务

// TagEventType 标签事件类型
type TagEventType string

const (
	TagEventCreated  TagEventType = "tag.created"  // 标签创建
	TagEventUpdated  TagEventType = "tag.updated"  // 标签更新
	TagEventDeleted  TagEventType = "tag.deleted"  // 标签删除
	TagEventDisabled TagEventType = "tag.disabled" // 标签禁用
)

// TagEvent 标签变更事件（MQ 消息体）
//
// 用户服务发送此消息到 MQ，活动服务消费处理
// Topic: activity.tag.sync
type TagEvent struct {
	EventID   string       `json:"event_id"`   // 事件唯一ID（用于幂等）
	EventType TagEventType `json:"event_type"` // 事件类型
	Timestamp int64        `json:"timestamp"`  // 事件时间戳
	Tag       TagEventData `json:"tag"`        // 标签数据
}

// TagEventData 标签事件数据
type TagEventData struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Icon        string `json:"icon"`
	Status      int8   `json:"status"`
	Description string `json:"description"`
	UpdatedAt   int64  `json:"updated_at"`
}

// TagEventHandler 标签事件处理器
type TagEventHandler struct {
	db               *gorm.DB
	tagCacheModel    *model.TagCacheModel
	activityTagModel *model.ActivityTagModel
	tagStatsModel    *model.ActivityTagStatsModel
	metrics          *SyncMetrics // 指标收集器
}

// NewTagEventHandler 创建标签事件处理器
func NewTagEventHandler(
	db *gorm.DB,
	tagCacheModel *model.TagCacheModel,
	activityTagModel *model.ActivityTagModel,
	tagStatsModel *model.ActivityTagStatsModel,
) *TagEventHandler {
	return &TagEventHandler{
		db:               db,
		tagCacheModel:    tagCacheModel,
		activityTagModel: activityTagModel,
		tagStatsModel:    tagStatsModel,
		metrics:          GetSyncMetrics(), // 使用全局指标实例
	}
}

// HandleMessage 处理 MQ 消息（MQ 消费者调用）
//
// 设计要点：
//   - 解析消息体
//   - 幂等性检查（基于 event_id）
//   - 路由到具体处理方法
//   - 记录处理结果和指标
func (h *TagEventHandler) HandleMessage(ctx context.Context, message []byte) error {
	// 记录收到消息
	h.metrics.RecordMQMessageReceived()

	// 1. 解析消息
	var event TagEvent
	if err := json.Unmarshal(message, &event); err != nil {
		logx.Errorf("[TagEventHandler] 解析消息失败: %v, raw: %s", err, string(message))
		return nil // 解析失败不重试，记录日志后丢弃
	}

	// 2. 基本校验
	if event.EventID == "" || event.Tag.ID == 0 {
		logx.Errorf("[TagEventHandler] 消息格式无效: %+v", event)
		return nil
	}

	logx.Infof("[TagEventHandler] 收到事件: type=%s, tag_id=%d, event_id=%s",
		event.EventType, event.Tag.ID, event.EventID)

	// 3. 路由到具体处理方法
	var err error
	var skipped bool
	switch event.EventType {
	case TagEventCreated, TagEventUpdated:
		skipped, err = h.handleTagUpsertWithSkip(ctx, &event)
	case TagEventDeleted:
		err = h.handleTagDeleted(ctx, &event)
	case TagEventDisabled:
		err = h.handleTagDisabled(ctx, &event)
	default:
		logx.Infof("[TagEventHandler] 未知事件类型: %s", event.EventType)
		return nil
	}

	// 4. 记录处理结果指标
	if err != nil {
		h.metrics.RecordMQMessageFailed(err)
		logx.Errorf("[TagEventHandler] 处理事件失败: type=%s, tag_id=%d, err=%v",
			event.EventType, event.Tag.ID, err)
		return err // 返回错误，MQ 会重试
	}

	if skipped {
		h.metrics.RecordMQMessageSkipped()
		logx.Infof("[TagEventHandler] 事件已跳过(幂等): type=%s, tag_id=%d",
			event.EventType, event.Tag.ID)
	} else {
		h.metrics.RecordMQMessageProcessed()
		logx.Infof("[TagEventHandler] 事件处理成功: type=%s, tag_id=%d",
			event.EventType, event.Tag.ID)
	}

	return nil
}

// handleTagUpsertWithSkip 处理标签创建/更新事件（返回是否跳过）
//
// 幂等性保证：通过 updated_at 时间戳判断
//   - 如果本地 tag_cache.updated_at >= event.updated_at，跳过（过期事件）
//   - 否则更新本地缓存
//
// 返回值：
//   - skipped: 是否因幂等性跳过
//   - err: 错误信息
func (h *TagEventHandler) handleTagUpsertWithSkip(ctx context.Context, event *TagEvent) (skipped bool, err error) {
	tagData := event.Tag

	// 1. 检查是否过期事件（幂等性）
	existingTag, findErr := h.tagCacheModel.FindByID(ctx, tagData.ID)
	if findErr == nil && existingTag != nil {
		if existingTag.UpdatedAt >= tagData.UpdatedAt {
			logx.Infof("[TagEventHandler] 跳过过期事件: tag_id=%d, local_updated=%d, event_updated=%d",
				tagData.ID, existingTag.UpdatedAt, tagData.UpdatedAt)
			return true, nil // 跳过，但不是错误
		}
	}

	// 2. 更新本地缓存
	tagCache := &model.TagCache{
		ID:          tagData.ID,
		Name:        tagData.Name,
		Color:       tagData.Color,
		Icon:        tagData.Icon,
		Status:      tagData.Status,
		Description: tagData.Description,
		SyncedAt:    time.Now().Unix(),
		UpdatedAt:   tagData.UpdatedAt,
	}

	return false, h.tagCacheModel.Upsert(ctx, tagCache)
}

// handleTagUpsert 处理标签创建/更新事件（兼容旧接口）
func (h *TagEventHandler) handleTagUpsert(ctx context.Context, event *TagEvent) error {
	_, err := h.handleTagUpsertWithSkip(ctx, event)
	return err
}

// handleTagDeleted 处理标签删除事件
//
// 关键操作（事务内）：
//  1. 获取所有使用该标签的活动 ID
//  2. 删除 activity_tags 关联记录
//  3. 删除 activity_tag_stats 统计记录
//  4. 删除 tag_cache 缓存记录
//
// 面试亮点：这里展示了"补偿机制"——删除标签时清理关联数据
func (h *TagEventHandler) handleTagDeleted(ctx context.Context, event *TagEvent) error {
	tagID := event.Tag.ID

	return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 删除活动-标签关联（activity_tags 表）
		if err := tx.Where("tag_id = ?", tagID).Delete(&model.ActivityTag{}).Error; err != nil {
			return err
		}

		// 2. 删除标签统计（activity_tag_stats 表）
		if err := tx.Where("tag_id = ?", tagID).Delete(&model.ActivityTagStats{}).Error; err != nil {
			return err
		}

		// 3. 删除标签缓存（tag_cache 表）
		if err := tx.Where("id = ?", tagID).Delete(&model.TagCache{}).Error; err != nil {
			return err
		}

		logx.Infof("[TagEventHandler] 标签删除处理完成: tag_id=%d", tagID)
		return nil
	})
}

// handleTagDisabled 处理标签禁用事件
//
// 与删除不同：
//   - 不删除关联记录（活动仍显示该标签，但标记为"已禁用"）
//   - 只更新 tag_cache.status = 0
//   - 新创建的活动无法选择该标签（ExistsByIDs 会过滤 status=0）
func (h *TagEventHandler) handleTagDisabled(ctx context.Context, event *TagEvent) error {
	tagData := event.Tag

	// 更新本地缓存状态
	tagCache := &model.TagCache{
		ID:          tagData.ID,
		Name:        tagData.Name,
		Color:       tagData.Color,
		Icon:        tagData.Icon,
		Status:      0, // 禁用状态
		Description: tagData.Description,
		SyncedAt:    time.Now().Unix(),
		UpdatedAt:   tagData.UpdatedAt,
	}

	return h.tagCacheModel.Upsert(ctx, tagCache)
}

// ==================== 反向统计同步（活动服务 → 用户服务） ====================

// TagUsageStats 标签使用统计（反向同步给用户服务）
type TagUsageStats struct {
	TagID         uint64 `json:"tag_id"`
	ActivityCount uint32 `json:"activity_count"` // 被多少活动使用
	TotalViews    uint64 `json:"total_views"`    // 关联活动的总浏览量
}

// GetTagUsageStats 获取标签使用统计（供用户服务 RPC 调用）
//
// 这个方法可以：
//  1. 被用户服务 RPC 调用（拉模式）
//  2. 定时推送到用户服务（推模式）
func (h *TagEventHandler) GetTagUsageStats(ctx context.Context, tagIDs []uint64) ([]TagUsageStats, error) {
	if len(tagIDs) == 0 {
		return []TagUsageStats{}, nil
	}

	statsList, err := h.tagStatsModel.FindByTagIDs(ctx, tagIDs)
	if err != nil {
		return nil, err
	}

	result := make([]TagUsageStats, len(statsList))
	for i, stats := range statsList {
		result[i] = TagUsageStats{
			TagID:         stats.TagID,
			ActivityCount: stats.ActivityCount,
			TotalViews:    stats.ViewCount,
		}
	}
	return result, nil
}
