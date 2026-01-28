package middleware

import (
	"net/http"

	"activity-platform/common/ctxdata"

	"github.com/google/uuid"
)

// RequestIDMiddleware 请求ID中间件
// 为每个请求生成唯一ID，用于链路追踪和日志关联
type RequestIDMiddleware struct{}

// NewRequestIDMiddleware 创建请求ID中间件
func NewRequestIDMiddleware() *RequestIDMiddleware {
	return &RequestIDMiddleware{}
}

// Handle 处理请求ID
func (m *RequestIDMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 优先从请求头获取，支持上游传递
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 获取追踪ID
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = requestID // 如果没有追踪ID，使用请求ID
		}

		// 注入上下文
		ctx := r.Context()
		ctx = ctxdata.WithRequestID(ctx, requestID)
		ctx = ctxdata.WithTraceID(ctx, traceID)

		// 设置响应头
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
