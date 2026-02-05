// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"activity-platform/app/chat/api/internal/config"
	"activity-platform/app/chat/rpc/chat"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config

	// Chat RPC 客户端
	ChatRpc chat.ChatServiceClient

	// Redis 客户端（用于获取用户状态）
	Redis *redis.Redis
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:  c,
		ChatRpc: chat.NewChatServiceClient(zrpc.MustNewClient(c.ChatRpc).Conn()),
		Redis:   redis.MustNewRedis(c.Redis),
	}
}
