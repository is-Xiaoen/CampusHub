package middleware

import (
	"net/http"
	"strings"
)

// CorsMiddleware CORS 跨域中间件
type CorsMiddleware struct {
	allowOrigins []string
	allowMethods []string
	allowHeaders []string
}

// NewCorsMiddleware 创建 CORS 中间件
func NewCorsMiddleware(origins, methods, headers []string) *CorsMiddleware {
	return &CorsMiddleware{
		allowOrigins: origins,
		allowMethods: methods,
		allowHeaders: headers,
	}
}

// Handle 处理 CORS
func (m *CorsMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// 检查是否允许该来源
		if m.isOriginAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", strings.Join(m.allowMethods, ", "))
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(m.allowHeaders, ", "))
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// 预检请求直接返回
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// isOriginAllowed 检查来源是否被允许
func (m *CorsMiddleware) isOriginAllowed(origin string) bool {
	for _, allowed := range m.allowOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}
