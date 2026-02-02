package messaging

import (
	"errors"
	"fmt"
)

// 基础错误定义

// ErrInvalidMessage 无效消息错误
var ErrInvalidMessage = errors.New("无效消息")

// ErrInvalidTopic 无效主题错误
var ErrInvalidTopic = errors.New("无效主题")

// ErrTimeout 超时错误
var ErrTimeout = errors.New("超时")

// ErrConnectionFailed 连接失败错误
var ErrConnectionFailed = errors.New("连接失败")

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
	if errors.Is(err, ErrTimeout) || errors.Is(err, ErrConnectionFailed) {
		return true
	}

	// 检查已知的不可重试错误
	if errors.Is(err, ErrInvalidMessage) || errors.Is(err, ErrInvalidTopic) {
		return false
	}

	// 默认情况下，未知错误视为可重试
	return true
}

// WrapError 包装错误并添加上下文
func WrapError(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s: %w", msg, err)
}
