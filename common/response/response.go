package response

import (
	"context"
	"net/http"

	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// SetupGlobalErrorHandler 设置全局错误处理器
// 必须在 server.Start() 之前调用
// 这样 goctl 生成的 handler 中的 httpx.ErrorCtx 也会使用统一格式
func SetupGlobalErrorHandler() {
	httpx.SetErrorHandler(func(err error) (int, interface{}) {
		bizErr := errorx.FromError(err)
		return getHttpStatus(bizErr.Code), &Response{
			Code:    bizErr.Code,
			Message: bizErr.Message,
		}
	})

	httpx.SetErrorHandlerCtx(func(ctx context.Context, err error) (int, interface{}) {
		bizErr := errorx.FromError(err)
		return getHttpStatus(bizErr.Code), &Response{
			Code:    bizErr.Code,
			Message: bizErr.Message,
		}
	})
}

// SetupGlobalOkHandler 设置全局成功处理器（可选）
// 如果想让 httpx.OkJsonCtx 也使用统一格式，调用此方法
// 注意：这会让所有响应都包装在 {code, message, data} 中
func SetupGlobalOkHandler() {
	httpx.SetOkHandler(func(ctx context.Context, data interface{}) interface{} {
		return &Response{
			Code:    errorx.CodeSuccess,
			Message: "success",
			Data:    data,
		}
	})
}

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageData 分页数据结构
type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

// Success 成功响应
func Success(w http.ResponseWriter, data interface{}) {
	resp := &Response{
		Code:    errorx.CodeSuccess,
		Message: "success",
		Data:    data,
	}
	httpx.OkJson(w, resp)
}

// SuccessWithMessage 成功响应（自定义消息）
func SuccessWithMessage(w http.ResponseWriter, message string, data interface{}) {
	resp := &Response{
		Code:    errorx.CodeSuccess,
		Message: message,
		Data:    data,
	}
	httpx.OkJson(w, resp)
}

// SuccessWithPage 分页成功响应
func SuccessWithPage(w http.ResponseWriter, list interface{}, total int64, page, pageSize int) {
	resp := &Response{
		Code:    errorx.CodeSuccess,
		Message: "success",
		Data: PageData{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	}
	httpx.OkJson(w, resp)
}

// Fail 失败响应（使用 BizError）
func Fail(w http.ResponseWriter, err error) {
	bizErr := errorx.FromError(err)
	resp := &Response{
		Code:    bizErr.Code,
		Message: bizErr.Message,
	}
	// 根据错误类型返回不同的 HTTP 状态码
	httpx.WriteJson(w, getHttpStatus(bizErr.Code), resp)
}

// FailWithCode 失败响应（指定错误码）
func FailWithCode(w http.ResponseWriter, code int) {
	resp := &Response{
		Code:    code,
		Message: errorx.GetMessage(code),
	}
	httpx.WriteJson(w, getHttpStatus(code), resp)
}

// FailWithCodeAndMessage 失败响应（指定错误码和消息）
func FailWithCodeAndMessage(w http.ResponseWriter, code int, message string) {
	resp := &Response{
		Code:    code,
		Message: message,
	}
	httpx.WriteJson(w, getHttpStatus(code), resp)
}

// getHttpStatus 根据业务错误码映射 HTTP 状态码
func getHttpStatus(code int) int {
	switch code {
	case errorx.CodeSuccess:
		return http.StatusOK
	case errorx.CodeInvalidParams:
		return http.StatusBadRequest
	case errorx.CodeServiceUnavailable:
		return http.StatusServiceUnavailable
	default:
		// 其他业务错误返回 200，但 code 非 0
		return http.StatusOK
	}
}

// HandleError 统一错误处理（用于 handler 层）
// 用法: response.HandleError(w, err, func() { response.Success(w, data) })
func HandleError(w http.ResponseWriter, err error, successFn func()) {
	if err != nil {
		Fail(w, err)
		return
	}
	successFn()
}

// Error 错误响应（简化版，用于中间件）
// code: HTTP状态码或业务码
// message: 错误消息
func Error(w http.ResponseWriter, code int, message string) {
	resp := &Response{
		Code:    code,
		Message: message,
	}
	httpx.WriteJson(w, code, resp)
}
