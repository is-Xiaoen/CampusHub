package messaging

import (
	"errors"
	"fmt"
)

// 错误类型定义

// ErrInvalidMessage 无效消息错误（不可重试）
var ErrInvalidMessage = errors.New("无效消息")

// ErrMessageTooLarge 消息过大错误（不可重试）
var ErrMessageTooLarge = errors.New("消息过大")

// ErrInvalidTopic 无效主题错误（不可重试）
var ErrInvalidTopic = errors.New("无效主题")

// ErrTimeout 超时错误（可重试）
var ErrTimeout = errors.New("超时")

// ErrConnectionFailed 连接失败错误（可重试）
var ErrConnectionFailed = errors.New("连接失败")

// ErrTemporaryFailure 临时失败错误（可重试）
var ErrTemporaryFailure = errors.New("临时失败")

// ErrPermanentFailure 永久失败错误（不可重试）
var ErrPermanentFailure = errors.New("永久失败")

// ErrHandlerPanic 处理器 panic 错误（不可重试）
var ErrHandlerPanic = errors.New("处理器panic")

// ErrMaxRetriesExceeded 超过最大重试次数错误
var ErrMaxRetriesExceeded = errors.New("超过最大重试次数")

// ErrorType 错误类型
type ErrorType int

const (
	// ErrorTypeRetryable 可重试错误
	ErrorTypeRetryable ErrorType = iota
	// ErrorTypeNonRetryable 不可重试错误
	ErrorTypeNonRetryable
	// ErrorTypeUnknown 未知错误类型
	ErrorTypeUnknown
)

// RetryableError 可重试错误接口
type RetryableError interface {
	error
	IsRetryable() bool
}

// retryableError 可重试错误实现
type retryableError struct {
	err       error
	retryable bool
}

func (e *retryableError) Error() string {
	return e.err.Error()
}

func (e *retryableError) Unwrap() error {
	return e.err
}

func (e *retryableError) IsRetryable() bool {
	return e.retryable
}

// NewRetryableError 创建可重试错误
func NewRetryableError(err error) error {
	return &retryableError{
		err:       err,
		retryable: true,
	}
}

// NewNonRetryableError 创建不可重试错误
func NewNonRetryableError(err error) error {
	return &retryableError{
		err:       err,
		retryable: false,
	}
}

// IsRetryable 判断错误是否可重试
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// 检查是否实现了 RetryableError 接口
	var retryableErr RetryableError
	if errors.As(err, &retryableErr) {
		return retryableErr.IsRetryable()
	}

	// 检查已知的可重试错误
	if errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrConnectionFailed) ||
		errors.Is(err, ErrTemporaryFailure) {
		return true
	}

	// 检查已知的不可重试错误
	if errors.Is(err, ErrInvalidMessage) ||
		errors.Is(err, ErrMessageTooLarge) ||
		errors.Is(err, ErrInvalidTopic) ||
		errors.Is(err, ErrPermanentFailure) ||
		errors.Is(err, ErrHandlerPanic) ||
		errors.Is(err, ErrMaxRetriesExceeded) {
		return false
	}

	// 默认情况下，未知错误视为可重试
	return true
}

// ClassifyError 分类错误
func ClassifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}

	if IsRetryable(err) {
		return ErrorTypeRetryable
	}

	return ErrorTypeNonRetryable
}

// WrapError 包装错误并添加上下文
func WrapError(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}

	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s: %w", msg, err)
}

// ValidationError 验证错误（不可重试）
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("验证错误: 字段=%s, 消息=%s", e.Field, e.Message)
}

func (e *ValidationError) IsRetryable() bool {
	return false
}

// NewValidationError 创建验证错误
func NewValidationError(field, message string) error {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// TimeoutError 超时错误（可重试）
type TimeoutError struct {
	Operation string
	Duration  string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("超时: 操作=%s, 持续时间=%s", e.Operation, e.Duration)
}

func (e *TimeoutError) IsRetryable() bool {
	return true
}

// NewTimeoutError 创建超时错误
func NewTimeoutError(operation, duration string) error {
	return &TimeoutError{
		Operation: operation,
		Duration:  duration,
	}
}
