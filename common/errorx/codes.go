/**
 * @projectName: CampusHub
 * @package: errorx
 * @className: codes
 * @author: lijunqi
 * @description: 统一错误码定义和错误处理
 * @date: 2026-01-30
 * @version: 1.0
 */

package errorx

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// 错误码规范：
// 0       - 成功
// 1xxx    - 通用错误
// 2xxx    - 用户服务错误
// 3xxx    - 活动服务错误
// 4xxx    - 聊天服务错误

const (
	// CodeSuccess 成功
	CodeSuccess = 0

	// ============ 通用错误 1xxx ============

	// CodeInternalError 内部服务器错误
	CodeInternalError = 1000
	// CodeInvalidParams 参数校验失败
	CodeInvalidParams = 1001
	// CodeUnauthorized 未授权访问
	CodeUnauthorized = 1002
	// CodeForbidden 禁止访问
	CodeForbidden = 1003
	// CodeNotFound 资源不存在
	CodeNotFound = 1004
	// CodeTooManyRequests 请求过于频繁
	CodeTooManyRequests = 1005
	// CodeServiceUnavailable 服务暂不可用
	CodeServiceUnavailable = 1006
	// CodeTimeout 请求超时
	CodeTimeout = 1007
	// CodeDBError 数据库错误
	CodeDBError = 1008
	// CodeCacheError 缓存错误
	CodeCacheError = 1009
	// CodeRPCError RPC调用失败
	CodeRPCError = 1010

	// ============ 用户服务错误 2xxx ============

	// 认证相关 2001-2010

	// CodeLoginRequired 需要登录（auth中间件依赖）
	CodeLoginRequired = 2001
	// CodeTokenInvalid Token无效（auth中间件依赖）
	CodeTokenInvalid = 2002
	// CodeTokenExpired Token已过期（auth中间件依赖）
	CodeTokenExpired = 2003

	// 信用分相关 2101-2120

	// CodeCreditNotFound 信用记录不存在
	CodeCreditNotFound = 2101
	// CodeCreditAlreadyInit 信用分已初始化
	CodeCreditAlreadyInit = 2102
	// CodeCreditBlacklist 用户在黑名单中
	CodeCreditBlacklist = 2103
	// CodeCreditRiskLimit 风险用户已达每日限制
	CodeCreditRiskLimit = 2104
	// CodeCreditCannotPublish 信用分不足，无法发布
	CodeCreditCannotPublish = 2105
	// CodeCreditSourceDup 信用变更来源重复（幂等）
	CodeCreditSourceDup = 2106
	// CodeCreditInvalidChange 无效的信用变更类型
	CodeCreditInvalidChange = 2107

	// 学生认证相关 2201-2220

	// CodeVerifyNotFound 认证记录不存在
	CodeVerifyNotFound = 2201
	// CodeVerifyAlreadyExist 认证记录已存在
	CodeVerifyAlreadyExist = 2202
	// CodeVerifyNotVerified 用户未通过学生认证
	CodeVerifyNotVerified = 2203
	// CodeVerifyStudentIDUsed 学号已被其他用户认证
	CodeVerifyStudentIDUsed = 2204

	// ============ 活动服务错误 3xxx ============
	// TODO(马肖阳): 添加活动服务相关错误码

	// ============ 聊天服务错误 4xxx ============
	// TODO(马华恩): 添加聊天服务相关错误码
)

// codeMessages 错误码对应的默认消息
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

	// 用户服务错误 - 认证相关
	CodeLoginRequired: "请先登录",
	CodeTokenInvalid:  "登录状态无效",
	CodeTokenExpired:  "登录已过期",

	// 用户服务错误 - 信用分相关
	CodeCreditNotFound:      "信用记录不存在",
	CodeCreditAlreadyInit:   "信用分已初始化",
	CodeCreditBlacklist:     "您的账户信用分过低，已被限制操作",
	CodeCreditRiskLimit:     "您的信用分处于风险区间，每日仅限报名1次",
	CodeCreditCannotPublish: "信用分不足90分，暂时无法发布活动",
	CodeCreditSourceDup:     "该操作已处理，请勿重复提交",
	CodeCreditInvalidChange: "无效的信用变更类型",

	// 用户服务错误 - 学生认证相关
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

// ==================== CodeError 错误结构体 ====================

// CodeError 业务错误结构体
type CodeError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// Error 实现 error 接口
func (e *CodeError) Error() string {
	return fmt.Sprintf("code: %d, msg: %s", e.Code, e.Msg)
}

// GRPCStatus 将 CodeError 转换为 gRPC Status
func (e *CodeError) GRPCStatus() *status.Status {
	return status.New(codes.Code(e.Code), e.Msg)
}

// ==================== 错误构造函数 ====================

// New 创建业务错误（自定义消息）
func New(code int, msg string) *CodeError {
	return &CodeError{Code: code, Msg: msg}
}

// NewWithCode 创建业务错误（使用默认消息）
func NewWithCode(code int) *CodeError {
	return &CodeError{Code: code, Msg: GetMessage(code)}
}

// Wrap 包装原始错误
func Wrap(code int, err error) *CodeError {
	if err == nil {
		return nil
	}
	return &CodeError{Code: code, Msg: err.Error()}
}

// FromError 从 error 转换为 CodeError
func FromError(err error) *CodeError {
	if err == nil {
		return nil
	}
	if ce, ok := err.(*CodeError); ok {
		return ce
	}
	return &CodeError{Code: CodeInternalError, Msg: err.Error()}
}

// ==================== 常用错误快捷方法 ====================

// ErrInvalidParams 参数校验失败
func ErrInvalidParams(msg string) *CodeError {
	if msg == "" {
		return NewWithCode(CodeInvalidParams)
	}
	return New(CodeInvalidParams, msg)
}

// ErrDBError 数据库错误
func ErrDBError(err error) *CodeError {
	return Wrap(CodeDBError, err)
}

// ErrCacheError 缓存错误
func ErrCacheError(err error) *CodeError {
	return Wrap(CodeCacheError, err)
}

// ErrCreditNotFound 信用记录不存在
func ErrCreditNotFound() *CodeError {
	return NewWithCode(CodeCreditNotFound)
}

// ErrCreditAlreadyInit 信用分已初始化
func ErrCreditAlreadyInit() *CodeError {
	return NewWithCode(CodeCreditAlreadyInit)
}

// ErrCreditSourceDup 信用变更来源重复
func ErrCreditSourceDup() *CodeError {
	return NewWithCode(CodeCreditSourceDup)
}
