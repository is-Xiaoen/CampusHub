package middleware

import (
	"context"
	"net/http"
	"strings"

	"activity-platform/common/ctxdata"
	"activity-platform/common/errorx"
	"activity-platform/common/response"
	"activity-platform/common/utils/jwt"
)

// AuthMiddleware JWT 认证中间件
type AuthMiddleware struct {
	jwtConfig *jwt.JwtConfig
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(accessSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		jwtConfig: &jwt.JwtConfig{
			AccessSecret: accessSecret,
		},
	}
}

// Handle 处理认证逻辑
func (m *AuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. 获取 Authorization 头
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.FailWithCode(w, errorx.CodeLoginRequired)
			return
		}

		// 2. 解析 Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.FailWithCode(w, errorx.CodeTokenInvalid)
			return
		}
		tokenString := parts[1]

		// 3. 解析 Token
		claims, err := jwt.ParseAccessToken(m.jwtConfig, tokenString)
		if err != nil {
			if jwt.IsTokenExpired(err) {
				response.FailWithCode(w, errorx.CodeTokenExpired)
				return
			}
			response.FailWithCode(w, errorx.CodeTokenInvalid)
			return
		}

		// 4. 将用户信息注入上下文
		ctx := r.Context()
		ctx = ctxdata.WithUserID(ctx, claims.UserID)
		ctx = ctxdata.WithPhone(ctx, claims.Phone)

		// 5. 同时将用户信息写入 context.Value（兼容 go-zero 风格）
		ctx = context.WithValue(ctx, "userId", claims.UserID)
		ctx = context.WithValue(ctx, "phone", claims.Phone)

		// 6. 继续处理请求
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
