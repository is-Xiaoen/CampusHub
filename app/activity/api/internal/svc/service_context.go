// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"activity-platform/app/activity/api/internal/config"
	"activity-platform/app/activity/api/internal/middleware"
	"activity-platform/app/activity/rpc/activityservice"

	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config

	// 中间件
	AdminAuth rest.Middleware

	// RPC 客户端
	ActivityRpc activityservice.ActivityService
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化 Activity RPC 客户端
	activityRpcClient := zrpc.MustNewClient(c.ActivityRpc)

	return &ServiceContext{
		Config:      c,
		AdminAuth:   middleware.NewAdminAuthMiddleware().Handle,
		ActivityRpc: activityservice.NewActivityService(activityRpcClient),
	}
}
