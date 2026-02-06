/**
 * @projectName: CampusHub
 * @package: handler
 * @className: TimeoutScanner
 * @author: lijunqi
 * @description: OCR 超时扫描器，定期检测并标记超时的认证记录
 * @date: 2026-02-06
 * @version: 1.0
 *
 * ==================== 业务说明 ====================
 *
 * 本模块作为安全兜底机制，定期扫描数据库中处于 OcrPending 状态且已超时的记录，
 * 将其标记为 Timeout 状态。
 *
 * 工作原理:
 *   - 每分钟执行一次扫描（可配置）
 *   - 查询 status=OcrPending 且 updated_at < (now - 10分钟) 的记录
 *   - 逐条更新为 Timeout 状态
 *
 * 为什么需要扫描器（不只依赖 MQ Handler 的超时检测）:
 *   1. MQ 消息可能丢失或延迟
 *   2. MQ Consumer 可能在处理过程中崩溃
 *   3. Redis Stream 可能出现异常
 *   4. 作为双重保险，确保不会有记录永远卡在 OcrPending 状态
 */

package handler

import (
	"context"
	"time"

	"activity-platform/app/user/mq/internal/svc"
	"activity-platform/common/constants"

	"github.com/zeromicro/go-zero/core/logx"
)

// TimeoutScanner OCR 超时扫描器
type TimeoutScanner struct {
	svcCtx   *svc.ServiceContext
	interval time.Duration // 扫描间隔
	stopCh   chan struct{} // 停止信号
}

// NewTimeoutScanner 创建超时扫描器
//
// 参数:
//   - svcCtx: 服务上下文
//   - interval: 扫描间隔时间（建议 1 分钟）
func NewTimeoutScanner(svcCtx *svc.ServiceContext, interval time.Duration) *TimeoutScanner {
	return &TimeoutScanner{
		svcCtx:   svcCtx,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start 启动超时扫描器（非阻塞，在后台 goroutine 运行）
func (s *TimeoutScanner) Start() {
	go s.run()
	logx.Infof("[TimeoutScanner] 启动成功，扫描间隔: %v，超时阈值: %d 分钟",
		s.interval, constants.VerifyOcrTimeoutMinutes)
}

// Stop 停止超时扫描器
func (s *TimeoutScanner) Stop() {
	close(s.stopCh)
	logx.Info("[TimeoutScanner] 已停止")
}

// run 扫描主循环
func (s *TimeoutScanner) run() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.scan()
		}
	}
}

// scan 执行一次超时扫描
func (s *TimeoutScanner) scan() {
	ctx := context.Background()
	logger := logx.WithContext(ctx)

	// 查询超时的 OcrPending 记录
	records, err := s.svcCtx.StudentVerificationModel.FindTimeoutRecords(
		ctx, constants.VerifyOcrTimeoutMinutes)
	if err != nil {
		logger.Errorf("[TimeoutScanner] 查询超时记录失败: err=%v", err)
		return
	}

	if len(records) == 0 {
		return // 没有超时记录，跳过
	}

	logger.Infof("[TimeoutScanner] 发现 %d 条超时记录", len(records))

	// 逐条更新为 Timeout 状态
	for _, record := range records {
		updates := map[string]interface{}{
			"operator": constants.VerifyOperatorTimeoutJob,
		}
		if err := s.svcCtx.StudentVerificationModel.UpdateStatus(
			ctx, record.ID, constants.VerifyStatusTimeout, updates); err != nil {
			logger.Errorf("[TimeoutScanner] 标记超时失败: verifyId=%d, userId=%d, err=%v",
				record.ID, record.UserID, err)
			continue
		}
		logger.Infof("[TimeoutScanner] 已标记超时: verifyId=%d, userId=%d, createdAt=%v",
			record.ID, record.UserID, record.CreatedAt)
	}
}
