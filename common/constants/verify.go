/**
 * @projectName: CampusHub
 * @package: constants
 * @className: verify
 * @author: lijunqi
 * @description: 学生认证状态机常量定义
 * @date: 2026-01-31
 * @version: 1.0
 */

package constants

// ============================================================================
// 学生认证状态机
// ============================================================================
//
// 状态流转图：
//   0(初始) -> 1(OCR审核中)
//   1(OCR审核中) -> 2(待确认) | 6(超时) | 7(取消) | 8(OCR失败)
//   2(待确认) -> 4(通过) | 3(人工审核) | 7(取消)
//   3(人工审核) -> 4(通过) | 5(拒绝) | 7(取消)
//   5,6,7,8 -> 0(初始) [允许重新申请]
//
// ============================================================================

// VerifyStatus 认证状态常量
const (
	// VerifyStatusInit 初始化（可申请）
	VerifyStatusInit int8 = 0
	// VerifyStatusOcrPending OCR审核中
	VerifyStatusOcrPending int8 = 1
	// VerifyStatusWaitConfirm 待用户确认（OCR成功）
	VerifyStatusWaitConfirm int8 = 2
	// VerifyStatusManualReview 人工审核中（用户修改了信息）
	VerifyStatusManualReview int8 = 3
	// VerifyStatusPassed 已通过（最终状态）
	VerifyStatusPassed int8 = 4
	// VerifyStatusRejected 已拒绝（人工审核拒绝）
	VerifyStatusRejected int8 = 5
	// VerifyStatusTimeout 已超时（10分钟未完成OCR）
	VerifyStatusTimeout int8 = 6
	// VerifyStatusCancelled 已取消（用户主动取消）
	VerifyStatusCancelled int8 = 7
	// VerifyStatusOcrFailed OCR失败（双OCR都失败，可重试）
	VerifyStatusOcrFailed int8 = 8
)

// VerifyStatusNameMap 状态名称映射
var VerifyStatusNameMap = map[int8]string{
	VerifyStatusInit:         "未申请",
	VerifyStatusOcrPending:   "OCR审核中",
	VerifyStatusWaitConfirm:  "待确认",
	VerifyStatusManualReview: "人工审核中",
	VerifyStatusPassed:       "已通过",
	VerifyStatusRejected:     "已拒绝",
	VerifyStatusTimeout:      "已超时",
	VerifyStatusCancelled:    "已取消",
	VerifyStatusOcrFailed:    "识别失败",
}

// GetVerifyStatusName 获取状态名称
func GetVerifyStatusName(status int8) string {
	if name, ok := VerifyStatusNameMap[status]; ok {
		return name
	}
	return "未知状态"
}

// VerifyStatusTransitions 状态转换规则
// key: 当前状态, value: 允许转换的目标状态
var VerifyStatusTransitions = map[int8][]int8{
	VerifyStatusInit:         {VerifyStatusOcrPending},
	VerifyStatusOcrPending:   {VerifyStatusWaitConfirm, VerifyStatusOcrFailed, VerifyStatusTimeout, VerifyStatusCancelled},
	VerifyStatusWaitConfirm:  {VerifyStatusPassed, VerifyStatusManualReview, VerifyStatusCancelled},
	VerifyStatusManualReview: {VerifyStatusPassed, VerifyStatusRejected, VerifyStatusCancelled},
	VerifyStatusOcrFailed:    {VerifyStatusInit},
	VerifyStatusRejected:     {VerifyStatusInit},
	VerifyStatusTimeout:      {VerifyStatusInit},
	VerifyStatusCancelled:    {VerifyStatusInit},
}

// CanApplyStatuses 可以提交申请的状态集合
var CanApplyStatuses = []int8{
	VerifyStatusInit,
	VerifyStatusOcrFailed,
	VerifyStatusRejected,
	VerifyStatusTimeout,
	VerifyStatusCancelled,
}

// CanCancelStatuses 可以取消的状态集合
var CanCancelStatuses = []int8{
	VerifyStatusOcrPending,
	VerifyStatusWaitConfirm,
	VerifyStatusManualReview,
}

// CanConfirmStatuses 可以确认/修改的状态集合
var CanConfirmStatuses = []int8{
	VerifyStatusWaitConfirm,
}

// CanVerifyTransition 检查状态转换是否合法
func CanVerifyTransition(from, to int8) bool {
	allowedStates, ok := VerifyStatusTransitions[from]
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

// IsInStatuses 检查状态是否在指定集合中
func IsInStatuses(status int8, statuses []int8) bool {
	for _, s := range statuses {
		if status == s {
			return true
		}
	}
	return false
}

// CanApply 检查是否可以提交申请
func CanApply(status int8) bool {
	return IsInStatuses(status, CanApplyStatuses)
}

// CanCancel 检查是否可以取消
func CanCancel(status int8) bool {
	return IsInStatuses(status, CanCancelStatuses)
}

// CanConfirm 检查是否可以确认/修改
func CanConfirm(status int8) bool {
	return IsInStatuses(status, CanConfirmStatuses)
}

// IsFinalStatus 检查是否为最终状态
func IsFinalStatus(status int8) bool {
	return status == VerifyStatusPassed
}

// NeedRetry 检查是否需要/可以重试
func NeedRetry(status int8) bool {
	return status == VerifyStatusOcrFailed ||
		status == VerifyStatusTimeout ||
		status == VerifyStatusCancelled
}

// ============================================================================
// 前端动作指示（返回给前端的中文提示）
// ============================================================================

// VerifyAction 前端应执行的动作（中文，便于前端直接展示）
const (
	// VerifyActionApply 显示申请表单
	VerifyActionApply = "请填写认证信息"
	// VerifyActionWaitOcr 等待OCR
	VerifyActionWaitOcr = "正在识别中，请稍候"
	// VerifyActionConfirm 显示确认页
	VerifyActionConfirm = "请确认识别信息"
	// VerifyActionWaitManual 等待人工审核
	VerifyActionWaitManual = "已提交审核，请耐心等待"
	// VerifyActionDone 认证完成
	VerifyActionDone = "认证已完成"
	// VerifyActionRejected 被拒绝可重新申请
	VerifyActionRejected = "认证被拒绝，可重新申请"
	// VerifyActionFailed 失败可重试
	VerifyActionFailed = "识别失败，请重新提交"
	// VerifyActionTimeout 超时可重试
	VerifyActionTimeout = "识别超时，请重新提交"
	// VerifyActionCancelled 已取消可重新申请
	VerifyActionCancelled = "已取消，可重新申请"
)

// GetNeedAction 根据状态获取前端应执行的动作
func GetNeedAction(status int8) string {
	switch status {
	case VerifyStatusInit:
		return VerifyActionApply
	case VerifyStatusOcrPending:
		return VerifyActionWaitOcr
	case VerifyStatusWaitConfirm:
		return VerifyActionConfirm
	case VerifyStatusManualReview:
		return VerifyActionWaitManual
	case VerifyStatusPassed:
		return VerifyActionDone
	case VerifyStatusRejected:
		return VerifyActionRejected
	case VerifyStatusTimeout:
		return VerifyActionTimeout
	case VerifyStatusCancelled:
		return VerifyActionCancelled
	case VerifyStatusOcrFailed:
		return VerifyActionFailed
	default:
		return VerifyActionApply
	}
}

// ============================================================================
// 业务配置常量
// ============================================================================

const (
	// VerifyRateLimitWindow 限流窗口时间（秒）
	VerifyRateLimitWindow = 20

	// VerifyRateLimitMax 限流窗口内最大请求次数
	VerifyRateLimitMax = 2

	// VerifyOcrTimeoutMinutes OCR超时时间（分钟）
	VerifyOcrTimeoutMinutes = 10

	// VerifyRejectCooldownHours 拒绝后冷却期（小时）
	VerifyRejectCooldownHours = 24
)

// ============================================================================
// 操作来源常量
// ============================================================================

const (
	// VerifyOperatorUserApply 用户申请
	VerifyOperatorUserApply = "user_apply"
	// VerifyOperatorOcrCallback OCR回调
	VerifyOperatorOcrCallback = "ocr_callback"
	// VerifyOperatorManualReview 人工审核
	VerifyOperatorManualReview = "manual_review"
	// VerifyOperatorTimeoutJob 超时任务
	VerifyOperatorTimeoutJob = "timeout_job"
	// VerifyOperatorUserConfirm 用户确认
	VerifyOperatorUserConfirm = "user_confirm"
	// VerifyOperatorUserCancel 用户取消
	VerifyOperatorUserCancel = "user_cancel"
)
