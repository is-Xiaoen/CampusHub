/**
 * @projectName: CampusHub
 * @package: errorx
 * @className: codes
 * @author: lijunqi
 * @description: 统一错误码定义
 * @date: 2026-01-30
 * @version: 1.0
 */

package errorx

// 错误码规范：
// 0       - 成功
// 1xxx    - 通用错误
// 2xxx    - 用户服务错误
// 3xxx    - 活动服务错误
// 4xxx    - 聊天服务错误

const (
	CodeSuccess            = 0    // 成功
	CodeInternalError      = 1000 // 内部服务器错误
	CodeInvalidParams      = 1001 // 参数校验失败
	CodeUnauthorized       = 1002 // 未授权访问
	CodeForbidden          = 1003 // 禁止访问
	CodeNotFound           = 1004 // 资源不存在
	CodeTooManyRequests    = 1005 // 请求过于频繁
	CodeServiceUnavailable = 1006 // 服务暂不可用
	CodeTimeout            = 1007 // 请求超时
	CodeDBError            = 1008 // 数据库错误
	CodeCacheError         = 1009 // 缓存错误
	CodeRPCError           = 1010 // RPC调用失败

	// 用户服务 - 认证 2001-2010
	CodeLoginRequired = 2001 // 需要登录
	CodeTokenInvalid  = 2002 // Token无效
	CodeTokenExpired  = 2003 // Token已过期

	// 用户服务 - 信用分 2101-2120
	CodeCreditNotFound      = 2101 // 信用记录不存在
	CodeCreditAlreadyInit   = 2102 // 信用分已初始化
	CodeCreditBlacklist     = 2103 // 用户在黑名单中
	CodeCreditRiskLimit     = 2104 // 风险用户已达每日限制
	CodeCreditCannotPublish = 2105 // 信用分不足，无法发布
	CodeCreditSourceDup     = 2106 // 信用变更来源重复
	CodeCreditInvalidChange = 2107 // 无效的信用变更类型

	// 用户服务 - 学生认证 2201-2230
	CodeVerifyNotFound       = 2201 // 认证记录不存在
	CodeVerifyAlreadyExist   = 2202 // 认证记录已存在
	CodeVerifyNotVerified    = 2203 // 用户未通过学生认证
	CodeVerifyStudentIDUsed  = 2204 // 学号已被其他用户认证
	CodeVerifyCannotApply    = 2205 // 当前状态不允许申请
	CodeVerifyCannotConfirm  = 2206 // 当前状态不允许确认
	CodeVerifyCannotCancel   = 2207 // 当前状态不允许取消
	CodeVerifyRateLimit      = 2208 // 申请过于频繁
	CodeVerifyInvalidTransit = 2209 // 无效的状态转换
	CodeVerifyPermissionDeny = 2210 // 无权操作此认证记录
	CodeVerifyRejectCooldown = 2211 // 拒绝后冷却期内，暂不能申请

	// 用户服务 - OCR识别 2231-2250
	CodeOcrNetworkTimeout      = 2231 // OCR服务网络超时
	CodeOcrImageInvalid        = 2232 // 图片无效或无法识别
	CodeOcrRecognizeFailed     = 2233 // OCR识别失败
	CodeOcrServiceUnavailable  = 2234 // OCR服务不可用
	CodeOcrInsufficientBalance = 2235 // OCR服务余额不足
	CodeOcrEmptyResult         = 2236 // OCR识别结果为空
	CodeOcrConfigInvalid       = 2237 // OCR配置无效

	// 活动服务 - 活动 3001-3050
	CodeActivityNotFound         = 3001 // 活动不存在
	CodeActivityStatusInvalid    = 3002 // 活动状态不允许此操作
	CodeActivityTimeInvalid      = 3003 // 活动时间设置无效
	CodeActivityConcurrentUpdate = 3004 // 活动并发更新冲突
	CodeActivityPermissionDenied = 3005 // 无权限操作此活动
	CodeActivityHasRegistration  = 3006 // 有报名记录不能删除

	// 活动服务 - 分类 3101-3120
	CodeCategoryNotFound = 3101 // 分类不存在
	CodeCategoryDisabled = 3102 // 分类已禁用

	// 活动服务 - 标签 3201-3220
	CodeTagNotFound      = 3201 // 标签不存在
	CodeTagLimitExceeded = 3202 // 标签数量超过限制

	// 聊天服务 4xxx - TODO(马华恩)
)

// codeMessages 错误码对应的默认消息
var codeMessages = map[int]string{
	CodeSuccess:             "success",
	CodeInternalError:       "内部服务器错误",
	CodeInvalidParams:       "参数校验失败",
	CodeUnauthorized:        "未授权访问",
	CodeForbidden:           "禁止访问",
	CodeNotFound:            "资源不存在",
	CodeTooManyRequests:     "请求过于频繁，请稍后再试",
	CodeServiceUnavailable:  "服务暂不可用",
	CodeTimeout:             "请求超时",
	CodeDBError:             "数据库错误",
	CodeCacheError:          "缓存错误",
	CodeRPCError:            "服务调用失败",
	CodeLoginRequired:       "请先登录",
	CodeTokenInvalid:        "登录状态无效",
	CodeTokenExpired:        "登录已过期",
	CodeCreditNotFound:      "信用记录不存在",
	CodeCreditAlreadyInit:   "信用分已初始化",
	CodeCreditBlacklist:     "您的账户信用分过低，已被限制操作",
	CodeCreditRiskLimit:     "您的信用分处于风险区间，每日仅限报名1次",
	CodeCreditCannotPublish: "信用分不足90分，暂时无法发布活动",
	CodeCreditSourceDup:     "该操作已处理，请勿重复提交",
	CodeCreditInvalidChange: "无效的信用变更类型",
	CodeVerifyNotFound:      "认证记录不存在",
	CodeVerifyAlreadyExist:  "认证记录已存在",
	CodeVerifyNotVerified:   "请先完成学生认证",
	CodeVerifyStudentIDUsed: "该学号已被其他用户认证",
	// 活动服务
	CodeActivityNotFound:         "活动不存在",
	CodeActivityStatusInvalid:    "活动状态不允许此操作",
	CodeActivityTimeInvalid:      "活动时间设置无效",
	CodeActivityConcurrentUpdate: "操作冲突，请重试",
	CodeActivityPermissionDenied: "无权限操作此活动",
	CodeActivityHasRegistration:  "有报名记录的活动不能删除",
	CodeCategoryNotFound:         "分类不存在",
	CodeCategoryDisabled:         "分类已禁用",
	CodeTagNotFound:              "标签不存在",
	CodeTagLimitExceeded:         "最多选择5个标签",
	// 学生认证
	CodeVerifyCannotApply:      "当前状态不允许申请认证",
	CodeVerifyCannotConfirm:    "当前状态不允许确认认证",
	CodeVerifyCannotCancel:     "当前状态不允许取消认证",
	CodeVerifyRateLimit:        "申请过于频繁，请20秒后再试",
	CodeVerifyInvalidTransit:   "无效的状态转换",
	CodeVerifyPermissionDeny:   "无权操作此认证记录",
	CodeVerifyRejectCooldown:   "您的认证申请被拒绝后24小时内不能重新申请",
	CodeOcrNetworkTimeout:      "识别服务繁忙，请稍后重试",
	CodeOcrImageInvalid:        "图片无效，请上传清晰的学生证照片",
	CodeOcrRecognizeFailed:     "识别失败，请重新上传照片",
	CodeOcrServiceUnavailable:  "识别服务暂不可用，请稍后重试",
	CodeOcrInsufficientBalance: "识别服务暂不可用，请联系管理员",
	CodeOcrEmptyResult:         "未能识别到有效信息，请上传清晰的学生证照片",
	CodeOcrConfigInvalid:       "识别服务配置错误，请联系管理员",
}

// GetMessage 根据错误码获取默认消息
func GetMessage(code int) string {
	if msg, ok := codeMessages[code]; ok {
		return msg
	}
	return "未知错误"
}

// IsValidCode 判断是否为有效的业务错误码
// 用于区分业务错误码和 gRPC 系统错误码
// 业务错误码应该返回给前端，系统错误码（如 Unknown=2）应该隐藏
func IsValidCode(code int) bool {
	_, exists := codeMessages[code]
	return exists
}
