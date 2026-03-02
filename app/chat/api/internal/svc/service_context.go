// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"activity-platform/app/activity/rpc/activityservice"
	"activity-platform/app/chat/api/internal/config"
	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config

	// Chat RPC 客户端
	ChatRpc chat.ChatServiceClient

	// User RPC 客户端
	UserRpc pb.UserBasicServiceClient

	// Activity RPC 客户端（弱依赖，用于获取活动封面图）
	ActivityRpc activityservice.ActivityService

	// Redis 客户端（用于获取用户状态）
	Redis *redis.Redis
}

func NewServiceContext(c config.Config) *ServiceContext {
	svcCtx := &ServiceContext{
		Config:  c,
		ChatRpc: chat.NewChatServiceClient(zrpc.MustNewClient(c.ChatRpc).Conn()),
		UserRpc: pb.NewUserBasicServiceClient(zrpc.MustNewClient(c.UserRpc).Conn()),
		Redis:   redis.MustNewRedis(c.Redis),
	}

	// 初始化 Activity RPC 客户端（可选，失败不影响服务启动）
	if c.ActivityRpc.Etcd.Key != "" {
		activityRpcClient, err := zrpc.NewClient(c.ActivityRpc)
		if err != nil {
			logx.Errorf("Activity RPC 连接失败（非致命）: %v", err)
		} else {
			svcCtx.ActivityRpc = activityservice.NewActivityService(activityRpcClient)
			logx.Info("Activity RPC 连接初始化成功")
		}
	}

	return svcCtx
}
