package syncer

import (
	"context"
	"fmt"
	"time"

	"activity-platform/app/activity/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// ==================== 标签数据对账服务 ====================
//
// 企业级最终一致性保障的三重机制：
//   1. 实时同步（MQ）- 秒级延迟
//   2. 定时同步（TagSyncer）- 5 分钟兜底
//   3. 数据对账（TagReconciler）- 每日全量校验 ← 本文件
//
// 面试亮点：
//   - 体现"防御性设计"思想
//   - 展示对"最终一致性"的深度理解
//   - 实际生产环境必备的数据治理能力

// ReconcileResult 对账结果
type ReconcileResult struct {
	TotalChecked   int      // 检查的标签总数
	Matched        int      // 一致的数量
	Mismatched     int      // 不一致的数量
	MissingInLocal int      // 本地缺失的数量
	Repaired       int      // 自动修复的数量
	FailedRepairs  int      // 修复失败的数量
	MismatchedIDs  []uint64 // 不一致的标签 ID（用于告警）
	Duration       time.Duration
}

// TagReconciler 标签数据对账器
//
// 职责：
//   - 定期对比用户服务和活动服务的标签数据
//   - 发现不一致时自动修复或告警
//   - 生成对账报告（可接入监控系统）
//
// 企业级特性：
//   - 分布式锁防止多实例重复执行
//   - 指标收集对接 Prometheus
//   - 多级告警（日志/Webhook）
type TagReconciler struct {
	db            *gorm.DB
	tagCacheModel *model.TagCacheModel
	userTagRPC    UserTagRPCClient // 用户服务 RPC 客户端
	alerter       Alerter          // 告警接口
	lock          DistributedLock  // 分布式锁（防止多实例重复执行）
	metrics       *SyncMetrics     // 指标收集器
}

// UserTagRPCClient 用户服务标签 RPC 接口（由调用方注入）
type UserTagRPCClient interface {
	// GetAllTags 获取用户服务所有启用的标签
	GetAllTags(ctx context.Context) ([]TagSyncData, error)
	// GetTagsByIDs 根据 ID 批量获取标签
	GetTagsByIDs(ctx context.Context, ids []uint64) ([]TagSyncData, error)
}

// Alerter 告警接口
type Alerter interface {
	// SendAlert 发送告警
	SendAlert(ctx context.Context, level AlertLevel, title, content string) error
}

// AlertLevel 告警级别
type AlertLevel int

const (
	AlertLevelInfo    AlertLevel = 1 // 信息
	AlertLevelWarning AlertLevel = 2 // 警告
	AlertLevelError   AlertLevel = 3 // 错误
)

// TagSyncData 标签同步数据（与用户服务的返回结构对应）
//
// 这是接口契约，由 UserTagRPCClient 的实现方保证数据映射
type TagSyncData struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Icon        string `json:"icon"`
	Status      int8   `json:"status"`
	Description string `json:"description"`
	UpdatedAt   int64  `json:"updated_at"`
}

// NewTagReconciler 创建标签对账器
func NewTagReconciler(
	db *gorm.DB,
	tagCacheModel *model.TagCacheModel,
	userTagRPC UserTagRPCClient,
	alerter Alerter,
) *TagReconciler {
	return &TagReconciler{
		db:            db,
		tagCacheModel: tagCacheModel,
		userTagRPC:    userTagRPC,
		alerter:       alerter,
		lock:          NewNoopLock(),    // 默认使用空锁（单实例）
		metrics:       GetSyncMetrics(), // 使用全局指标实例
	}
}

// NewTagReconcilerWithLock 创建带分布式锁的标签对账器
//
// 适用于多实例部署，确保只有一个实例执行对账
func NewTagReconcilerWithLock(
	db *gorm.DB,
	tagCacheModel *model.TagCacheModel,
	userTagRPC UserTagRPCClient,
	alerter Alerter,
	lock DistributedLock,
) *TagReconciler {
	return &TagReconciler{
		db:            db,
		tagCacheModel: tagCacheModel,
		userTagRPC:    userTagRPC,
		alerter:       alerter,
		lock:          lock,
		metrics:       GetSyncMetrics(),
	}
}

// Reconcile 执行数据对账
//
// 对账流程：
//  1. 从用户服务获取所有标签（权威数据源）
//  2. 从本地缓存获取所有标签
//  3. 对比差异：
//     - 本地缺失 → 自动补充
//     - 数据不一致 → 自动更新
//     - 本地多余 → 记录日志（不自动删除，防止误删）
//  4. 生成对账报告
//  5. 超过阈值则告警
func (r *TagReconciler) Reconcile(ctx context.Context) (*ReconcileResult, error) {
	startTime := time.Now()
	result := &ReconcileResult{}

	logx.Info("[TagReconciler] 开始数据对账...")

	// 1. 获取用户服务的标签（权威数据源）
	remoteTags, err := r.userTagRPC.GetAllTags(ctx)
	if err != nil {
		logx.Errorf("[TagReconciler] 获取用户服务标签失败: %v", err)
		return nil, fmt.Errorf("获取用户服务标签失败: %w", err)
	}

	// 2. 获取本地缓存的标签
	localTags, err := r.tagCacheModel.FindAll(ctx)
	if err != nil {
		logx.Errorf("[TagReconciler] 获取本地缓存标签失败: %v", err)
		return nil, fmt.Errorf("获取本地缓存标签失败: %w", err)
	}

	// 3. 构建映射表便于对比
	remoteMap := make(map[uint64]TagSyncData, len(remoteTags))
	for _, tag := range remoteTags {
		remoteMap[tag.ID] = tag
	}

	localMap := make(map[uint64]model.TagCache, len(localTags))
	for _, tag := range localTags {
		localMap[tag.ID] = tag
	}

	result.TotalChecked = len(remoteTags)

	// 4. 对比并修复
	var toUpsert []model.TagCache

	// 4.1 检查远程有、本地缺失或不一致的
	for id, remoteTag := range remoteMap {
		localTag, exists := localMap[id]

		if !exists {
			// 本地缺失
			result.MissingInLocal++
			toUpsert = append(toUpsert, r.convertToTagCache(remoteTag))
			logx.Infof("[TagReconciler] 发现缺失标签: id=%d, name=%s", id, remoteTag.Name)
		} else if !r.isConsistent(localTag, remoteTag) {
			// 数据不一致
			result.Mismatched++
			result.MismatchedIDs = append(result.MismatchedIDs, id)
			toUpsert = append(toUpsert, r.convertToTagCache(remoteTag))
			logx.Infof("[TagReconciler] 发现不一致标签: id=%d, local_updated=%d, remote_updated=%d",
				id, localTag.UpdatedAt, remoteTag.UpdatedAt)
		} else {
			result.Matched++
		}
	}

	// 4.2 检查本地有、远程没有的（可能是被删除的标签）
	for id := range localMap {
		if _, exists := remoteMap[id]; !exists {
			// 本地多余（远程已删除）
			// 这里只记录日志，不自动删除，由 TagEventHandler 处理删除事件
			logx.Infof("[TagReconciler][WARNING] 发现孤儿标签: id=%d (远程已删除但本地仍存在)", id)
		}
	}

	// 5. 执行修复
	if len(toUpsert) > 0 {
		err = r.tagCacheModel.UpsertBatch(ctx, toUpsert)
		if err != nil {
			logx.Errorf("[TagReconciler] 批量修复失败: %v", err)
			result.FailedRepairs = len(toUpsert)
		} else {
			result.Repaired = len(toUpsert)
			logx.Infof("[TagReconciler] 修复完成: %d 条记录", len(toUpsert))
		}
	}

	result.Duration = time.Since(startTime)

	// 6. 生成对账报告
	r.logReconcileResult(result)

	// 7. 超过阈值则告警
	if err := r.checkAndAlert(ctx, result); err != nil {
		logx.Errorf("[TagReconciler] 发送告警失败: %v", err)
	}

	return result, nil
}

// isConsistent 检查本地和远程数据是否一致
//
// 一致性判断标准：
//   - 核心字段相同（name, color, icon, status）
//   - 本地 updated_at >= 远程 updated_at（允许本地更新）
func (r *TagReconciler) isConsistent(local model.TagCache, remote TagSyncData) bool {
	// 如果远程数据更新，则认为不一致
	if remote.UpdatedAt > local.UpdatedAt {
		return false
	}

	// 核心字段对比
	return local.Name == remote.Name &&
		local.Color == remote.Color &&
		local.Icon == remote.Icon &&
		local.Status == remote.Status
}

// convertToTagCache 转换为本地缓存模型
func (r *TagReconciler) convertToTagCache(data TagSyncData) model.TagCache {
	return model.TagCache{
		ID:          data.ID,
		Name:        data.Name,
		Color:       data.Color,
		Icon:        data.Icon,
		Status:      data.Status,
		Description: data.Description,
		SyncedAt:    time.Now().Unix(),
		UpdatedAt:   data.UpdatedAt,
	}
}

// logReconcileResult 记录对账结果日志
func (r *TagReconciler) logReconcileResult(result *ReconcileResult) {
	logx.Infof("[TagReconciler] 对账完成 | "+
		"总计检查: %d | 一致: %d | 不一致: %d | 本地缺失: %d | "+
		"已修复: %d | 修复失败: %d | 耗时: %v",
		result.TotalChecked, result.Matched, result.Mismatched, result.MissingInLocal,
		result.Repaired, result.FailedRepairs, result.Duration)
}

// checkAndAlert 检查是否需要告警
//
// 告警策略：
//   - 不一致率 > 5%：Warning
//   - 不一致率 > 10%：Error
//   - 修复失败：Error
func (r *TagReconciler) checkAndAlert(ctx context.Context, result *ReconcileResult) error {
	if r.alerter == nil {
		return nil
	}

	// 计算不一致率
	if result.TotalChecked == 0 {
		return nil
	}

	inconsistentRate := float64(result.Mismatched+result.MissingInLocal) / float64(result.TotalChecked) * 100

	// 修复失败：Error 级别告警
	if result.FailedRepairs > 0 {
		return r.alerter.SendAlert(ctx, AlertLevelError,
			"[活动服务] 标签对账修复失败",
			fmt.Sprintf("修复失败数量: %d，请检查数据库连接和用户服务状态", result.FailedRepairs))
	}

	// 不一致率 > 10%：Error 级别告警
	if inconsistentRate > 10 {
		return r.alerter.SendAlert(ctx, AlertLevelError,
			"[活动服务] 标签数据不一致率过高",
			fmt.Sprintf("不一致率: %.2f%%，不一致标签ID: %v", inconsistentRate, result.MismatchedIDs))
	}

	// 不一致率 > 5%：Warning 级别告警
	if inconsistentRate > 5 {
		return r.alerter.SendAlert(ctx, AlertLevelWarning,
			"[活动服务] 标签数据存在不一致",
			fmt.Sprintf("不一致率: %.2f%%，已自动修复 %d 条", inconsistentRate, result.Repaired))
	}

	return nil
}

// ==================== 定时任务调度 ====================

// StartReconcileJob 启动对账定时任务
//
// 调度策略：
//   - 每天凌晨 3:00 执行全量对账
//   - 使用分布式锁防止多实例重复执行
func (r *TagReconciler) StartReconcileJob(ctx context.Context) {
	logx.Info("[TagReconciler] 对账定时任务已启动")

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// 计算到下一个凌晨 3 点的时间
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
	if now.After(next) {
		next = next.Add(24 * time.Hour)
	}
	initialDelay := time.Until(next)

	logx.Infof("[TagReconciler] 下次对账时间: %v (%.2f 小时后)", next, initialDelay.Hours())

	// 等待到凌晨 3 点
	select {
	case <-ctx.Done():
		return
	case <-time.After(initialDelay):
	}

	// 执行首次对账
	r.executeReconcile(ctx)

	// 之后每 24 小时执行一次
	for {
		select {
		case <-ctx.Done():
			logx.Info("[TagReconciler] 对账定时任务已停止")
			return
		case <-ticker.C:
			r.executeReconcile(ctx)
		}
	}
}

// executeReconcile 执行对账（带分布式锁和重试）
func (r *TagReconciler) executeReconcile(ctx context.Context) {
	// 1. 尝试获取分布式锁
	acquired, err := r.lock.TryLock(ctx)
	if err != nil {
		logx.Errorf("[TagReconciler] 获取分布式锁失败: %v", err)
		return
	}
	if !acquired {
		logx.Infof("[TagReconciler] 其他实例正在执行对账，跳过本次执行")
		return
	}
	defer func() {
		if err := r.lock.Unlock(ctx); err != nil {
			logx.Errorf("[TagReconciler] 释放分布式锁失败: %v", err)
		}
	}()

	// 2. 带重试执行对账
	const maxRetries = 3

	for i := 0; i < maxRetries; i++ {
		result, err := r.Reconcile(ctx)
		if err != nil {
			logx.Errorf("[TagReconciler] 对账执行失败 (重试 %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(time.Duration(i+1) * time.Minute) // 指数退避
			continue
		}

		// 3. 记录指标
		r.metrics.RecordReconcileResult(result)

		logx.Infof("[TagReconciler] 对账执行成功: matched=%d, repaired=%d",
			result.Matched, result.Repaired)
		return
	}

	// 重试耗尽，发送告警
	if r.alerter != nil {
		_ = r.alerter.SendAlert(ctx, AlertLevelError,
			"[活动服务] 标签对账任务执行失败",
			fmt.Sprintf("重试 %d 次后仍失败，请人工检查", maxRetries))
	}
}
