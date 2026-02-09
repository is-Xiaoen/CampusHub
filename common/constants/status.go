// ============================================================================
// 状态常量定义
// ============================================================================
//
// 说明：
//   定义项目中所有状态码常量，避免魔法数字
//   各模块可在此基础上扩展和测试全部CI
//
// ============================================================================

package constants

// ==================== 用户状态 ====================

const (
	UserStatusNormal   = 1 // 正常
	UserStatusDisabled = 2 // 禁用
	UserStatusDeleted  = 3 // 注销
)

// UserStatusMap 用户状态映射（用于展示）
var UserStatusMap = map[int]string{
	UserStatusNormal:   "正常",
	UserStatusDisabled: "禁用",
	UserStatusDeleted:  "已注销",
}

// ==================== 学生认证状态 ====================
// 已迁移到 verify.go 文件中，使用更完整的状态机定义
// 请使用 constants.VerifyStatusXxx 常量

// ==================== 活动状态（状态机） ====================
//
// 状态流转：
//   Draft(0) -> Pending(1) -> Published(2) -> Ongoing(3) -> Ended(4)
//                  ↓
//              Rejected(5)
//   任意状态 -> Cancelled(6)
//

const (
	ActivityStatusDraft     = 0 // 草稿
	ActivityStatusPending   = 1 // 待审核
	ActivityStatusPublished = 2 // 已发布（报名中）
	ActivityStatusOngoing   = 3 // 进行中
	ActivityStatusEnded     = 4 // 已结束
	ActivityStatusRejected  = 5 // 已拒绝
	ActivityStatusCancelled = 6 // 已取消
)

// ActivityStatusMap 活动状态映射
var ActivityStatusMap = map[int]string{
	ActivityStatusDraft:     "草稿",
	ActivityStatusPending:   "待审核",
	ActivityStatusPublished: "已发布",
	ActivityStatusOngoing:   "进行中",
	ActivityStatusEnded:     "已结束",
	ActivityStatusRejected:  "已拒绝",
	ActivityStatusCancelled: "已取消",
}

// ActivityStatusTransitions 活动状态合法转换
// key: 当前状态, value: 可转换的目标状态列表
var ActivityStatusTransitions = map[int][]int{
	ActivityStatusDraft:     {ActivityStatusPending, ActivityStatusCancelled},
	ActivityStatusPending:   {ActivityStatusPublished, ActivityStatusRejected, ActivityStatusCancelled},
	ActivityStatusPublished: {ActivityStatusOngoing, ActivityStatusCancelled},
	ActivityStatusOngoing:   {ActivityStatusEnded, ActivityStatusCancelled},
	ActivityStatusRejected:  {ActivityStatusDraft}, // 可以重新编辑
}

// CanTransition 检查状态转换是否合法
func CanTransition(from, to int) bool {
	allowedStates, ok := ActivityStatusTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowedStates {
		if s == to {
			return true
		}
	}
	return false
}

// ==================== 报名状态 ====================

const (
	RegistrationStatusPending  = 0 // 待审核
	RegistrationStatusApproved = 1 // 已通过
	RegistrationStatusRejected = 2 // 已拒绝
	RegistrationStatusCanceled = 3 // 已取消
)

// RegistrationStatusMap 报名状态映射
var RegistrationStatusMap = map[int]string{
	RegistrationStatusPending:  "待审核",
	RegistrationStatusApproved: "已通过",
	RegistrationStatusRejected: "已拒绝",
	RegistrationStatusCanceled: "已取消",
}

// ==================== 签到状态 ====================

const (
	CheckinStatusNo  = 0 // 未签到
	CheckinStatusYes = 1 // 已签到
)

// ==================== 消息类型 ====================

const (
	MessageTypeText   = 1 // 文本消息
	MessageTypeImage  = 2 // 图片消息
	MessageTypeSystem = 3 // 系统消息
)

// ==================== 通知类型 ====================

const (
	NotifyTypeRegistration = 1 // 报名成功
	NotifyTypeReminder     = 2 // 活动提醒
	NotifyTypeAudit        = 3 // 审核结果
	NotifyTypeSystem       = 4 // 系统公告
)

// ==================== 通用状态 ====================

const (
	StatusEnabled  = 1 // 启用
	StatusDisabled = 2 // 禁用
)

// ==================== 布尔状态 ====================

const (
	No  = 0 // 否
	Yes = 1 // 是
)

// ==================== 分页默认值 ====================

const (
	DefaultPage     = 1   // 默认页码
	DefaultPageSize = 20  // 默认每页条数
	MaxPageSize     = 100 // 最大每页条数
)
