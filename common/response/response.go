package response

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// SetupGlobalErrorHandler 设置全局错误处理器
// 必须在 server.Start() 之前调用
// 这样 goctl 生成的 handler 中的 httpx.ErrorCtx 也会使用统一格式
func SetupGlobalErrorHandler() {
	httpx.SetErrorHandler(func(err error) (int, interface{}) {
		bizErr := parseError(context.Background(), err)
		return getHttpStatus(bizErr.Code), &Response{
			Code:    bizErr.Code,
			Message: bizErr.Message,
		}
	})

	httpx.SetErrorHandlerCtx(func(ctx context.Context, err error) (int, interface{}) {
		bizErr := parseError(ctx, err)
		return getHttpStatus(bizErr.Code), &Response{
			Code:    bizErr.Code,
			Message: bizErr.Message,
		}
	})
}

// parseError 智能解析错误，区分参数错误和内部错误
//
// go-zero 的 httpx.Parse 在 JSON 解析失败时会抛出原生 Go 错误，
// 这些错误本质上是客户端参数问题，应该返回 1001 而不是 1000。
func parseError(ctx context.Context, err error) *errorx.BizError {
	// 1. 先走原有的 BizError / gRPC Status 解析
	bizErr := errorx.FromError(err)
	if bizErr.Code != errorx.CodeInternalError {
		return bizErr
	}

	// 2. 到这里说明 FromError 没能识别，被兜底成了 1000
	//    检查是否是参数解析类错误，如果是，改为 1001 并返回具体原因
	if paramErr := tryParseAsParamError(err); paramErr != nil {
		logx.WithContext(ctx).Infof("[ParamError] %v", err)
		return paramErr
	}

	// 3. 真正的内部错误，打日志方便排查
	logx.WithContext(ctx).Errorf("[InternalError] %+v", err)
	return bizErr
}

// tryParseAsParamError 尝试将错误识别为参数错误
// 返回 nil 表示不是参数错误
func tryParseAsParamError(err error) *errorx.BizError {
	if err == nil {
		return nil
	}
	msg := err.Error()

	// JSON 类型不匹配：比如传了字符串 "2026-01-30" 但期望 int64
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return errorx.ErrInvalidParams(
			"字段 " + typeErr.Field + " 类型错误，期望 " + typeErr.Type.String(),
		)
	}

	// JSON 语法错误：比如 JSON 格式不合法
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return errorx.ErrInvalidParams("JSON 格式错误")
	}

	// 请求体为空
	if errors.Is(err, io.EOF) {
		return errorx.ErrInvalidParams("请求体不能为空")
	}

	// go-zero httpx.Parse 的常见错误信息模式匹配
	lower := strings.ToLower(msg)

	// 缺少必填字段
	if strings.Contains(lower, "is not set") || strings.Contains(lower, "missing") {
		return errorx.ErrInvalidParams(msg)
	}

	// 字段值不合法
	if strings.Contains(lower, "invalid") || strings.Contains(lower, "not valid") {
		return errorx.ErrInvalidParams(msg)
	}

	// 类型转换失败
	if strings.Contains(lower, "unmarshal") || strings.Contains(lower, "cannot parse") ||
		strings.Contains(lower, "type mismatch") || strings.Contains(lower, "strconv") {
		return errorx.ErrInvalidParams("参数类型错误: " + msg)
	}

	return nil
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
