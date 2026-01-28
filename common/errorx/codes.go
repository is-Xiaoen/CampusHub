package errorx

// 错误码规范：
// 0       - 成功
// 1xxx    - 通用错误
// 2xxx    - 用户服务错误
// 3xxx    - 活动服务错误
// 4xxx    - 聊天服务错误

const (
	// 成功
	CodeSuccess = 0

	// ============ 通用错误 1xxx ============
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

	// ============ 用户服务错误 2xxx ============
	// TODO(杨春路): 添加用户服务相关错误码
	CodeLoginRequired = 2001 // 需要登录（auth中间件依赖）
	CodeTokenInvalid  = 2002 // Token无效（auth中间件依赖）
	CodeTokenExpired  = 2003 // Token已过期（auth中间件依赖）

	// ============ 活动服务错误 3xxx ============
	// TODO(马肖阳): 添加活动服务相关错误码

	// ============ 聊天服务错误 4xxx ============
	// TODO(马华恩): 添加聊天服务相关错误码
)

// 错误码对应的默认消息
var codeMessages = map[int]string{
	CodeSuccess: "success",

	// 通用错误
	CodeInternalError:      "内部服务器错误",
	CodeInvalidParams:      "参数校验失败",
	CodeUnauthorized:       "未授权访问",
	CodeForbidden:          "禁止访问",
	CodeNotFound:           "资源不存在",
	CodeTooManyRequests:    "请求过于频繁，请稍后再试",
	CodeServiceUnavailable: "服务暂不可用",
	CodeTimeout:            "请求超时",
	CodeDBError:            "数据库错误",
	CodeCacheError:         "缓存错误",
	CodeRPCError:           "服务调用失败",

	// 用户服务错误（auth中间件依赖的最小集）
	CodeLoginRequired: "请先登录",
	CodeTokenInvalid:  "登录状态无效",
	CodeTokenExpired:  "登录已过期",

	// TODO: 其他错误码消息由各负责人添加
}

// GetMessage 根据错误码获取默认消息
func GetMessage(code int) string {
	if msg, ok := codeMessages[code]; ok {
		return msg
	}
	return "未知错误"
}
