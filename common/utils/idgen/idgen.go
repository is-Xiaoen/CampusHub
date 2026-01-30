/**
 * @projectName: CampusHub
 * @package: idgen
 * @className: idgen
 * @author: lijunqi
 * @description: 幂等来源ID生成工具
 * @date: 2026-01-30
 * @version: 1.0
 */

package idgen

import (
	"fmt"
	"time"
)

// ==================== 说明 ====================
// 主键ID生成策略：
//   - 推荐使用 MySQL 自增ID（GORM autoIncrement）
//   - 简单、可靠、无分布式冲突风险
//
// 本文件只提供 SourceID（幂等键）生成工具
// SourceID 用于保证业务操作的幂等性
// ==================== 幂等来源ID生成器 ====================

// SourceType 来源类型常量
// 用于生成幂等键的前缀
const (
	// SourceTypeInit 初始化信用分
	SourceTypeInit = "init"
	// SourceTypeCheckin 签到
	SourceTypeCheckin = "checkin"
	// SourceTypeCancel 取消报名
	SourceTypeCancel = "cancel"
	// SourceTypeNoShow 爽约
	SourceTypeNoShow = "noshow"
	// SourceTypeHostSuccess 活动圆满举办
	SourceTypeHostSuccess = "host_success"
	// SourceTypeHostDelete 删除活动
	SourceTypeHostDelete = "host_delete"
	// SourceTypeAdminAdjust 管理员调整
	SourceTypeAdminAdjust = "admin_adjust"
)

// GenSourceID 生成幂等来源ID
// 格式: {sourceType}:{bizID}
// 示例: init:10001, checkin:activity_123:user_456
func GenSourceID(sourceType string, bizID int64) string {
	return fmt.Sprintf("%s:%d", sourceType, bizID)
}

// GenSourceIDWithSub 生成带子ID的幂等来源ID
// 格式: {sourceType}:{bizID}:{subID}
// 示例: checkin:123:456 (活动ID:用户ID)
func GenSourceIDWithSub(sourceType string, bizID, subID int64) string {
	return fmt.Sprintf("%s:%d:%d", sourceType, bizID, subID)
}

// GenSourceIDMulti 生成多部分的幂等来源ID
// 格式: {sourceType}:{parts[0]}:{parts[1]}:...
// 示例: cancel:activity_123:user_456:timestamp
func GenSourceIDMulti(sourceType string, parts ...interface{}) string {
	result := sourceType
	for _, part := range parts {
		result = fmt.Sprintf("%s:%v", result, part)
	}
	return result
}

// ==================== 常用场景快捷方法 ====================

// GenInitSourceID 生成初始化信用分的来源ID
// 格式: init:{userID}
func GenInitSourceID(userID int64) string {
	return GenSourceID(SourceTypeInit, userID)
}

// GenCheckinSourceID 生成签到的来源ID
// 格式: checkin:{activityID}:{userID}
func GenCheckinSourceID(activityID, userID int64) string {
	return GenSourceIDWithSub(SourceTypeCheckin, activityID, userID)
}

// GenCancelSourceID 生成取消报名的来源ID
// 格式: cancel:{activityID}:{userID}
func GenCancelSourceID(activityID, userID int64) string {
	return GenSourceIDWithSub(SourceTypeCancel, activityID, userID)
}

// GenNoShowSourceID 生成爽约的来源ID
// 格式: noshow:{activityID}:{userID}
func GenNoShowSourceID(activityID, userID int64) string {
	return GenSourceIDWithSub(SourceTypeNoShow, activityID, userID)
}

// GenHostSuccessSourceID 生成活动圆满举办的来源ID
// 格式: host_success:{activityID}
func GenHostSuccessSourceID(activityID int64) string {
	return GenSourceID(SourceTypeHostSuccess, activityID)
}

// GenHostDeleteSourceID 生成删除活动的来源ID
// 格式: host_delete:{activityID}
func GenHostDeleteSourceID(activityID int64) string {
	return GenSourceID(SourceTypeHostDelete, activityID)
}

// GenAdminAdjustSourceID 生成管理员调整的来源ID
// 格式: admin_adjust:{userID}:{timestamp}
func GenAdminAdjustSourceID(userID int64) string {
	return GenSourceIDWithSub(SourceTypeAdminAdjust, userID, time.Now().UnixMilli())
}
