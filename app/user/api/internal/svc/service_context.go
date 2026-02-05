// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"activity-platform/app/user/api/internal/config"
	"activity-platform/app/user/api/internal/middleware"
	"activity-platform/app/user/rpc/client/creditservice"
	"activity-platform/app/user/rpc/client/userbasicservice"
	"activity-platform/app/user/rpc/client/verifyservice"

	"github.com/go-redis/redis/v8"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config             config.Config
	UserRoleMiddleware rest.Middleware

	Redis *redis.Client
	// CreditServiceRpc 信用分服务 RPC 客户端
	CreditServiceRpc creditservice.CreditService

	// VerifyServiceRpc 认证服务 RPC 客户端
	VerifyServiceRpc verifyservice.VerifyService

	// UserBasicServiceRpc 用户基础服务 RPC 客户端（登录、注册、忘记密码等）
	UserBasicServiceRpc userbasicservice.UserBasicService
}

func NewServiceContext(c config.Config) *ServiceContext {

	// 创建 User RPC 客户端连接
	userRpcClient := zrpc.MustNewClient(c.UserRpc)

	// 初始化 Redis 客户端
	rdb := initRedis(c)

	return &ServiceContext{
		Config:             c,
		UserRoleMiddleware: middleware.NewUserRoleMiddleware().Handle,
		Redis:              rdb,
		// 初始化 RPC 客户端
		CreditServiceRpc:    creditservice.NewCreditService(userRpcClient),
		VerifyServiceRpc:    verifyservice.NewVerifyService(userRpcClient),
		UserBasicServiceRpc: userbasicservice.NewUserBasicService(userRpcClient),
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
