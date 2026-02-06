/**
 * @projectName: CampusHub
 * @package: verifyservicelogic
 * @className: ProcessOcrVerifyLogic
 * @author: lijunqi
 * @description: OCR 识别处理逻辑层（供统一 MQ Consumer 调用）
 * @date: 2026-02-06
 * @version: 1.0
 *
 * ==================== 业务说明 ====================
 *
 * 本方法由统一 MQ 消费者（app/mq）在消费 verify:events 事件后调用。
 * 将原 app/user/mq 中的 verify_callback_handler 和 apply_ocr_handler 逻辑下沉到 RPC 层。
 *
 * 处理流程:
 *   1. 参数校验
 *   2. 查询认证记录，校验状态（必须为 OcrPending）
 *   3. 检查是否超时（>10min 直接标记超时）
 *   4. 调用 OCR 识别（主提供商 + 备用故障转移，30s 超时）
 *   5. OCR 成功 → 更新为 WaitConfirm + 回填 OCR 数据
 *   6. OCR 失败 → 更新为 OcrFailed
 *   7. 清除认证缓存
 *
 * 竞态防护:
 *   - OCR 处理期间用户取消: 处理完后再次检查状态，若已变更则跳过更新
 *   - 超时扫描器先行标记超时: 检查状态时发现非 OcrPending，跳过处理
 */

package verifyservicelogic

import (
	"context"
	"time"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/constants"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type ProcessOcrVerifyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewProcessOcrVerifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProcessOcrVerifyLogic {
	return &ProcessOcrVerifyLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ProcessOcrVerify 处理 OCR 识别（供统一 MQ Consumer 调用）
func (l *ProcessOcrVerifyLogic) ProcessOcrVerify(in *pb.ProcessOcrVerifyReq) (*pb.ProcessOcrVerifyResp, error) {
	// ==================== Step 1: 参数校验 ====================
	if in.VerifyId <= 0 || in.UserId <= 0 {
		l.Infof("[ProcessOcr] 无效参数: verifyId=%d, userId=%d", in.VerifyId, in.UserId)
		return &pb.ProcessOcrVerifyResp{
			Success:      false,
			ResultStatus: 0,
			Message:      "无效参数",
		}, nil // 不返回 error，MQ 不需要重试
	}

	// ==================== Step 2: 查询认证记录 + 校验状态 ====================
	verification, err := l.svcCtx.StudentVerificationModel.FindByID(l.ctx, in.VerifyId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			l.Infof("[ProcessOcr] 记录不存在: verifyId=%d", in.VerifyId)
			return &pb.ProcessOcrVerifyResp{
				Success: false,
				Message: "记录不存在",
			}, nil
		}
		l.Errorf("[ProcessOcr] 查询失败: verifyId=%d, err=%v", in.VerifyId, err)
		return nil, errorx.ErrDBError(err) // DB 错误，MQ 触发重试
	}

	// 校验用户ID
	if verification.UserID != in.UserId {
		l.Infof("[ProcessOcr] 用户ID不匹配: expected=%d, got=%d", verification.UserID, in.UserId)
		return &pb.ProcessOcrVerifyResp{
			Success: false,
			Message: "用户ID不匹配",
		}, nil
	}

	// 状态必须为 OcrPending（防止已取消/超时的继续处理）
	if verification.Status != constants.VerifyStatusOcrPending {
		l.Infof("[ProcessOcr] 状态已变更，跳过: verifyId=%d, status=%d(%s)",
			in.VerifyId, verification.Status, constants.GetVerifyStatusName(verification.Status))
		return &pb.ProcessOcrVerifyResp{
			Success:      true,
			ResultStatus: int32(verification.Status),
			Message:      "状态已变更，跳过处理",
		}, nil
	}

	// ==================== Step 3: 检查是否已超时 ====================
	timeoutThreshold := verification.UpdatedAt.Add(
		time.Duration(constants.VerifyOcrTimeoutMinutes) * time.Minute)
	if time.Now().After(timeoutThreshold) {
		l.Infof("[ProcessOcr] OCR已超时: verifyId=%d, updatedAt=%v", in.VerifyId, verification.UpdatedAt)
		updates := map[string]interface{}{
			"operator": constants.VerifyOperatorTimeoutJob,
		}
		if updateErr := l.svcCtx.StudentVerificationModel.UpdateStatus(
			l.ctx, in.VerifyId, constants.VerifyStatusTimeout, updates); updateErr != nil {
			l.Errorf("[ProcessOcr] 标记超时失败: verifyId=%d, err=%v", in.VerifyId, updateErr)
			return nil, errorx.ErrDBError(updateErr)
		}
		return &pb.ProcessOcrVerifyResp{
			Success:      true,
			ResultStatus: int32(constants.VerifyStatusTimeout),
			Message:      "OCR超时",
		}, nil
	}

	// ==================== Step 4: 调用 OCR 识别 ====================
	if l.svcCtx.OcrFactory == nil {
		l.Errorf("[ProcessOcr] OCR工厂未初始化: verifyId=%d", in.VerifyId)
		updates := map[string]interface{}{
			"operator": constants.VerifyOperatorOcrCallback,
		}
		_ = l.svcCtx.StudentVerificationModel.UpdateStatus(
			l.ctx, in.VerifyId, constants.VerifyStatusOcrFailed, updates)
		return &pb.ProcessOcrVerifyResp{
			Success:      true,
			ResultStatus: int32(constants.VerifyStatusOcrFailed),
			Message:      "OCR服务不可用",
		}, nil
	}

	// 30 秒超时进行 OCR 识别
	ocrCtx, ocrCancel := context.WithTimeout(l.ctx, 30*time.Second)
	defer ocrCancel()

	ocrResult, err := l.svcCtx.OcrFactory.Recognize(ocrCtx, in.FrontImageUrl, in.BackImageUrl)

	// ==================== Step 5: 处理 OCR 结果 ====================
	if err != nil {
		l.Errorf("[ProcessOcr] OCR识别失败: verifyId=%d, err=%v", in.VerifyId, err)

		// 再次检查状态（防止 OCR 期间用户取消）
		if l.isStatusChanged(in.VerifyId) {
			l.Infof("[ProcessOcr] OCR期间状态已变更，跳过: verifyId=%d", in.VerifyId)
			return &pb.ProcessOcrVerifyResp{
				Success: true,
				Message: "OCR期间状态已变更",
			}, nil
		}

		// 标记为 OcrFailed
		updates := map[string]interface{}{
			"operator": constants.VerifyOperatorOcrCallback,
		}
		if updateErr := l.svcCtx.StudentVerificationModel.UpdateStatus(
			l.ctx, in.VerifyId, constants.VerifyStatusOcrFailed, updates); updateErr != nil {
			l.Errorf("[ProcessOcr] 标记OCR失败: verifyId=%d, err=%v", in.VerifyId, updateErr)
			return nil, errorx.ErrDBError(updateErr)
		}

		// 清除缓存
		l.deleteVerifyCache(in.UserId)

		return &pb.ProcessOcrVerifyResp{
			Success:      true,
			ResultStatus: int32(constants.VerifyStatusOcrFailed),
			Message:      "OCR识别失败",
		}, nil
	}

	// ==================== Step 6: OCR 成功，更新为 WaitConfirm ====================
	l.Infof("[ProcessOcr] OCR识别成功: verifyId=%d, platform=%s, school=%s, name=%s",
		in.VerifyId, ocrResult.Platform, ocrResult.SchoolName, ocrResult.RealName)

	// 再次检查状态（防止 OCR 期间用户取消）
	if l.isStatusChanged(in.VerifyId) {
		l.Infof("[ProcessOcr] OCR完成但状态已变更，跳过更新: verifyId=%d", in.VerifyId)
		return &pb.ProcessOcrVerifyResp{
			Success: true,
			Message: "OCR完成但状态已变更",
		}, nil
	}

	// 构建 OCR 结果并更新数据库
	ocrData := BuildOcrResultData(&pb.VerifyOcrData{
		RealName:      ocrResult.RealName,
		SchoolName:    ocrResult.SchoolName,
		StudentId:     ocrResult.StudentID,
		Department:    ocrResult.Department,
		AdmissionYear: ocrResult.AdmissionYear,
		OcrPlatform:   ocrResult.Platform,
		OcrConfidence: ocrResult.Confidence,
		OcrRawJson:    ocrResult.RawResponse,
	})

	if updateErr := l.svcCtx.StudentVerificationModel.UpdateOcrResult(
		l.ctx, in.VerifyId, ocrData); updateErr != nil {
		l.Errorf("[ProcessOcr] 更新OCR结果失败: verifyId=%d, err=%v", in.VerifyId, updateErr)
		return nil, errorx.ErrDBError(updateErr)
	}

	// 清除缓存
	l.deleteVerifyCache(in.UserId)

	l.Infof("[ProcessOcr] 处理成功: verifyId=%d, userId=%d → WaitConfirm", in.VerifyId, in.UserId)

	return &pb.ProcessOcrVerifyResp{
		Success:      true,
		ResultStatus: int32(constants.VerifyStatusWaitConfirm),
		Message:      "OCR识别成功",
	}, nil
}

// isStatusChanged 检查认证记录的状态是否已经不是 OcrPending
func (l *ProcessOcrVerifyLogic) isStatusChanged(verifyID int64) bool {
	fresh, err := l.svcCtx.StudentVerificationModel.FindByID(l.ctx, verifyID)
	if err != nil {
		return false // 查询失败，保守假设未变更
	}
	return fresh.Status != constants.VerifyStatusOcrPending
}

// deleteVerifyCache 清除认证缓存
func (l *ProcessOcrVerifyLogic) deleteVerifyCache(userID int64) {
	if err := l.svcCtx.VerifyCache.Delete(l.ctx, userID); err != nil {
		l.Errorf("[ProcessOcr] 清除缓存失败: userId=%d, err=%v", userID, err)
	}
}
