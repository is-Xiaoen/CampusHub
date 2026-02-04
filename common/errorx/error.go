package errorx

import (
	"fmt"

	"github.com/pkg/errors"
	"google.golang.org/grpc/status"
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
// 支持以下错误类型：
//  1. *BizError：直接返回
//  2. gRPC Status：从 RPC 返回的错误，解析 message 中的业务错误
//  3. 其他错误：返回内部错误（隐藏细节）
func FromError(err error) *BizError {
	if err == nil {
		return nil
	}

	// 获取原始错误（支持 errors.Wrap 包装的错误）
	causeErr := errors.Cause(err)

	// 1. 检查是否是本地 BizError
	if bizErr, ok := causeErr.(*BizError); ok {
		return bizErr
	}

	// 2. 检查是否是 gRPC Status（从 RPC 返回的错误）
	if gstatus, ok := status.FromError(causeErr); ok {
		message := gstatus.Message()
		grpcCode := int(gstatus.Code())

		// 尝试从 message 中解析业务错误码
		// go-zero 的 RPC 错误格式：message 中包含了业务错误信息
		// 格式可能是 "BizError: code=2201, message=认证记录不存在"
		// 或者直接是业务消息 "认证记录不存在"
		var bizCode int
		var bizMsg string

		// 尝试解析 "BizError: code=xxx, message=xxx" 格式
		n, _ := fmt.Sscanf(message, "BizError: code=%d, message=", &bizCode)
		if n == 1 && IsValidCode(bizCode) {
			// 提取 message= 后面的内容
			prefix := fmt.Sprintf("BizError: code=%d, message=", bizCode)
			if len(message) > len(prefix) {
				bizMsg = message[len(prefix):]
			} else {
				bizMsg = GetMessage(bizCode)
			}
			return &BizError{
				Code:    bizCode,
				Message: bizMsg,
			}
		}

		// 如果 gRPC code 本身是业务错误码
		if IsValidCode(grpcCode) {
			return &BizError{
				Code:    grpcCode,
				Message: message,
			}
		}

		// gRPC 标准错误码，但 message 可能有用
		// 返回内部错误，但记录原始消息供调试
	}

	// 3. 其他错误：返回内部错误，不暴露细节
	return &BizError{
		Code:    CodeInternalError,
		Message: "内部服务器错误",
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

// ErrInvalidToken Token无效
func ErrInvalidToken() *BizError {
	return New(CodeTokenInvalid)
}

// NewDefaultError 创建默认业务错误（通常用于提示用户）
func NewDefaultError(msg string) *BizError {
	return NewWithMessage(CodeInvalidParams, msg)
}

// NewSystemError 创建系统错误
func NewSystemError(msg string) *BizError {
	return NewWithMessage(CodeInternalError, msg)
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

// ============ 学生认证相关错误 ============

// ErrVerifyNotFound 认证记录不存在
func ErrVerifyNotFound() *BizError {
	return New(CodeVerifyNotFound)
}

// ErrVerifyAlreadyExist 认证记录已存在
func ErrVerifyAlreadyExist() *BizError {
	return New(CodeVerifyAlreadyExist)
}

// ErrVerifyNotVerified 用户未通过学生认证
func ErrVerifyNotVerified() *BizError {
	return New(CodeVerifyNotVerified)
}

// ErrVerifyStudentIDUsed 学号已被其他用户认证
func ErrVerifyStudentIDUsed() *BizError {
	return New(CodeVerifyStudentIDUsed)
}

// ErrVerifyCannotApply 当前状态不允许申请
func ErrVerifyCannotApply() *BizError {
	return New(CodeVerifyCannotApply)
}

// ErrVerifyCannotConfirm 当前状态不允许确认
func ErrVerifyCannotConfirm() *BizError {
	return New(CodeVerifyCannotConfirm)
}

// ErrVerifyCannotCancel 当前状态不允许取消
func ErrVerifyCannotCancel() *BizError {
	return New(CodeVerifyCannotCancel)
}

// ErrVerifyRateLimit 申请过于频繁
func ErrVerifyRateLimit() *BizError {
	return New(CodeVerifyRateLimit)
}

// ErrVerifyInvalidTransit 无效的状态转换
func ErrVerifyInvalidTransit() *BizError {
	return New(CodeVerifyInvalidTransit)
}

// ErrVerifyPermissionDeny 无权操作此认证记录
func ErrVerifyPermissionDeny() *BizError {
	return New(CodeVerifyPermissionDeny)
}

// ErrVerifyRejectCooldown 拒绝后冷却期内
func ErrVerifyRejectCooldown() *BizError {
	return New(CodeVerifyRejectCooldown)
}

// ============ OCR识别相关错误 ============

// ErrOcrNetworkTimeout OCR服务网络超时
func ErrOcrNetworkTimeout() *BizError {
	return New(CodeOcrNetworkTimeout)
}

// ErrOcrImageInvalid 图片无效或无法识别
func ErrOcrImageInvalid() *BizError {
	return New(CodeOcrImageInvalid)
}

// ErrOcrImageInvalidWithMsg 图片无效（自定义消息）
func ErrOcrImageInvalidWithMsg(msg string) *BizError {
	if msg == "" {
		return New(CodeOcrImageInvalid)
	}
	return NewWithMessage(CodeOcrImageInvalid, msg)
}

// ErrOcrRecognizeFailed OCR识别失败
func ErrOcrRecognizeFailed() *BizError {
	return New(CodeOcrRecognizeFailed)
}

// ErrOcrServiceUnavailable OCR服务不可用
func ErrOcrServiceUnavailable() *BizError {
	return New(CodeOcrServiceUnavailable)
}

// ErrOcrInsufficientBalance OCR服务余额不足
func ErrOcrInsufficientBalance() *BizError {
	return New(CodeOcrInsufficientBalance)
}

// ErrOcrEmptyResult OCR识别结果为空
func ErrOcrEmptyResult() *BizError {
	return New(CodeOcrEmptyResult)
}

// ErrOcrConfigInvalid OCR配置无效
func ErrOcrConfigInvalid() *BizError {
	return New(CodeOcrConfigInvalid)
}
