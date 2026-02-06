package cron

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/internal/mq"
	"activity-platform/common/messaging"

	"github.com/google/uuid"
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
	msgProducer    *mq.Producer // 消息发布器（可为 nil）

	intervalSeconds int           // 执行间隔（秒）
	stopChan        chan struct{} // 停止信号
	running         atomic.Bool   // 运行状态（原子操作，并发安全）
	stopOnce        sync.Once     // 保证 close(stopChan) 只执行一次
	ownerID         string        // 分布式锁 owner 标识（防止误删他人锁）
}

// NewStatusCron 创建状态定时任务
func NewStatusCron(
	rds *redis.Redis,
	db *gorm.DB,
	activityModel *model.ActivityModel,
	statusLogModel *model.ActivityStatusLogModel,
	msgProducer *mq.Producer,
) *StatusCron {
	return &StatusCron{
		redis:           rds,
		db:              db,
		activityModel:   activityModel,
		statusLogModel:  statusLogModel,
		msgProducer:     msgProducer,
		intervalSeconds: defaultIntervalSeconds,
		stopChan:        make(chan struct{}),
		ownerID:         uuid.New().String(), // 唯一标识，用于分布式锁 owner 校验
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
	// CAS 操作：只有从 false → true 时才启动，天然防重入
	if !c.running.CompareAndSwap(false, true) {
		logx.Info("[StatusCron] 定时任务已在运行中，跳过重复启动")
		return
	}

	logx.Infof("[StatusCron] 启动状态自动流转定时任务，执行间隔: %d 秒, owner: %s", c.intervalSeconds, c.ownerID)

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
	if !c.running.Load() {
		return
	}
	// sync.Once 保证 close(stopChan) 只执行一次，防止 double close panic
	c.stopOnce.Do(func() {
		close(c.stopChan)
	})
	c.running.Store(false)
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
	act *model.Activity,
	fromStatus, toStatus int8,
	remark string,
) error {
	err := c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 更新状态（带乐观锁）
		result := tx.Model(&model.Activity{}).
			Where("id = ? AND version = ? AND status = ?",
				act.ID, act.Version, fromStatus).
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
			ActivityID:   act.ID,
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
	if err != nil {
		return err
	}

	// 3. 活动结束时异步处理信用事件（Ongoing→Finished）
	if fromStatus == model.StatusOngoing && toStatus == model.StatusFinished {
		go c.processFinishedCreditEvents(act.ID, act.OrganizerID)
	}

	return nil
}

// processFinishedCreditEvents 处理活动结束后的信用事件
// - 发布组织者成功举办事件（host_success）
// - 查询未签到用户，发布爽约事件（noshow）
func (c *StatusCron) processFinishedCreditEvents(activityID, organizerID uint64) {
	defer func() {
		if r := recover(); r != nil {
			logx.Errorf("[StatusCron] processFinishedCreditEvents panic: activityId=%d, err=%v", activityID, r)
		}
	}()

	ctx := context.Background()

	// 1. 发布组织者成功举办事件
	c.msgProducer.PublishCreditEvent(ctx, messaging.CreditEventHostSuccess, int64(activityID), int64(organizerID))

	// 2. 查询已报名但未签到的用户 → 发布 noshow 事件
	noshowUsers, err := c.findNoshowUsers(ctx, activityID)
	if err != nil {
		logx.Errorf("[StatusCron] 查询未签到用户失败: activityId=%d, err=%v", activityID, err)
		return
	}
	for _, userID := range noshowUsers {
		c.msgProducer.PublishCreditEvent(ctx, messaging.CreditEventNoShow, int64(activityID), int64(userID))
	}

	if len(noshowUsers) > 0 {
		logx.Infof("[StatusCron] 活动结束信用处理: activityId=%d, noshowCount=%d", activityID, len(noshowUsers))
	}
}

// findNoshowUsers 查询已报名但未签到的用户
// SQL: SELECT r.user_id FROM activity_registrations r
//
//	LEFT JOIN activity_tickets t ON t.registration_id = r.id
//	WHERE r.activity_id = ? AND r.status = 'success'
//	AND (t.id IS NULL OR t.status != 'used')
func (c *StatusCron) findNoshowUsers(ctx context.Context, activityID uint64) ([]uint64, error) {
	var userIDs []uint64
	err := c.db.WithContext(ctx).
		Table("activity_registrations r").
		Select("r.user_id").
		Joins("LEFT JOIN activity_tickets t ON t.registration_id = r.id").
		Where("r.activity_id = ? AND r.status = ?", activityID, model.RegistrationStatusSuccess).
		Where("t.id IS NULL OR t.status != ?", model.TicketStatusUsed).
		Pluck("r.user_id", &userIDs).Error
	return userIDs, err
}

// ==================== 分布式锁 ====================

// unlockScript Lua 脚本：只有 owner 匹配时才删除锁
// 防止锁过期后误删其他实例持有的锁
//
// 原理：
//
//	KEYS[1] = 锁的 key
//	ARGV[1] = 当前实例的 ownerID
//	如果 GET(key) == ownerID，则 DEL(key) 返回 1
//	否则返回 0（说明锁已被其他实例持有）
const unlockScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end
`

// tryLock 尝试获取分布式锁（带 owner 标识）
func (c *StatusCron) tryLock(ctx context.Context, key string) (bool, error) {
	// SETNX + EXPIRE 原子操作，value 存入 ownerID
	ok, err := c.redis.SetnxExCtx(ctx, key, c.ownerID, lockExpireSeconds)
	if err != nil {
		return false, err
	}
	return ok, nil
}

// unlock 释放分布式锁（仅 owner 匹配时才删除）
func (c *StatusCron) unlock(ctx context.Context, key string) {
	result, err := c.redis.EvalCtx(ctx, unlockScript, []string{key}, c.ownerID)
	if err != nil {
		logx.Errorf("[StatusCron] 释放锁失败: key=%s, err=%v", key, err)
		return
	}
	// result == 0 表示锁已被其他实例持有（过期后被抢占），这是正常现象
	if fmt.Sprintf("%v", result) == "0" {
		logx.Infof("[StatusCron] 锁已被其他实例持有，跳过释放: key=%s", key)
	}
}

// ==================== 手动触发（供测试/运维使用） ====================

// RunOnce 手动执行一次状态流转
func (c *StatusCron) RunOnce() {
	logx.Info("[StatusCron] 手动触发状态流转")
	c.execute()
}
