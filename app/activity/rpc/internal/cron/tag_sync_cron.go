package cron

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"activity-platform/app/activity/model"
	"activity-platform/app/user/rpc/client/tagservice"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// ==================== 常量定义 ====================

const (
	// 分布式锁配置
	tagSyncLockKey        = "activity:cron:tag_sync"
	tagSyncLockExpire     = 30  // 锁过期时间（秒）
	tagSyncDefaultSeconds = 300 // 默认同步间隔：5 分钟
)

// ==================== TagSyncCron 标签同步定时任务 ====================

// TagSyncCron 标签数据同步定时任务
//
// 功能说明：
//   - 定期从用户服务（TagRpc）拉取所有兴趣标签
//   - 同步到活动服务本地的 tag_cache 表
//   - 保证活动列表/详情返回的标签数据（含 icon、color）是最新的
//
// 执行策略：
//   - 启动时立即执行一次全量同步
//   - 之后每 5 分钟执行一次
//   - 使用 Redis 分布式锁，多实例部署时只有一个实例执行
type TagSyncCron struct {
	redis         *redis.Redis
	tagRpc        tagservice.TagService
	tagCacheModel *model.TagCacheModel

	intervalSeconds int
	stopChan        chan struct{}
	running         atomic.Bool
	stopOnce        sync.Once
	ownerID         string
}

// NewTagSyncCron 创建标签同步定时任务
func NewTagSyncCron(
	rds *redis.Redis,
	tagRpc tagservice.TagService,
	tagCacheModel *model.TagCacheModel,
) *TagSyncCron {
	return &TagSyncCron{
		redis:           rds,
		tagRpc:          tagRpc,
		tagCacheModel:   tagCacheModel,
		intervalSeconds: tagSyncDefaultSeconds,
		stopChan:        make(chan struct{}),
		ownerID:         uuid.New().String(),
	}
}

// Start 启动定时任务
func (c *TagSyncCron) Start() {
	if !c.running.CompareAndSwap(false, true) {
		logx.Info("[TagSyncCron] 定时任务已在运行中，跳过重复启动")
		return
	}

	logx.Infof("[TagSyncCron] 启动标签同步定时任务，执行间隔: %d 秒, owner: %s",
		c.intervalSeconds, c.ownerID)

	go func() {
		// 启动后立即执行一次全量同步
		c.execute()

		ticker := time.NewTicker(time.Duration(c.intervalSeconds) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.execute()
			case <-c.stopChan:
				logx.Info("[TagSyncCron] 标签同步定时任务已停止")
				return
			}
		}
	}()
}

// Stop 停止定时任务
func (c *TagSyncCron) Stop() {
	if !c.running.Load() {
		return
	}
	c.stopOnce.Do(func() {
		close(c.stopChan)
	})
	c.running.Store(false)
}

// execute 执行标签同步
func (c *TagSyncCron) execute() {
	ctx := context.Background()

	// 1. 尝试获取分布式锁
	locked, err := c.redis.SetnxExCtx(ctx, tagSyncLockKey, c.ownerID, tagSyncLockExpire)
	if err != nil {
		logx.Errorf("[TagSyncCron] 获取锁失败: %v", err)
		return
	}
	if !locked {
		return // 其他实例正在执行
	}
	defer c.releaseLock(ctx)

	// 2. 从用户服务拉取所有兴趣标签
	resp, err := c.tagRpc.GetAllInterestTags(ctx, &tagservice.GetAllInterestTagsReq{})
	if err != nil {
		logx.Errorf("[TagSyncCron] 拉取兴趣标签失败: %v", err)
		return
	}

	if len(resp.InterestTags) == 0 {
		logx.Info("[TagSyncCron] 用户服务返回空标签列表，跳过同步")
		return
	}

	// 3. 转换为 TagCache 模型
	now := time.Now().Unix()
	tagCaches := make([]model.TagCache, len(resp.InterestTags))
	for i, tag := range resp.InterestTags {
		tagCaches[i] = model.TagCache{
			ID:          tag.Id,
			Name:        tag.TagName,
			Color:       tag.TagColor,
			Icon:        tag.TagIcon,
			Status:      1, // 启用
			Description: tag.TagDesc,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	}

	// 4. 批量 Upsert 到 tag_cache 表
	if err := c.tagCacheModel.UpsertBatch(ctx, tagCaches); err != nil {
		logx.Errorf("[TagSyncCron] 批量写入标签缓存失败: %v", err)
		return
	}

	logx.Infof("[TagSyncCron] 标签同步完成: 同步 %d 个标签", len(tagCaches))
}

// releaseLock 释放分布式锁（仅 owner 匹配时才删除）
func (c *TagSyncCron) releaseLock(ctx context.Context) {
	result, err := c.redis.EvalCtx(ctx, unlockScript, []string{tagSyncLockKey}, c.ownerID)
	if err != nil {
		logx.Errorf("[TagSyncCron] 释放锁失败: %v", err)
		return
	}
	if fmt.Sprintf("%v", result) == "0" {
		logx.Infof("[TagSyncCron] 锁已被其他实例持有，跳过释放")
	}
}
