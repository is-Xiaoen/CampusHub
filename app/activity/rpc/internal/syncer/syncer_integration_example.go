package syncer

import (
	"context"

	"activity-platform/app/activity/model"
	"activity-platform/app/user/rpc/client/tagservice"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"
)

// ==================== 服务集成示例 ====================
//
// 本文件展示如何在 ServiceContext 中集成标签同步组件
// 这是一个示例文件，实际使用时应根据项目结构调整
//
// 集成步骤：
//   1. 在 ServiceContext 中添加同步组件字段
//   2. 在 NewServiceContext 中初始化组件
//   3. 在服务启动时启动同步任务
//   4. 在服务停止时优雅关闭

// SyncerComponents 同步组件集合
//
// 用于在 ServiceContext 中管理所有同步相关组件
type SyncerComponents struct {
	// 核心组件
	TagSyncer       *TagSyncer       // 定时同步器
	TagEventHandler *TagEventHandler // MQ 事件处理器
	TagReconciler   *TagReconciler   // 数据对账器

	// 辅助组件
	Alerter Alerter       // 告警器
	Metrics *SyncMetrics  // 指标收集器
}

// SyncerConfig 同步器配置
type SyncerConfig struct {
	// 同步配置
	SyncIntervalMinutes int // 定时同步间隔（分钟），默认 5

	// 对账配置
	ReconcileEnabled bool   // 是否启用对账
	ReconcileHour    int    // 对账执行时间（小时），默认 3（凌晨3点）

	// 告警配置
	AlertWebhookURL   string // 告警 Webhook URL（为空则只记录日志）
	AlertRatePerMin   int    // 每分钟最大告警数，默认 10

	// 分布式锁配置
	DistributedLockEnabled bool // 是否启用分布式锁（多实例部署时启用）
}

// DefaultSyncerConfig 默认配置
func DefaultSyncerConfig() SyncerConfig {
	return SyncerConfig{
		SyncIntervalMinutes:    5,
		ReconcileEnabled:       true,
		ReconcileHour:          3,
		AlertWebhookURL:        "",
		AlertRatePerMin:        10,
		DistributedLockEnabled: false,
	}
}

// NewSyncerComponents 创建同步组件集合
//
// 参数：
//   - db: GORM 数据库连接
//   - rds: Redis 客户端（用于分布式锁）
//   - tagRpc: 用户服务 TagService RPC 客户端
//   - tagCacheModel: 标签缓存 Model
//   - activityTagModel: 活动标签 Model
//   - tagStatsModel: 标签统计 Model
//   - config: 同步器配置
func NewSyncerComponents(
	db *gorm.DB,
	rds *redis.Redis,
	tagRpc tagservice.TagService,
	tagCacheModel *model.TagCacheModel,
	activityTagModel *model.ActivityTagModel,
	tagStatsModel *model.ActivityTagStatsModel,
	config SyncerConfig,
) *SyncerComponents {
	// 1. 创建告警器
	var alerter Alerter
	if config.AlertWebhookURL != "" {
		// 生产环境：Webhook + 日志双通道
		alerter = NewCompositeAlerter(
			NewLogAlerter(),
			NewWebhookAlerter(config.AlertWebhookURL, config.AlertRatePerMin),
		)
	} else {
		// 开发环境：仅日志
		alerter = NewLogAlerter()
	}

	// 2. 创建分布式锁
	var lock DistributedLock
	if config.DistributedLockEnabled && rds != nil {
		lock = NewRedisLock(rds, "lock:activity:tag:reconcile", 30*1000*1000*1000) // 30s
	} else {
		lock = NewNoopLock()
	}

	// 3. 创建 RPC 适配器
	userTagRPC := NewUserTagRPCAdapter(tagRpc)

	// 4. 创建核心组件
	components := &SyncerComponents{
		TagSyncer: NewTagSyncer(
			tagRpc,
			tagCacheModel,
			5*60*1000*1000*1000, // 5 分钟
		),
		TagEventHandler: NewTagEventHandler(
			db,
			tagCacheModel,
			activityTagModel,
			tagStatsModel,
		),
		TagReconciler: NewTagReconcilerWithLock(
			db,
			tagCacheModel,
			userTagRPC,
			alerter,
			lock,
		),
		Alerter: alerter,
		Metrics: GetSyncMetrics(),
	}

	return components
}

// Start 启动所有同步任务
//
// 在服务启动时调用
func (c *SyncerComponents) Start(ctx context.Context) {
	// 1. 启动定时同步
	c.TagSyncer.Start()

	// 2. 启动对账任务
	go c.TagReconciler.StartReconcileJob(ctx)
}

// Stop 停止所有同步任务
//
// 在服务停止时调用
func (c *SyncerComponents) Stop() {
	c.TagSyncer.Stop()
}

// ==================== ServiceContext 集成示例 ====================
//
// 在 app/activity/rpc/internal/svc/service_context.go 中添加：
//
// type ServiceContext struct {
//     Config config.Config
//     // ... 其他字段
//
//     // 同步组件
//     SyncerComponents *syncer.SyncerComponents
// }
//
// func NewServiceContext(c config.Config) *ServiceContext {
//     // ... 初始化其他组件
//
//     // 初始化同步组件
//     syncerComponents := syncer.NewSyncerComponents(
//         db,
//         rds,
//         tagRpc,
//         tagCacheModel,
//         activityTagModel,
//         tagStatsModel,
//         syncer.DefaultSyncerConfig(),
//     )
//
//     return &ServiceContext{
//         Config: c,
//         // ...
//         SyncerComponents: syncerComponents,
//     }
// }

// ==================== MQ 消费者集成示例 ====================
//
// 在 MQ 消费者中使用 TagEventHandler：
//
// func (c *TagSyncConsumer) Consume(ctx context.Context, msg *mq.Message) error {
//     return c.svcCtx.SyncerComponents.TagEventHandler.HandleMessage(ctx, msg.Body)
// }

// ==================== 健康检查集成示例 ====================
//
// 在健康检查接口中使用 SyncMetrics：
//
// func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
//     metrics := syncer.GetSyncMetrics()
//     status := metrics.GetHealthStatus()
//
//     if !metrics.IsHealthy() {
//         w.WriteHeader(http.StatusServiceUnavailable)
//     }
//
//     json.NewEncoder(w).Encode(status)
// }
