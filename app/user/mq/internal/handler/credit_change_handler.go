/**
 * @projectName: CampusHub
 * @package: handler
 * @className: CreditChangeHandler
 * @author: lijunqi
 * @description: 信用分变更消息处理器
 * @date: 2026-01-30
 * @version: 1.0
 *
 * ==================== 业务说明 ====================
 *
 * 本处理器负责消费来自 Activity 服务的信用事件消息，根据事件类型更新用户信用分。
 *
 * 消息来源:
 *   - Activity RPC 服务在以下场景发布事件到 Redis Stream (topic: credit:events)
 *
 * 支持的事件类型及分值变动:
 *   | 事件类型      | 场景               | 分值变动 |
 *   |--------------|-------------------|---------|
 *   | checkin      | 用户签到成功        | +2      |
 *   | cancel_early | 提前24h取消报名     | 0       |
 *   | cancel_late  | 临期取消报名(<24h)  | -5      |
 *   | noshow       | 活动爽约未签到      | -10     |
 *   | host_success | 组织者成功举办活动   | +5      |
 *   | host_delete  | 删除已有报名的活动   | -10     |
 *
 * 幂等性保证:
 *   - 使用 SourceID (格式: {eventType}:{activityId}:{userId}) 作为唯一键
 *   - 数据库 credit_logs 表有 uk_source_id 唯一索引
 *   - 重复消息会触发唯一索引冲突，被识别为幂等并跳过
 *
 * 错误处理策略:
 *   - JSON 解析失败: 丢弃消息，不重试（消息格式错误无法恢复）
 *   - 参数校验失败: 丢弃消息，不重试（无效数据无法处理）
 *   - 用户不存在: 丢弃消息，不重试（用户可能未注册）
 *   - 唯一索引冲突: 视为成功，不重试（幂等保护）
 *   - 数据库错误: 返回错误，触发 Watermill 重试机制
 *
 * 缓存一致性:
 *   - 信用分更新成功后，主动删除 Redis 缓存 (key: user:credit:{userId})
 *   - 采用 Cache-Aside 模式，下次查询时重新加载
 */

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"activity-platform/app/user/model"
	"activity-platform/app/user/mq/internal/svc"
	"activity-platform/common/constants"
	"activity-platform/common/errorx"
	"activity-platform/common/messaging"

	"github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// ==================== 事件类型映射表 ====================
// 消息数据结构 CreditEventData 定义在 common/messaging/credit_event.go
// 事件类型常量 CreditEvent* 定义在 common/messaging/credit_event.go

// eventTypeToChangeType 事件类型 -> 信用变更类型（对应 constants.CreditChangeType*）
// 用于数据库记录和分值计算
var eventTypeToChangeType = map[string]int32{
	messaging.CreditEventCheckin:     constants.CreditChangeTypeCheckin,     // 签到成功 -> +2
	messaging.CreditEventCancelEarly: constants.CreditChangeTypeCancelEarly, // 提前取消 -> 0（无责）
	messaging.CreditEventCancelLate:  constants.CreditChangeTypeCancelLate,  // 临期取消 -> -5
	messaging.CreditEventNoShow:      constants.CreditChangeTypeNoShow,      // 爽约 -> -10
	messaging.CreditEventHostSuccess: constants.CreditChangeTypeHostSuccess, // 成功举办 -> +5
	messaging.CreditEventHostDelete:  constants.CreditChangeTypeHostDelete,  // 删除活动 -> -10
}

// eventTypeToReason 事件类型 -> 变更原因描述
// 用于 credit_logs 表的 reason 字段，便于用户查看变更历史
var eventTypeToReason = map[string]string{
	messaging.CreditEventCheckin:     "活动签到成功",
	messaging.CreditEventCancelEarly: "提前取消活动报名",
	messaging.CreditEventCancelLate:  "临期取消活动报名（<24h）",
	messaging.CreditEventNoShow:      "活动爽约未签到",
	messaging.CreditEventHostSuccess: "成功举办活动",
	messaging.CreditEventHostDelete:  "删除已有报名的活动",
}

// ==================== 处理器入口 ====================

// NewCreditChangeHandler 创建信用分变更处理器
//
// 参数:
//   - svcCtx: 服务上下文，包含 DB、Redis、Model 等依赖
//
// 返回:
//   - Handler: 符合 handler.Handler 签名的处理函数
//
// 处理流程:
//  1. 解析 JSON 消息 -> CreditEventData
//  2. 参数校验（UserID、ActivityID）
//  3. 映射事件类型 -> 变更类型 + 分值变动
//  4. 构造幂等键 SourceID
//  5. 执行信用分更新（事务：插日志 + 更新分数）
//  6. 删除 Redis 缓存
func NewCreditChangeHandler(svcCtx *svc.ServiceContext) Handler {
	return func(ctx context.Context, msg *Message) error {
		logger := logx.WithContext(ctx)
		logger.Infof("[CreditChangeHandler] 收到消息: id=%s", msg.ID)

		// ==================== Step 1: 解析消息 ====================
		var event messaging.CreditEventData
		if err := json.Unmarshal([]byte(msg.Data), &event); err != nil {
			// 解析失败说明消息格式错误，无法恢复，直接丢弃不重试
			logger.Errorf("[CreditChangeHandler] 解析消息失败: %v, data=%s", err, msg.Data)
			return nil
		}

		// ==================== Step 2: 参数校验 ====================
		// 无效参数无法处理，丢弃不重试
		if event.UserID <= 0 {
			logger.Infof("[CreditChangeHandler] [WARN] 无效的用户ID: %d, msgId=%s", event.UserID, msg.ID)
			return nil
		}
		if event.ActivityID <= 0 {
			logger.Infof("[CreditChangeHandler] [WARN] 无效的活动ID: %d, userId=%d, msgId=%s", event.ActivityID, event.UserID, msg.ID)
			return nil
		}

		// ==================== Step 3: 映射事件类型 ====================
		changeType, ok := eventTypeToChangeType[event.Type]
		if !ok {
			// 未知事件类型，可能是版本不兼容，记录日志后丢弃
			logger.Infof("[CreditChangeHandler] [WARN] 未知事件类型: type=%s, userId=%d, activityId=%d, msgId=%s",
				event.Type, event.UserID, event.ActivityID, msg.ID)
			return nil
		}

		// ==================== Step 4: 计算分值变动 ====================
		delta := constants.GetCreditDelta(changeType, 0)

		// 特殊处理: cancel_early（提前取消）分值为 0，但仍需记录日志
		// 其他 delta=0 的情况直接跳过
		if delta == 0 && event.Type != messaging.CreditEventCancelEarly {
			logger.Infof("[CreditChangeHandler] 分值无变动，跳过: type=%s, userId=%d", event.Type, event.UserID)
			return nil
		}

		// ==================== Step 5: 构造幂等键 ====================
		// SourceID 格式: {eventType}:{activityId}:{userId}
		// 同一用户对同一活动的同一事件类型只会处理一次
		sourceID := fmt.Sprintf("%s:%d:%d", event.Type, event.ActivityID, event.UserID)

		// ==================== Step 6: 获取变更原因 ====================
		reason := eventTypeToReason[event.Type]
		if reason == "" {
			reason = fmt.Sprintf("活动事件: %s", event.Type)
		}

		// ==================== Step 7: 执行信用分更新 ====================
		err := processCreditChange(ctx, svcCtx, &creditChangeParams{
			UserID:     event.UserID,
			ChangeType: changeType,
			SourceID:   sourceID,
			Reason:     reason,
			Delta:      delta,
		})

		if err != nil {
			// 唯一索引冲突 = 重复消息，视为成功（幂等保护）
			if isDuplicateKeyError(err) {
				logger.Infof("[CreditChangeHandler] 幂等拦截: sourceId=%s", sourceID)
				return nil
			}
			// 其他数据库错误，返回 err 触发 Watermill 重试
			logger.Errorf("[CreditChangeHandler] 处理失败: userId=%d, type=%s, err=%v",
				event.UserID, event.Type, err)
			return err
		}

		// ==================== Step 8: 删除缓存 ====================
		// 采用 Cache-Aside 模式：更新 DB 后删除缓存，下次查询时重新加载
		if err := svcCtx.CreditCache.Delete(ctx, event.UserID); err != nil {
			// 缓存删除失败不影响主流程，只记录日志
			// 最坏情况：用户在缓存 TTL 内看到旧数据
			logger.Errorf("[CreditChangeHandler] 删除缓存失败: userId=%d, err=%v", event.UserID, err)
		}

		logger.Infof("[CreditChangeHandler] 处理成功: userId=%d, type=%s, delta=%d, sourceId=%s",
			event.UserID, event.Type, delta, sourceID)

		return nil
	}
}

// ==================== 内部数据结构 ====================

// creditChangeParams 信用分变更参数（内部使用）
type creditChangeParams struct {
	UserID     int64  // 用户ID
	ChangeType int32  // 变更类型（对应 constants.CreditChangeType*）
	SourceID   string // 幂等键，格式: {eventType}:{activityId}:{userId}
	Reason     string // 变更原因描述
	Delta      int    // 分值变动（正数加分，负数扣分）
}

// ==================== 核心业务逻辑 ====================

// processCreditChange 执行信用分变更
//
// 核心逻辑: 先插日志（幂等校验），再更新分数（事务保证）
//
// 参数:
//   - ctx: 上下文（带 trace_id）
//   - svcCtx: 服务上下文
//   - params: 变更参数
//
// 返回:
//   - error: nil 表示成功，唯一索引冲突也返回 error（由调用方判断幂等）
//
// 事务流程:
//  1. 查询当前信用分
//  2. 计算新分数（限制在 0-100 范围）
//  3. 开启事务:
//     3.1 插入 credit_logs 记录（利用 uk_source_id 唯一索引保证幂等）
//     3.2 更新 user_credits 表的 score 和 level
//  4. 事务提交
//
// 幂等性:
//   - credit_logs.source_id 有唯一索引
//   - 重复插入会触发 MySQL Error 1062 (Duplicate entry)
//   - 调用方通过 isDuplicateKeyError() 判断是否为幂等拦截
func processCreditChange(ctx context.Context, svcCtx *svc.ServiceContext, params *creditChangeParams) error {
	logger := logx.WithContext(ctx)

	// ==================== Step 1: 查询当前信用记录 ====================
	credit, err := svcCtx.UserCreditModel.FindByUserID(ctx, params.UserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 用户信用记录不存在，可能是未注册用户，不重试
			logger.Infof("[CreditChange] [WARN] 信用记录不存在: userId=%d, sourceId=%s", params.UserID, params.SourceID)
			return nil
		}
		// 数据库查询错误，返回触发重试
		logger.Errorf("[CreditChange] 查询信用记录失败: userId=%d, sourceId=%s, err=%v", params.UserID, params.SourceID, err)
		return errorx.Wrap(errorx.CodeDBError, err)
	}

	// ==================== Step 2: 计算新分数 ====================
	beforeScore := credit.Score
	afterScore := beforeScore + params.Delta

	// 分数边界限制: [0, 100]
	if afterScore < constants.CreditScoreMin {
		afterScore = constants.CreditScoreMin
	}
	if afterScore > constants.CreditScoreMax {
		afterScore = constants.CreditScoreMax
	}

	// 实际变动值（可能因边界限制与 params.Delta 不同）
	actualDelta := afterScore - beforeScore

	// 根据新分数计算信用等级
	// Level 0: 黑名单(<60), Level 1: 风险(60-70), Level 2: 良好(70-90)
	// Level 3: 优秀(90-95), Level 4: 社区之星(>=95)
	newLevel := constants.CalculateCreditLevel(afterScore)

	// 确定变更方向（用于日志记录）
	changeType := model.CreditChangeTypeAdd
	if actualDelta < 0 {
		changeType = model.CreditChangeTypeDeduct
	}

	// ==================== Step 3: 事务更新 ====================
	return svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Step 3.1: 先插入变更日志（幂等校验点）
		// 如果 source_id 已存在，INSERT 会失败，触发唯一索引冲突
		creditLog := &model.CreditLog{
			UserID:      params.UserID,
			ChangeType:  changeType,
			SourceID:    params.SourceID, // 幂等键
			BeforeScore: beforeScore,
			AfterScore:  afterScore,
			Delta:       actualDelta,
			Reason:      params.Reason,
		}
		if err := tx.Create(creditLog).Error; err != nil {
			// 唯一索引冲突或其他错误，都返回让调用方处理
			return err
		}

		// Step 3.2: 更新信用分和等级
		return tx.Model(&model.UserCredit{}).
			Where("user_id = ?", params.UserID).
			Updates(map[string]interface{}{
				"score": afterScore,
				"level": newLevel,
			}).Error
	})
}

// ==================== 辅助函数 ====================

// isDuplicateKeyError 判断是否为 MySQL 唯一索引冲突错误
//
// 用于识别幂等场景：同一 source_id 的消息重复消费
//
// 参数:
//   - err: 数据库操作返回的错误
//
// 返回:
//   - bool: true 表示是唯一索引冲突（幂等拦截），false 表示其他错误
//
// 判断依据:
//   - MySQL Error 1062: Duplicate entry '...' for key '...'
//   - 错误信息包含 "Duplicate entry" 或 "duplicate key"
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	// 优先判断 MySQL 错误码（更精确）
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		return mysqlErr.Number == 1062 // ER_DUP_ENTRY
	}
	// 兜底：字符串匹配（兼容其他 ORM 封装）
	return strings.Contains(err.Error(), "Duplicate entry") ||
		strings.Contains(err.Error(), "duplicate key")
}
