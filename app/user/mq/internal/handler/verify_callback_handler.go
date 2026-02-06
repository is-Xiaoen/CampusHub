/**
 * @projectName: CampusHub
 * @package: handler
 * @className: VerifyCallbackHandler
 * @author: lijunqi
 * @description: 学生认证 OCR 处理器（消费 verify:events 事件）
 * @date: 2026-02-06
 * @version: 2.0
 *
 * ==================== 业务说明 ====================
 *
 * 本处理器负责消费来自 User RPC 的认证申请事件，执行 OCR 识别并更新认证状态。
 *
 * 消息来源:
 *   - User RPC 服务在用户提交认证申请后发布事件到 Redis Stream (topic: verify:events)
 *
 * 处理流程:
 *   1. 解析消息 → VerifyApplyEventData
 *   2. 检查当前状态（必须仍为 OcrPending，防止已取消的继续处理）
 *   3. 检查是否超时（超过10分钟直接标记超时）
 *   4. 调用 OCR 识别（主提供商 + 备用提供商故障转移）
 *   5. OCR 成功 → 更新为 WaitConfirm 状态
 *   6. OCR 失败 → 更新为 OcrFailed 状态
 *
 * 错误处理策略:
 *   - JSON 解析失败: 丢弃，不重试
 *   - 参数校验失败: 丢弃，不重试
 *   - 状态已变更（已取消等）: 丢弃，不重试
 *   - OCR 识别失败: 标记为 OcrFailed，不重试
 *   - 数据库错误: 返回错误，触发 Watermill 重试
 *
 * 竞态处理:
 *   - 用户在 OCR 处理中取消: handler 检查 status 时发现非 OcrPending，跳过处理
 *   - 超时扫描器先行标记超时: handler 检查 status 时发现非 OcrPending，跳过处理
 */

package handler

import (
	"context"
	"encoding/json"
	"time"

	"activity-platform/app/user/model"
	"activity-platform/app/user/mq/internal/svc"
	"activity-platform/common/constants"
	"activity-platform/common/messaging"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// NewVerifyCallbackHandler 创建认证 OCR 处理器
//
// 参数:
//   - svcCtx: 服务上下文，包含 DB、Redis、Model、OcrFactory 等依赖
//
// 返回:
//   - Handler: 符合 handler.Handler 签名的处理函数
func NewVerifyCallbackHandler(svcCtx *svc.ServiceContext) Handler {
	return func(ctx context.Context, msg *Message) error {
		logger := logx.WithContext(ctx)
		logger.Infof("[VerifyHandler] 收到消息: id=%s", msg.ID)

		// ==================== Step 1: 解析消息 ====================
		var event messaging.VerifyApplyEventData
		if err := json.Unmarshal([]byte(msg.Data), &event); err != nil {
			logger.Errorf("[VerifyHandler] 解析消息失败: %v, data=%s", err, msg.Data)
			return nil // 格式错误，丢弃不重试
		}

		// ==================== Step 2: 参数校验 ====================
		if event.VerifyID <= 0 || event.UserID <= 0 {
			logger.Infof("[VerifyHandler] [WARN] 无效参数: verifyId=%d, userId=%d",
				event.VerifyID, event.UserID)
			return nil // 无效参数，丢弃
		}

		// ==================== Step 3: 查询当前记录并校验状态 ====================
		verification, err := svcCtx.StudentVerificationModel.FindByID(ctx, event.VerifyID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				logger.Infof("[VerifyHandler] [WARN] 记录不存在: verifyId=%d", event.VerifyID)
				return nil // 记录不存在，丢弃
			}
			logger.Errorf("[VerifyHandler] 查询记录失败: verifyId=%d, err=%v", event.VerifyID, err)
			return err // DB 错误，触发重试
		}

		// 校验用户ID
		if verification.UserID != event.UserID {
			logger.Infof("[VerifyHandler] [WARN] 用户ID不匹配: expected=%d, got=%d",
				verification.UserID, event.UserID)
			return nil // 数据异常，丢弃
		}

		// 状态必须仍为 OcrPending（防止已取消/超时的继续处理）
		if verification.Status != constants.VerifyStatusOcrPending {
			logger.Infof("[VerifyHandler] 状态已变更，跳过处理: verifyId=%d, status=%d(%s)",
				event.VerifyID, verification.Status, constants.GetVerifyStatusName(verification.Status))
			return nil // 状态已变更，丢弃
		}

		// ==================== Step 4: 检查是否已超时 ====================
		timeoutThreshold := verification.UpdatedAt.Add(
			time.Duration(constants.VerifyOcrTimeoutMinutes) * time.Minute)
		if time.Now().After(timeoutThreshold) {
			logger.Infof("[VerifyHandler] OCR 已超时: verifyId=%d, updatedAt=%v",
				event.VerifyID, verification.UpdatedAt)
			updates := map[string]interface{}{
				"operator": constants.VerifyOperatorTimeoutJob,
			}
			if updateErr := svcCtx.StudentVerificationModel.UpdateStatus(
				ctx, event.VerifyID, constants.VerifyStatusTimeout, updates); updateErr != nil {
				logger.Errorf("[VerifyHandler] 标记超时失败: verifyId=%d, err=%v", event.VerifyID, updateErr)
				return updateErr // DB 错误，触发重试
			}
			return nil
		}

		// ==================== Step 5: 调用 OCR 识别 ====================
		if svcCtx.OcrFactory == nil {
			logger.Errorf("[VerifyHandler] OCR 工厂未初始化: verifyId=%d", event.VerifyID)
			updates := map[string]interface{}{
				"operator": constants.VerifyOperatorOcrCallback,
			}
			_ = svcCtx.StudentVerificationModel.UpdateStatus(
				ctx, event.VerifyID, constants.VerifyStatusOcrFailed, updates)
			return nil // OCR 未配置，丢弃
		}

		// 使用 30 秒超时进行 OCR 识别
		ocrCtx, ocrCancel := context.WithTimeout(ctx, 30*time.Second)
		defer ocrCancel()

		ocrResult, err := svcCtx.OcrFactory.Recognize(ocrCtx, event.FrontImageURL, event.BackImageURL)

		// ==================== Step 6: 处理 OCR 结果 ====================
		if err != nil {
			logger.Errorf("[VerifyHandler] OCR 识别失败: verifyId=%d, err=%v", event.VerifyID, err)

			// 再次检查状态（防止 OCR 期间用户取消）
			if isStatusChanged(ctx, svcCtx, event.VerifyID) {
				logger.Infof("[VerifyHandler] OCR 期间状态已变更，跳过: verifyId=%d", event.VerifyID)
				return nil
			}

			// 标记为 OcrFailed
			updates := map[string]interface{}{
				"operator": constants.VerifyOperatorOcrCallback,
			}
			if updateErr := svcCtx.StudentVerificationModel.UpdateStatus(
				ctx, event.VerifyID, constants.VerifyStatusOcrFailed, updates); updateErr != nil {
				logger.Errorf("[VerifyHandler] 标记 OCR 失败: verifyId=%d, err=%v", event.VerifyID, updateErr)
				return updateErr // DB 错误，触发重试
			}
			return nil
		}

		// ==================== Step 7: OCR 成功，更新为 WaitConfirm ====================
		logger.Infof("[VerifyHandler] OCR 识别成功: verifyId=%d, platform=%s, school=%s, name=%s",
			event.VerifyID, ocrResult.Platform, ocrResult.SchoolName, ocrResult.RealName)

		// 再次检查状态（防止 OCR 期间用户取消）
		if isStatusChanged(ctx, svcCtx, event.VerifyID) {
			logger.Infof("[VerifyHandler] OCR 完成但状态已变更，跳过更新: verifyId=%d", event.VerifyID)
			return nil
		}

		// 构建 OCR 结果数据并更新到数据库（状态 → WaitConfirm）
		ocrData := &model.OcrResultData{
			RealName:      ocrResult.RealName,
			SchoolName:    ocrResult.SchoolName,
			StudentID:     ocrResult.StudentID,
			Department:    ocrResult.Department,
			AdmissionYear: ocrResult.AdmissionYear,
			OcrPlatform:   ocrResult.Platform,
			OcrConfidence: ocrResult.Confidence,
			OcrRawJSON:    ocrResult.RawResponse,
		}

		if updateErr := svcCtx.StudentVerificationModel.UpdateOcrResult(
			ctx, event.VerifyID, ocrData); updateErr != nil {
			logger.Errorf("[VerifyHandler] 更新 OCR 结果失败: verifyId=%d, err=%v", event.VerifyID, updateErr)
			return updateErr // DB 错误，触发重试
		}

		logger.Infof("[VerifyHandler] 处理成功: verifyId=%d, userId=%d → WaitConfirm",
			event.VerifyID, event.UserID)
		return nil
	}
}

// isStatusChanged 检查认证记录的状态是否已经不是 OcrPending
// 用于防止 OCR 处理期间用户取消导致的竞态
func isStatusChanged(ctx context.Context, svcCtx *svc.ServiceContext, verifyID int64) bool {
	fresh, err := svcCtx.StudentVerificationModel.FindByID(ctx, verifyID)
	if err != nil {
		return false // 查询失败，保守假设未变更
	}
	return fresh.Status != constants.VerifyStatusOcrPending
}
