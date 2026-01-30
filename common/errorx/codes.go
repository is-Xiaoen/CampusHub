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
// 用于封装业务错误码和错误信息
type CodeError struct {
	// Code 业务错误码
	Code int `json:"code"`
	// Msg 错误信息
	Msg string `json:"msg"`
}

// Error 实现 error 接口
func (e *CodeError) Error() string {
	return fmt.Sprintf("code: %d, msg: %s", e.Code, e.Msg)
}

// GetCode 获取错误码
func (e *CodeError) GetCode() int {
	return e.Code
}

// GetMsg 获取错误信息
func (e *CodeError) GetMsg() string {
	return e.Msg
}

// GRPCStatus 将 CodeError 转换为 gRPC Status
// 便于在 RPC 调用中传递业务错误码
func (e *CodeError) GRPCStatus() *status.Status {
	return status.New(codes.Code(e.Code), e.Msg)
}

// ==================== 错误构造函数 ====================

// New 创建一个新的业务错误
// code: 错误码
// msg: 错误信息
func New(code int, msg string) *CodeError {
	return &CodeError{
		Code: code,
		Msg:  msg,
	}
}

// NewWithCode 根据错误码创建错误（使用默认消息）
// code: 错误码
func NewWithCode(code int) *CodeError {
	return &CodeError{
		Code: code,
		Msg:  GetMessage(code),
	}
}

// NewWithMsg 根据错误码创建错误（自定义消息）
// code: 错误码
// msg: 自定义错误信息
func NewWithMsg(code int, msg string) *CodeError {
	return &CodeError{
		Code: code,
		Msg:  msg,
	}
}

// Wrap 包装原始错误，添加业务错误码
// code: 错误码
// err: 原始错误
func Wrap(code int, err error) *CodeError {
	if err == nil {
		return nil
	}
	return &CodeError{
		Code: code,
		Msg:  err.Error(),
	}
}

// ==================== 常用错误快捷方法 ====================

// ErrInternalError 内部服务器错误
func ErrInternalError() *CodeError {
	return NewWithCode(CodeInternalError)
}

// ErrInvalidParams 参数校验失败
func ErrInvalidParams(msg string) *CodeError {
	if msg == "" {
		return NewWithCode(CodeInvalidParams)
	}
	return New(CodeInvalidParams, msg)
}

// ErrNotFound 资源不存在
func ErrNotFound(resource string) *CodeError {
	if resource == "" {
		return NewWithCode(CodeNotFound)
	}
	return New(CodeNotFound, fmt.Sprintf("%s不存在", resource))
}

// ErrDBError 数据库错误
func ErrDBError(err error) *CodeError {
	return Wrap(CodeDBError, err)
}

// ErrCacheError 缓存错误
func ErrCacheError(err error) *CodeError {
	return Wrap(CodeCacheError, err)
}

// ErrRPCError RPC调用失败
func ErrRPCError(err error) *CodeError {
	return Wrap(CodeRPCError, err)
}

// ==================== 信用分相关错误 ====================

// ErrCreditNotFound 信用记录不存在
func ErrCreditNotFound() *CodeError {
	return NewWithCode(CodeCreditNotFound)
}

// ErrCreditAlreadyInit 信用分已初始化
func ErrCreditAlreadyInit() *CodeError {
	return NewWithCode(CodeCreditAlreadyInit)
}

// ErrCreditBlacklist 用户在黑名单中
func ErrCreditBlacklist() *CodeError {
	return NewWithCode(CodeCreditBlacklist)
}

// ErrCreditRiskLimit 风险用户已达每日限制
func ErrCreditRiskLimit() *CodeError {
	return NewWithCode(CodeCreditRiskLimit)
}

// ErrCreditCannotPublish 信用分不足，无法发布
func ErrCreditCannotPublish() *CodeError {
	return NewWithCode(CodeCreditCannotPublish)
}

// ErrCreditSourceDup 信用变更来源重复
func ErrCreditSourceDup() *CodeError {
	return NewWithCode(CodeCreditSourceDup)
}

// ErrCreditInvalidChange 无效的信用变更类型
func ErrCreditInvalidChange() *CodeError {
	return NewWithCode(CodeCreditInvalidChange)
}

// ==================== 学生认证相关错误 ====================

// ErrVerifyNotFound 认证记录不存在
func ErrVerifyNotFound() *CodeError {
	return NewWithCode(CodeVerifyNotFound)
}

// ErrVerifyAlreadyExist 认证记录已存在
func ErrVerifyAlreadyExist() *CodeError {
	return NewWithCode(CodeVerifyAlreadyExist)
}

// ErrVerifyNotVerified 用户未通过学生认证
func ErrVerifyNotVerified() *CodeError {
	return NewWithCode(CodeVerifyNotVerified)
}

// ErrVerifyStudentIDUsed 学号已被其他用户认证
func ErrVerifyStudentIDUsed() *CodeError {
	return NewWithCode(CodeVerifyStudentIDUsed)
}

// ==================== 错误判断辅助函数 ====================

// IsCodeError 判断是否为 CodeError 类型
func IsCodeError(err error) bool {
	_, ok := err.(*CodeError)
	return ok
}

// GetCodeError 从 error 中提取 CodeError
// 如果不是 CodeError 类型，返回 nil
func GetCodeError(err error) *CodeError {
	if err == nil {
		return nil
	}
	if ce, ok := err.(*CodeError); ok {
		return ce
	}
	return nil
}
