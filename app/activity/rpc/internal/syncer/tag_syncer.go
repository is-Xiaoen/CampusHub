package syncer

import (
	"context"
	"sync"
	"time"

	"activity-platform/app/activity/model"
	"activity-platform/app/user/rpc/client/tagservice"

	"github.com/zeromicro/go-zero/core/logx"
)

// TagSyncer 标签同步器
//
// 功能：从用户服务同步标签数据到本地 tag_cache 表
//
// 同步策略：
//  1. 服务启动时：执行一次全量同步
//  2. 定时任务：每 5 分钟增量同步（基于 since_timestamp）
//  3. 手动触发：支持调用 SyncNow() 立即同步
//
// 技术要点：
//   - 使用 sync.Once 确保启动同步只执行一次
//   - 使用 context 支持优雅关闭
//   - 记录 lastSyncTime 实现增量同步
type TagSyncer struct {
	tagRpc        tagservice.TagService
	tagCacheModel *model.TagCacheModel

	interval     time.Duration // 同步间隔
	lastSyncTime int64         // 上次同步时间戳（用于增量同步）

	stopCh   chan struct{}
	stopOnce sync.Once
	startMu  sync.Mutex
	started  bool
}

// NewTagSyncer 创建标签同步器
//
// 参数：
//   - tagRpc: 用户服务的 TagService RPC 客户端
//   - tagCacheModel: 本地 tag_cache 表的 Model
//   - interval: 同步间隔（建议 5 分钟）
func NewTagSyncer(tagRpc tagservice.TagService, tagCacheModel *model.TagCacheModel, interval time.Duration) *TagSyncer {
	return &TagSyncer{
		tagRpc:        tagRpc,
		tagCacheModel: tagCacheModel,
		interval:      interval,
		lastSyncTime:  0,
		stopCh:        make(chan struct{}),
	}
}

// Start 启动同步器
//
// 行为：
//  1. 立即执行一次全量同步
//  2. 启动定时器，按 interval 间隔执行增量同步
//  3. 支持重复调用（幂等）
func (s *TagSyncer) Start() {
	s.startMu.Lock()
	if s.started {
		s.startMu.Unlock()
		return
	}
	s.started = true
	s.startMu.Unlock()

	logx.Info("[TagSyncer] 标签同步器启动")

	// 1. 立即执行一次全量同步
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.syncAll(ctx); err != nil {
			logx.Errorf("[TagSyncer] 启动全量同步失败: %v", err)
		} else {
			logx.Info("[TagSyncer] 启动全量同步完成")
		}
	}()

	// 2. 启动定时同步
	go s.runLoop()
}

// Stop 停止同步器
func (s *TagSyncer) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
		logx.Info("[TagSyncer] 标签同步器已停止")
	})
}

// SyncNow 立即触发一次同步（手动调用）
func (s *TagSyncer) SyncNow(ctx context.Context) error {
	return s.syncIncremental(ctx)
}

// runLoop 定时同步循环
func (s *TagSyncer) runLoop() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := s.syncIncremental(ctx); err != nil {
				logx.Errorf("[TagSyncer] 定时增量同步失败: %v", err)
			}
			cancel()
		}
	}
}

// syncAll 全量同步（since_timestamp = 0）
func (s *TagSyncer) syncAll(ctx context.Context) error {
	return s.doSync(ctx, 0)
}

// syncIncremental 增量同步（基于 lastSyncTime）
func (s *TagSyncer) syncIncremental(ctx context.Context) error {
	return s.doSync(ctx, s.lastSyncTime)
}

// doSync 执行同步
//
// 参数：
//   - sinceTimestamp: 0 表示全量，>0 表示增量（只获取该时间戳之后更新的标签）
func (s *TagSyncer) doSync(ctx context.Context, sinceTimestamp int64) error {
	syncType := "增量"
	if sinceTimestamp == 0 {
		syncType = "全量"
	}

	// 1. 调用用户服务获取标签
	resp, err := s.tagRpc.GetAllTags(ctx, &tagservice.GetAllTagsReq{
		SinceTimestamp: sinceTimestamp,
	})
	if err != nil {
		logx.Errorf("[TagSyncer] 调用 GetAllTags 失败: %v", err)
		return err
	}

	// 2. 没有新数据，跳过
	if len(resp.Tags) == 0 {
		logx.Infof("[TagSyncer] %s同步完成，无新数据", syncType)
		s.lastSyncTime = resp.ServerTimestamp
		return nil
	}

	// 3. 转换为本地模型
	tagCaches := make([]model.TagCache, len(resp.Tags))
	now := time.Now().Unix()
	for i, tag := range resp.Tags {
		tagCaches[i] = model.TagCache{
			ID:          tag.Id,
			Name:        tag.Name,
			Color:       tag.Color,
			Icon:        tag.Icon,
			Status:      int8(tag.Status), // proto 中是 uint64，本地存储用 int8
			Description: tag.Description,
			SyncedAt:    now,
		}
	}

	// 4. 批量更新到本地 tag_cache 表
	if err := s.tagCacheModel.UpsertBatch(ctx, tagCaches); err != nil {
		logx.Errorf("[TagSyncer] 批量更新 tag_cache 失败: %v", err)
		return err
	}

	// 5. 更新最后同步时间
	s.lastSyncTime = resp.ServerTimestamp

	logx.Infof("[TagSyncer] %s同步完成，同步 %d 条标签", syncType, len(resp.Tags))
	return nil
}
