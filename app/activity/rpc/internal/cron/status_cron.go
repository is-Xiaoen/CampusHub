package cron

import (
	"context"
	"fmt"
	"time"

	"activity-platform/app/activity/model"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"
)

// ==================== 常量定义 ====================

const (
	// 分布式锁配置
	lockKeyPrefix     = "activity:cron:status:"
	lockExpireSeconds = 60 // 锁过期时间（秒）

	// 批量处理配置
	batchSize = 100 // 每批处理数量

	// 默认执行间隔（秒）
	defaultIntervalSeconds = 60
)

// ==================== StatusCron 状态定时任务 ====================

// StatusCron 活动状态自动流转定时任务
//
// 功能说明：
//   - 自动将已发布(Published)的活动流转为进行中(Ongoing)
//   - 自动将进行中(Ongoing)的活动流转为已结束(Finished)
//
// 执行策略：
//   - 默认每分钟执行一次
//   - 使用 Redis 分布式锁，确保多实例部署时只有一个实例执行
//   - 分批更新，避免锁表时间过长
//   - 记录状态变更日志
type StatusCron struct {
	redis          *redis.Redis
	db             *gorm.DB
	activityModel  *model.ActivityModel
	statusLogModel *model.ActivityStatusLogModel

	intervalSeconds int           // 执行间隔（秒）
	stopChan        chan struct{} // 停止信号
	running         bool          // 运行状态
}

// NewStatusCron 创建状态定时任务
func NewStatusCron(
	rds *redis.Redis,
	db *gorm.DB,
	activityModel *model.ActivityModel,
	statusLogModel *model.ActivityStatusLogModel,
) *StatusCron {
	return &StatusCron{
		redis:           rds,
		db:              db,
		activityModel:   activityModel,
		statusLogModel:  statusLogModel,
		intervalSeconds: defaultIntervalSeconds,
		stopChan:        make(chan struct{}),
	}
}

// SetInterval 设置执行间隔（秒）
func (c *StatusCron) SetInterval(seconds int) {
	if seconds > 0 {
		c.intervalSeconds = seconds
	}
}

// Start 启动定时任务
func (c *StatusCron) Start() {
	if c.running {
		logx.Info("[StatusCron] 定时任务已在运行中，跳过重复启动")
		return
	}
	c.running = true

	logx.Infof("[StatusCron] 启动状态自动流转定时任务，执行间隔: %d 秒", c.intervalSeconds)

	go func() {
		// 启动后立即执行一次
		c.execute()

		ticker := time.NewTicker(time.Duration(c.intervalSeconds) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.execute()
			case <-c.stopChan:
				logx.Info("[StatusCron] 定时任务已停止")
				return
			}
		}
	}()
}

// Stop 停止定时任务
func (c *StatusCron) Stop() {
	if !c.running {
		return
	}
	close(c.stopChan)
	c.running = false
}

// execute 执行状态流转
func (c *StatusCron) execute() {
	ctx := context.Background()
	now := time.Now().Unix()

	// 1. Published → Ongoing（活动开始）
	c.transitionPublishedToOngoing(ctx, now)

	// 2. Ongoing → Finished（活动结束）
	c.transitionOngoingToFinished(ctx, now)
}

// ==================== 状态流转实现 ====================

// transitionPublishedToOngoing 已发布 → 进行中
//
// 触发条件：activity_start_time <= now
// 场景：活动开始时间到达，自动开始活动
func (c *StatusCron) transitionPublishedToOngoing(ctx context.Context, now int64) {
	lockKey := lockKeyPrefix + "published_to_ongoing"

	// 尝试获取分布式锁
	locked, err := c.tryLock(ctx, lockKey)
	if err != nil {
		logx.Errorf("[StatusCron] 获取锁失败: key=%s, err=%v", lockKey, err)
		return
	}
	if !locked {
		// 其他实例正在执行，跳过
		return
	}
	defer c.unlock(ctx, lockKey)

	// 执行批量更新
	affected, err := c.batchTransition(
		ctx,
		model.StatusPublished,
		model.StatusOngoing,
		"activity_start_time",
		now,
		"活动开始时间到达，自动开始",
	)

	if err != nil {
		logx.Errorf("[StatusCron] Published→Ongoing 失败: %v", err)
		return
	}
	if affected > 0 {
		logx.Infof("[StatusCron] Published→Ongoing 完成: 更新 %d 条记录", affected)
	}
}

// transitionOngoingToFinished 进行中 → 已结束
//
// 触发条件：activity_end_time <= now
// 场景：活动结束时间到达，自动结束活动
func (c *StatusCron) transitionOngoingToFinished(ctx context.Context, now int64) {
	lockKey := lockKeyPrefix + "ongoing_to_finished"

	// 尝试获取分布式锁
	locked, err := c.tryLock(ctx, lockKey)
	if err != nil {
		logx.Errorf("[StatusCron] 获取锁失败: key=%s, err=%v", lockKey, err)
		return
	}
	if !locked {
		return
	}
	defer c.unlock(ctx, lockKey)

	// 执行批量更新
	affected, err := c.batchTransition(
		ctx,
		model.StatusOngoing,
		model.StatusFinished,
		"activity_end_time",
		now,
		"活动结束时间到达，自动结束",
	)

	if err != nil {
		logx.Errorf("[StatusCron] Ongoing→Finished 失败: %v", err)
		return
	}
	if affected > 0 {
		logx.Infof("[StatusCron] Ongoing→Finished 完成: 更新 %d 条记录", affected)
	}
}

// batchTransition 批量状态流转
//
// 流程：
//  1. 查询需要更新的活动 ID
//  2. 分批更新状态
//  3. 记录状态变更日志
func (c *StatusCron) batchTransition(
	ctx context.Context,
	fromStatus, toStatus int8,
	timeField string,
	beforeTime int64,
	remark string,
) (int64, error) {
	var totalAffected int64

	for {
		// 1. 查询需要更新的活动（限制数量）
		var activities []model.Activity
		err := c.db.WithContext(ctx).
			Select("id, version, organizer_id").
			Where("status = ? AND "+timeField+" <= ?", fromStatus, beforeTime).
			Limit(batchSize).
			Find(&activities).Error
		if err != nil {
			return totalAffected, fmt.Errorf("查询活动失败: %w", err)
		}

		if len(activities) == 0 {
			break
		}

		// 2. 逐个更新（事务内更新状态 + 记录日志）
		for _, activity := range activities {
			err := c.transitionOne(ctx, &activity, fromStatus, toStatus, remark)
			if err != nil {
				logx.Errorf("[StatusCron] 更新活动 %d 失败: %v", activity.ID, err)
				continue // 单个失败不影响其他
			}
			totalAffected++
		}

		// 短暂休眠，降低数据库压力
		time.Sleep(50 * time.Millisecond)
	}

	return totalAffected, nil
}

// transitionOne 单个活动状态流转
func (c *StatusCron) transitionOne(
	ctx context.Context,
	activity *model.Activity,
	fromStatus, toStatus int8,
	remark string,
) error {
	return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 更新状态（带乐观锁）
		result := tx.Model(&model.Activity{}).
			Where("id = ? AND version = ? AND status = ?",
				activity.ID, activity.Version, fromStatus).
			Updates(map[string]interface{}{
				"status":  toStatus,
				"version": gorm.Expr("version + 1"),
			})

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			// 可能已被其他进程更新，跳过
			return nil
		}

		// 2. 记录状态变更日志
		log := &model.ActivityStatusLog{
			ActivityID:   activity.ID,
			FromStatus:   fromStatus,
			ToStatus:     toStatus,
			OperatorID:   0, // 系统操作
			OperatorType: model.OperatorTypeSystem,
			Reason:       remark,
		}
		if err := tx.Create(log).Error; err != nil {
			return fmt.Errorf("记录日志失败: %w", err)
		}

		return nil
	})
}

// ==================== 分布式锁 ====================

// tryLock 尝试获取分布式锁
func (c *StatusCron) tryLock(ctx context.Context, key string) (bool, error) {
	// SETNX + EXPIRE 原子操作
	ok, err := c.redis.SetnxExCtx(ctx, key, "1", lockExpireSeconds)
	if err != nil {
		return false, err
	}
	return ok, nil
}

// unlock 释放分布式锁
func (c *StatusCron) unlock(ctx context.Context, key string) {
	_, err := c.redis.DelCtx(ctx, key)
	if err != nil {
		logx.Errorf("[StatusCron] 释放锁失败: key=%s, err=%v", key, err)
	}
}

// ==================== 手动触发（供测试/运维使用） ====================

// RunOnce 手动执行一次状态流转
func (c *StatusCron) RunOnce() {
	logx.Info("[StatusCron] 手动触发状态流转")
	c.execute()
}
