/**
 * @projectName: CampusHub
 * @package: svc
 * @className: ServiceContext
 * @author: lijunqi
 * @description: User API 服务上下文，负责依赖注入
 * @date: 2026-01-30
 * @version: 1.0
 */

package svc

import (
	"activity-platform/app/user/api/internal/config"
	"activity-platform/app/user/rpc/client/creditservice"
	"activity-platform/app/user/rpc/client/verifyservice"

	"github.com/zeromicro/go-zero/zrpc"
)

// ServiceContext API 服务上下文
// 包含所有依赖：配置、RPC 客户端等
type ServiceContext struct {
	// Config 服务配置
	Config config.Config

	// ==================== User RPC 客户端 ====================

	// CreditServiceRpc 信用分服务 RPC 客户端
	CreditServiceRpc creditservice.CreditService

	// VerifyServiceRpc 认证服务 RPC 客户端
	VerifyServiceRpc verifyservice.VerifyService
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	// 创建 User RPC 客户端连接
	userRpcClient := zrpc.MustNewClient(c.UserRpc)

	return &ServiceContext{
		Config: c,

		// 初始化 RPC 客户端
		CreditServiceRpc: creditservice.NewCreditService(userRpcClient),
		VerifyServiceRpc: verifyservice.NewVerifyService(userRpcClient),
	}
}
