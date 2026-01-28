// ============================================================================
// 服务上下文（Service Context）
// ============================================================================
//
// 功能说明：
//   ServiceContext 是网关的核心组件，负责初始化和管理：
//   - 配置信息
//   - 中间件实例
//   - RPC 客户端连接（服务发现）
//
// 设计原则：
//   - 所有依赖在启动时初始化，避免运行时创建
//   - RPC 客户端通过 Etcd 自动发现，无需硬编码地址
//   - 中间件实例复用，避免每次请求重复创建
//
// ============================================================================

package svc

import (
	"activity-platform/app/gateway/api/internal/config"
	"activity-platform/app/gateway/api/internal/middleware"
)

// ServiceContext API 网关服务上下文
type ServiceContext struct {
	Config config.Config

	// ==================== 中间件 ====================
	CorsMiddleware      *middleware.CorsMiddleware
	RateLimitMiddleware *middleware.RateLimitMiddleware
	RequestIDMiddleware *middleware.RequestIDMiddleware
	AuthMiddleware      *middleware.AuthMiddleware

	// ==================== RPC 客户端 ====================
	// TODO(杨春路): User RPC 服务实现后取消注释
	// UserRpc userrpc.UserClient

	// TODO(马肖阳): Activity RPC 服务实现后取消注释
	// ActivityRpc activityrpc.ActivityClient

	// TODO(马华恩): Chat RPC 服务实现后取消注释
	// ChatRpc chatrpc.ChatClient
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,

		// 初始化中间件
		CorsMiddleware: middleware.NewCorsMiddleware(
			c.Cors.AllowOrigins,
			c.Cors.AllowMethods,
			c.Cors.AllowHeaders,
		),
		RateLimitMiddleware: middleware.NewRateLimitMiddleware(
			float64(c.RateLimit.Rate),  // 全局限流
			c.RateLimit.Burst,          // 全局突发
			10,                         // 单 IP 每秒 10 次
			20,                         // 单 IP 突发 20 次
		),
		RequestIDMiddleware: middleware.NewRequestIDMiddleware(),
		AuthMiddleware:      middleware.NewAuthMiddleware(c.Auth.AccessSecret),

		// ==================== RPC 客户端初始化 ====================
		// 说明：当对应 RPC 服务实现完成后，取消注释并导入对应的包
		//
		// UserRpc: userrpc.NewUser(zrpc.MustNewClient(c.UserRpc)),
		// ActivityRpc: activityrpc.NewActivity(zrpc.MustNewClient(c.ActivityRpc)),
		// ChatRpc: chatrpc.NewChat(zrpc.MustNewClient(c.ChatRpc)),
	}
}
