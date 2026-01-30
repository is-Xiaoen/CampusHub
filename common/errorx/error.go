package errorx

import (
	"fmt"
)

// BizError 业务错误，实现 error 接口
type BizError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error 实现 error 接口
func (e *BizError) Error() string {
	return fmt.Sprintf("BizError: code=%d, message=%s", e.Code, e.Message)
}

// GetCode 获取错误码
func (e *BizError) GetCode() int {
	return e.Code
}

// GetMessage 获取错误消息
func (e *BizError) GetMessage() string {
	return e.Message
}

// New 创建业务错误（使用默认消息）
func New(code int) *BizError {
	return &BizError{
		Code:    code,
		Message: GetMessage(code),
	}
}

// NewWithMessage 创建业务错误（自定义消息）
func NewWithMessage(code int, message string) *BizError {
	return &BizError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装错误，添加上下文信息
func Wrap(code int, err error) *BizError {
	if err == nil {
		return New(code)
	}
	return &BizError{
		Code:    code,
		Message: fmt.Sprintf("%s: %v", GetMessage(code), err),
	}
}

// Is 判断是否为特定错误码
func Is(err error, code int) bool {
	if err == nil {
		return false
	}
	if bizErr, ok := err.(*BizError); ok {
		return bizErr.Code == code
	}
	return false
}

// FromError 从 error 转换为 BizError
// 如果不是 BizError，则返回内部错误
func FromError(err error) *BizError {
	if err == nil {
		return nil
	}
	if bizErr, ok := err.(*BizError); ok {
		return bizErr
	}
	return &BizError{
		Code:    CodeInternalError,
		Message: err.Error(),
	}
}

// ============ 常用错误快捷方法 ============

// ErrInternalError 内部错误
func ErrInternalError() *BizError {
	return New(CodeInternalError)
}

// ErrInvalidParams 参数错误
func ErrInvalidParams(msg string) *BizError {
	if msg == "" {
		return New(CodeInvalidParams)
	}
	return NewWithMessage(CodeInvalidParams, msg)
}

// ErrUnauthorized 未授权
func ErrUnauthorized() *BizError {
	return New(CodeUnauthorized)
}

// ErrForbidden 禁止访问
func ErrForbidden() *BizError {
	return New(CodeForbidden)
}

// ErrNotFound 资源不存在
func ErrNotFound() *BizError {
	return New(CodeNotFound)
}

// ErrTooManyRequests 请求过于频繁
func ErrTooManyRequests() *BizError {
	return New(CodeTooManyRequests)
}

// ErrDBError 数据库错误
func ErrDBError(err error) *BizError {
	return Wrap(CodeDBError, err)
}

// ErrCacheError 缓存错误
func ErrCacheError(err error) *BizError {
	return Wrap(CodeCacheError, err)
}

// ErrRPCError RPC调用错误
func ErrRPCError(err error) *BizError {
	return Wrap(CodeRPCError, err)
}

// ErrCreditNotFound 信用记录不存在
func ErrCreditNotFound() *BizError {
	return New(CodeCreditNotFound)
}

// ErrCreditAlreadyInit 信用分已初始化
func ErrCreditAlreadyInit() *BizError {
	return New(CodeCreditAlreadyInit)
}

// ErrCreditSourceDup 信用变更来源重复
func ErrCreditSourceDup() *BizError {
	return New(CodeCreditSourceDup)
}
