// ============================================================================
// 路由注册
// ============================================================================
//
// 功能说明：
//   集中管理所有 HTTP 路由，包括：
//   - 公开路由（无需认证）
//   - 认证路由（需要 JWT Token）
//   - 中间件应用顺序
//
// 路由命名规范：
//   - RESTful 风格
//   - 资源名使用复数：/users, /activities
//   - 动作使用 HTTP 方法：GET/POST/PUT/DELETE
//
// 中间件执行顺序：
//   CORS -> RequestID -> RateLimit -> [Auth] -> Handler
//
// ============================================================================

package handler

import (
	"net/http"

	"activity-platform/app/gateway/api/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

// RegisterHandlers 注册所有路由
func RegisterHandlers(server *rest.Server, ctx *svc.ServiceContext) {
	// ==================== 全局中间件 ====================
	// 按执行顺序添加：CORS -> RequestID -> RateLimit
	server.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return ctx.CorsMiddleware.Handle(next)
	})
	server.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return ctx.RequestIDMiddleware.Handle(next)
	})
	server.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return ctx.RateLimitMiddleware.Handle(next)
	})

	// ==================== 公开路由（无需认证） ====================
	server.AddRoutes(
		[]rest.Route{
			// 健康检查
			{
				Method:  http.MethodGet,
				Path:    "/health",
				Handler: HealthHandler(ctx),
			},
			// 服务信息
			{
				Method:  http.MethodGet,
				Path:    "/",
				Handler: IndexHandler(ctx),
			},
		},
	)

	// ==================== 用户模块路由（部分公开） ====================
	// TODO(杨春路): User RPC 实现后添加具体路由
	// 示例路由结构（当前被注释，实现后取消注释）：
	//
	// server.AddRoutes(
	//     []rest.Route{
	//         { Method: http.MethodPost, Path: "/api/v1/user/register", Handler: UserRegisterHandler(ctx) },
	//         { Method: http.MethodPost, Path: "/api/v1/user/login", Handler: UserLoginHandler(ctx) },
	//     },
	// )

	// ==================== 需要认证的路由 ====================
	// 使用 rest.WithJwt 或自定义 Auth 中间件
	// TODO: 各模块 RPC 实现后，添加需要认证的路由
	//
	// 示例：
	// server.AddRoutes(
	//     rest.WithMiddlewares(
	//         []rest.Middleware{authMiddleware},
	//         []rest.Route{
	//             { Method: http.MethodGet, Path: "/api/v1/user/profile", Handler: UserProfileHandler(ctx) },
	//         }...,
	//     ),
	// )

	// ==================== 活动模块路由 ====================
	// TODO(马肖阳): Activity RPC 实现后添加具体路由
	//
	// 公开路由：
	// - GET  /api/v1/activities       活动列表
	// - GET  /api/v1/activities/:id   活动详情
	//
	// 需认证路由：
	// - POST   /api/v1/activities           创建活动
	// - PUT    /api/v1/activities/:id       更新活动
	// - DELETE /api/v1/activities/:id       删除活动
	// - POST   /api/v1/activities/:id/join  报名活动
	// - POST   /api/v1/activities/:id/checkin 签到

	// ==================== 聊天模块路由 ====================
	// TODO(马华恩): Chat RPC 实现后添加具体路由
	//
	// - GET /api/v1/chat/ws  WebSocket 连接
	// - GET /api/v1/chat/rooms/:id/messages 消息历史
}
