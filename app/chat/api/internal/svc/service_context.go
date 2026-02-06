// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"activity-platform/app/chat/api/internal/config"
	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config

	// Chat RPC 客户端
	ChatRpc chat.ChatServiceClient

	// User RPC 客户端
	UserRpc pb.UserBasicServiceClient

	// Redis 客户端（用于获取用户状态）
	Redis *redis.Redis
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:  c,
		ChatRpc: chat.NewChatServiceClient(zrpc.MustNewClient(c.ChatRpc).Conn()),
		UserRpc: pb.NewUserBasicServiceClient(zrpc.MustNewClient(c.UserRpc).Conn()),
		Redis:   redis.MustNewRedis(c.Redis),
	}
}
