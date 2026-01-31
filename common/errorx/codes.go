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

	// 用户服务 - 学生认证 2201-2220
	CodeVerifyNotFound      = 2201 // 认证记录不存在
	CodeVerifyAlreadyExist  = 2202 // 认证记录已存在
	CodeVerifyNotVerified   = 2203 // 用户未通过学生认证
	CodeVerifyStudentIDUsed = 2204 // 学号已被其他用户认证

	// 活动服务 3xxx - TODO(马肖阳)
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
