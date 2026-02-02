/**
 * @projectName: CampusHub
 * @package: svc
 * @className: ServiceContext
 * @author: lijunqi
 * @description: User API 服务上下文，负责依赖注入
 * @date: 2026-01-30
 * @version: 1.1
 */

package svc

import (
	"activity-platform/app/user/api/internal/config"
	"activity-platform/app/user/rpc/client/creditservice"
	"activity-platform/app/user/rpc/client/verifyservice"

	"github.com/go-redis/redis/v8"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

// ServiceContext API 服务上下文
// 包含所有依赖：配置、RPC 客户端等
// 注意：OCR 功能已迁移到 RPC 层，API 层不再直接调用 OCR
type ServiceContext struct {
	// Config 服务配置
	Config config.Config

	// ==================== Redis 客户端 ====================

	// Redis Redis客户端
	Redis *redis.Client

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

	// 初始化 Redis 客户端
	rdb := initRedis(c)

	return &ServiceContext{
		Config: c,
		Redis:  rdb,

		// 初始化 RPC 客户端
		CreditServiceRpc: creditservice.NewCreditService(userRpcClient),
		VerifyServiceRpc: verifyservice.NewVerifyService(userRpcClient),
	}
}

// initRedis 初始化Redis客户端
func initRedis(c config.Config) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     c.BizRedis.Host,
		Password: c.BizRedis.Pass,
		DB:       0,
	})
	logx.Info("Redis连接初始化成功")
	return rdb
}
