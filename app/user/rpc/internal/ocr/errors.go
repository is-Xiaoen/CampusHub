/**
 * @projectName: CampusHub
 * @package: ocr
 * @className: errors
 * @author: lijunqi
 * @description: OCR模块错误辅助函数（直接使用 errorx.BizError）
 * @date: 2026-02-02
 * @version: 3.0
 */

package ocr

import "activity-platform/common/errorx"

// ============================================================================
// 可重试错误码列表
// 用于 ProviderFactory 的故障转移判断
// ============================================================================

// retryableCodes 可重试的错误码列表
// 这些错误可以尝试切换到备用提供商
var retryableCodes = map[int]bool{
	errorx.CodeOcrNetworkTimeout:      true,  // 网络超时，可重试
	errorx.CodeOcrRecognizeFailed:     true,  // 识别失败，可重试
	errorx.CodeOcrServiceUnavailable:  true,  // 服务不可用，可重试
	errorx.CodeOcrImageInvalid:        false, // 图片无效，不可重试（用户问题）
	errorx.CodeOcrEmptyResult:         false, // 结果为空，不可重试（用户问题）
	errorx.CodeOcrInsufficientBalance: false, // 余额不足，不可重试（配置问题）
	errorx.CodeOcrConfigInvalid:       false, // 配置无效，不可重试（配置问题）
}

// ============================================================================
// 错误判断辅助函数
// ============================================================================

// IsRetryable 判断错误是否可重试
// 用于 ProviderFactory 决定是否切换到备用提供商
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	if bizErr, ok := err.(*errorx.BizError); ok {
		if retryable, exists := retryableCodes[bizErr.Code]; exists {
			return retryable
		}
	}

	// 未知错误默认可重试
	return true
}

// IsUserError 判断是否为用户导致的错误（图片问题）
// 这类错误不应该切换提供商，而是直接返回给用户
func IsUserError(err error) bool {
	if err == nil {
		return false
	}

	if bizErr, ok := err.(*errorx.BizError); ok {
		switch bizErr.Code {
		case errorx.CodeOcrImageInvalid, errorx.CodeOcrEmptyResult:
			return true
		}
	}

	return false
}
