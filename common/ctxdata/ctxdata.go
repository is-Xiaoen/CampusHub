package ctxdata

import (
	"context"
	"encoding/json"
	"strconv"
)

// 定义上下文 key 类型，避免冲突
type contextKey string

const (
	// CtxKeyUserID 用户ID在上下文中的key
	CtxKeyUserID contextKey = "userId"
	// CtxKeyPhone 手机号在上下文中的key
	CtxKeyPhone contextKey = "phone"
	// CtxKeyRequestID 请求ID
	CtxKeyRequestID contextKey = "requestId"
	// CtxKeyTraceID 追踪ID
	CtxKeyTraceID contextKey = "traceId"
)

// GetUserIDFromCtx 从上下文中获取用户ID
// go-zero 会将 JWT payload 中的字段注入到 context 中
func GetUserIDFromCtx(ctx context.Context) int64 {
	// 先尝试从自定义 key 获取
	if val := ctx.Value(CtxKeyUserID); val != nil {
		return parseToInt64(val)
	}

	// 兼容 go-zero 的 JWT 解析方式（字符串 key）
	if val := ctx.Value("userId"); val != nil {
		return parseToInt64(val)
	}

	return 0
}

// GetPhoneFromCtx 从上下文中获取手机号
func GetPhoneFromCtx(ctx context.Context) string {
	if val := ctx.Value(CtxKeyPhone); val != nil {
		if phone, ok := val.(string); ok {
			return phone
		}
	}

	if val := ctx.Value("phone"); val != nil {
		if phone, ok := val.(string); ok {
			return phone
		}
	}

	return ""
}

// GetRequestIDFromCtx 从上下文中获取请求ID
func GetRequestIDFromCtx(ctx context.Context) string {
	if val := ctx.Value(CtxKeyRequestID); val != nil {
		if reqID, ok := val.(string); ok {
			return reqID
		}
	}
	return ""
}

// GetTraceIDFromCtx 从上下文中获取追踪ID
func GetTraceIDFromCtx(ctx context.Context) string {
	if val := ctx.Value(CtxKeyTraceID); val != nil {
		if traceID, ok := val.(string); ok {
			return traceID
		}
	}
	return ""
}

// WithUserID 将用户ID注入上下文
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, CtxKeyUserID, userID)
}

// WithPhone 将手机号注入上下文
func WithPhone(ctx context.Context, phone string) context.Context {
	return context.WithValue(ctx, CtxKeyPhone, phone)
}

// WithRequestID 将请求ID注入上下文
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, CtxKeyRequestID, requestID)
}

// WithTraceID 将追踪ID注入上下文
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, CtxKeyTraceID, traceID)
}

// parseToInt64 将各种类型转换为 int64
func parseToInt64(val interface{}) int64 {
	switch v := val.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i
		}
	case string:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

// CtxData 用于 RPC 调用时传递上下文数据
type CtxData struct {
	UserID    int64  `json:"userId"`
	Phone     string `json:"phone"`
	RequestID string `json:"requestId"`
	TraceID   string `json:"traceId"`
}

// ExtractCtxData 从上下文提取数据结构
func ExtractCtxData(ctx context.Context) *CtxData {
	return &CtxData{
		UserID:    GetUserIDFromCtx(ctx),
		Phone:     GetPhoneFromCtx(ctx),
		RequestID: GetRequestIDFromCtx(ctx),
		TraceID:   GetTraceIDFromCtx(ctx),
	}
}

// InjectCtxData 将数据结构注入上下文
func InjectCtxData(ctx context.Context, data *CtxData) context.Context {
	if data == nil {
		return ctx
	}
	ctx = WithUserID(ctx, data.UserID)
	ctx = WithPhone(ctx, data.Phone)
	ctx = WithRequestID(ctx, data.RequestID)
	ctx = WithTraceID(ctx, data.TraceID)
	return ctx
}
