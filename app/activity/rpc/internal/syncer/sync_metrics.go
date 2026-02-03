package syncer

import (
	"sync"
	"sync/atomic"
	"time"
)

// ==================== 同步指标收集器 ====================
//
// 用途：监控跨服务数据同步的健康状态
// 对接：Prometheus / Grafana
//
// 面试亮点：
//   - 展示"可观测性"意识
//   - 最终一致性系统的监控实践
//   - 原子操作保证并发安全

// SyncMetrics 同步指标
type SyncMetrics struct {
	// 同步统计
	TotalSyncs        int64 // 总同步次数
	SuccessfulSyncs   int64 // 成功同步次数
	FailedSyncs       int64 // 失败同步次数
	LastSyncTime      int64 // 最后同步时间戳
	LastSyncDuration  int64 // 最后同步耗时（毫秒）
	LastSyncItemCount int64 // 最后同步的数据条数

	// MQ 消费统计
	MQMessagesReceived  int64 // 收到的 MQ 消息数
	MQMessagesProcessed int64 // 成功处理的 MQ 消息数
	MQMessagesFailed    int64 // 处理失败的 MQ 消息数
	MQMessagesSkipped   int64 // 跳过的 MQ 消息数（幂等去重）

	// 对账统计
	ReconcileRuns       int64 // 对账执行次数
	ReconcileRepairs    int64 // 对账修复次数
	LastReconcileTime   int64 // 最后对账时间
	LastInconsistentPct int64 // 最后不一致率（百分比 * 100）

	// 错误统计
	ConsecutiveFailures int64 // 连续失败次数（用于熔断）
	LastErrorTime       int64 // 最后错误时间
	LastErrorMessage    string
	errorMutex          sync.RWMutex
}

// globalMetrics 全局指标实例
var globalMetrics = &SyncMetrics{}

// GetSyncMetrics 获取全局同步指标
func GetSyncMetrics() *SyncMetrics {
	return globalMetrics
}

// ==================== 同步指标记录方法 ====================

// RecordSyncStart 记录同步开始
func (m *SyncMetrics) RecordSyncStart() {
	atomic.AddInt64(&m.TotalSyncs, 1)
}

// RecordSyncSuccess 记录同步成功
func (m *SyncMetrics) RecordSyncSuccess(duration time.Duration, itemCount int) {
	atomic.AddInt64(&m.SuccessfulSyncs, 1)
	atomic.StoreInt64(&m.LastSyncTime, time.Now().Unix())
	atomic.StoreInt64(&m.LastSyncDuration, duration.Milliseconds())
	atomic.StoreInt64(&m.LastSyncItemCount, int64(itemCount))
	atomic.StoreInt64(&m.ConsecutiveFailures, 0) // 重置连续失败计数
}

// RecordSyncFailure 记录同步失败
func (m *SyncMetrics) RecordSyncFailure(err error) {
	atomic.AddInt64(&m.FailedSyncs, 1)
	atomic.AddInt64(&m.ConsecutiveFailures, 1)
	atomic.StoreInt64(&m.LastErrorTime, time.Now().Unix())

	m.errorMutex.Lock()
	m.LastErrorMessage = err.Error()
	m.errorMutex.Unlock()
}

// ==================== MQ 消息指标记录方法 ====================

// RecordMQMessageReceived 记录收到 MQ 消息
func (m *SyncMetrics) RecordMQMessageReceived() {
	atomic.AddInt64(&m.MQMessagesReceived, 1)
}

// RecordMQMessageProcessed 记录成功处理 MQ 消息
func (m *SyncMetrics) RecordMQMessageProcessed() {
	atomic.AddInt64(&m.MQMessagesProcessed, 1)
	atomic.StoreInt64(&m.ConsecutiveFailures, 0)
}

// RecordMQMessageFailed 记录处理失败的 MQ 消息
func (m *SyncMetrics) RecordMQMessageFailed(err error) {
	atomic.AddInt64(&m.MQMessagesFailed, 1)
	atomic.AddInt64(&m.ConsecutiveFailures, 1)
	atomic.StoreInt64(&m.LastErrorTime, time.Now().Unix())

	m.errorMutex.Lock()
	m.LastErrorMessage = err.Error()
	m.errorMutex.Unlock()
}

// RecordMQMessageSkipped 记录跳过的 MQ 消息（幂等去重）
func (m *SyncMetrics) RecordMQMessageSkipped() {
	atomic.AddInt64(&m.MQMessagesSkipped, 1)
}

// ==================== 对账指标记录方法 ====================

// RecordReconcileResult 记录对账结果
func (m *SyncMetrics) RecordReconcileResult(result *ReconcileResult) {
	atomic.AddInt64(&m.ReconcileRuns, 1)
	atomic.AddInt64(&m.ReconcileRepairs, int64(result.Repaired))
	atomic.StoreInt64(&m.LastReconcileTime, time.Now().Unix())

	// 计算不一致率（百分比 * 100，用整数存储避免浮点精度问题）
	if result.TotalChecked > 0 {
		pct := int64((result.Mismatched + result.MissingInLocal) * 10000 / result.TotalChecked)
		atomic.StoreInt64(&m.LastInconsistentPct, pct)
	}
}

// ==================== 健康检查方法 ====================

// IsHealthy 检查同步服务是否健康
//
// 健康标准：
//   - 连续失败次数 < 5
//   - 最后成功同步时间在 10 分钟内（定时同步周期的 2 倍）
//   - 最后对账不一致率 < 10%
func (m *SyncMetrics) IsHealthy() bool {
	consecutiveFailures := atomic.LoadInt64(&m.ConsecutiveFailures)
	lastSyncTime := atomic.LoadInt64(&m.LastSyncTime)
	lastInconsistentPct := atomic.LoadInt64(&m.LastInconsistentPct)

	// 连续失败次数过多
	if consecutiveFailures >= 5 {
		return false
	}

	// 最后同步时间过久（超过 10 分钟）
	if lastSyncTime > 0 && time.Since(time.Unix(lastSyncTime, 0)) > 10*time.Minute {
		return false
	}

	// 不一致率过高（> 10%）
	if lastInconsistentPct > 1000 { // 1000 = 10.00%
		return false
	}

	return true
}

// GetHealthStatus 获取健康状态详情
func (m *SyncMetrics) GetHealthStatus() map[string]interface{} {
	m.errorMutex.RLock()
	lastError := m.LastErrorMessage
	m.errorMutex.RUnlock()

	return map[string]interface{}{
		"healthy":               m.IsHealthy(),
		"total_syncs":           atomic.LoadInt64(&m.TotalSyncs),
		"successful_syncs":      atomic.LoadInt64(&m.SuccessfulSyncs),
		"failed_syncs":          atomic.LoadInt64(&m.FailedSyncs),
		"consecutive_failures":  atomic.LoadInt64(&m.ConsecutiveFailures),
		"last_sync_time":        time.Unix(atomic.LoadInt64(&m.LastSyncTime), 0).Format(time.RFC3339),
		"last_sync_duration_ms": atomic.LoadInt64(&m.LastSyncDuration),
		"last_sync_items":       atomic.LoadInt64(&m.LastSyncItemCount),
		"mq_received":           atomic.LoadInt64(&m.MQMessagesReceived),
		"mq_processed":          atomic.LoadInt64(&m.MQMessagesProcessed),
		"mq_failed":             atomic.LoadInt64(&m.MQMessagesFailed),
		"mq_skipped":            atomic.LoadInt64(&m.MQMessagesSkipped),
		"reconcile_runs":        atomic.LoadInt64(&m.ReconcileRuns),
		"reconcile_repairs":     atomic.LoadInt64(&m.ReconcileRepairs),
		"last_inconsistent_pct": float64(atomic.LoadInt64(&m.LastInconsistentPct)) / 100,
		"last_error":            lastError,
	}
}

// ==================== Prometheus 格式导出 ====================

// ToPrometheusFormat 导出为 Prometheus 格式
//
// 示例输出：
//
//	activity_tag_sync_total{status="success"} 1234
//	activity_tag_sync_total{status="failed"} 5
//	activity_tag_sync_duration_ms 150
//	activity_tag_mq_messages_total{status="processed"} 5678
//	activity_tag_reconcile_inconsistent_percent 2.5
func (m *SyncMetrics) ToPrometheusFormat() string {
	return "" // 具体实现根据 Prometheus client 库
	// 这里只提供接口定义，实际项目中使用 prometheus/client_golang
}
