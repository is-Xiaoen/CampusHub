/**
 * @projectName: CampusHub
 * @package: handler
 * @className: CreditChangeHandler
 * @author: lijunqi
 * @description: 信用分变更消息处理器
 * @date: 2026-01-30
 * @version: 1.0
 */

package handler

import (
	"context"
	"encoding/json"

	"activity-platform/app/user/mq/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// CreditChangeData 信用分变更消息数据
// [待确认] 需要和 Activity 服务的同学确认字段
type CreditChangeData struct {
	// UserID 用户ID
	UserID int64 `json:"user_id"`

	// ChangeType 变更类型
	// 参考 common/constants/credit.go 中的定义
	ChangeType int32 `json:"change_type"`

	// SourceID 来源ID（用于幂等）
	// 如: activity_123_checkin, activity_456_absent
	SourceID string `json:"source_id"`

	// Reason 变更原因
	Reason string `json:"reason"`

	// Delta 自定义变更值（可选，仅管理员调整时使用）
	Delta int32 `json:"delta,omitempty"`
}

// NewCreditChangeHandler 创建信用分变更处理器
func NewCreditChangeHandler(svcCtx *svc.ServiceContext) Handler {
	return func(ctx context.Context, msg *Message) error {
		logx.WithContext(ctx).Infof("[CreditChangeHandler] 收到消息: id=%s", msg.ID)

		// 1. 解析消息
		var data CreditChangeData
		if err := json.Unmarshal([]byte(msg.Data), &data); err != nil {
			logx.WithContext(ctx).Errorf("[CreditChangeHandler] 解析消息失败: %v, data=%s",
				err, msg.Data)
			// 解析失败不重试，直接丢弃
			return nil
		}

		// 2. 参数校验
		if data.UserID <= 0 {
			logx.WithContext(ctx).Errorf("[CreditChangeHandler] 无效的用户ID: %d", data.UserID)
			return nil
		}

		// 3. 调用业务逻辑处理信用分变更
		// TODO: 实现具体业务逻辑
		// 可以调用 RPC 或直接操作 Model
		//
		// 示例:
		// credit, err := svcCtx.UserCreditModel.FindByUserID(ctx, data.UserID)
		// if err != nil {
		//     return err // 返回错误会触发重试（取决于队友的实现）
		// }
		// ... 更新信用分逻辑 ...
		_ = svcCtx // 避免未使用警告

		logx.WithContext(ctx).Infof("[CreditChangeHandler] 处理完成: userId=%d, changeType=%d, sourceId=%s",
			data.UserID, data.ChangeType, data.SourceID)

		return nil
	}
}
