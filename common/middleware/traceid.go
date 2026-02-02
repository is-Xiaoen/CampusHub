package middleware

import (
	"net/http"

	"activity-platform/common/ctxdata"

	"github.com/google/uuid"
)

// TraceIDMiddleware 追踪 ID 中间件
// 自动为每个 HTTP 请求注入 trace_id，用于全链路追踪
//
// 工作流程：
// 1. 从请求头 X-Trace-ID 中获取 trace_id（如果客户端传递）
// 2. 如果没有，自动生成新的 trace_id
// 3. 将 trace_id 注入到 context 中
// 4. 将 trace_id 写入响应头（方便客户端追踪）
//
// 使用方式：
//
//	server.Use(middleware.TraceIDMiddleware)
func TraceIDMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. 从 Header 中获取 trace_id（如果有）
		// 支持多种 Header 名称（兼容不同的客户端）
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = r.Header.Get("X-Request-ID")
		}
		if traceID == "" {
			traceID = r.Header.Get("Trace-ID")
		}

		// 2. 如果没有，生成新的 trace_id
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// 3. 注入到 context
		ctx := ctxdata.WithTraceID(r.Context(), traceID)

		// 4. 将 trace_id 写入响应头（方便客户端追踪）
		w.Header().Set("X-Trace-ID", traceID)

		// 5. 继续处理请求
		next(w, r.WithContext(ctx))
	}
}

// TraceIDHandler 追踪 ID 处理器（go-zero 风格）
// 用于 go-zero 的 rest.Server
//
// 使用方式：
//
//	server := rest.MustNewServer(c.RestConf)
//	server.Use(middleware.TraceIDHandler)
func TraceIDHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. 从 Header 中获取 trace_id
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = r.Header.Get("X-Request-ID")
		}
		if traceID == "" {
			traceID = r.Header.Get("Trace-ID")
		}

		// 2. 如果没有，生成新的 trace_id
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// 3. 注入到 context
		ctx := ctxdata.WithTraceID(r.Context(), traceID)

		// 4. 将 trace_id 写入响应头
		w.Header().Set("X-Trace-ID", traceID)

		// 5. 继续处理请求
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
