package svc

import (
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/zrpc"

	"activity-platform/app/chat/rpc/chatservice"
	"activity-platform/app/chat/ws/internal/config"
	"activity-platform/app/chat/ws/internal/queue"
	"activity-platform/common/messaging"
)

// ServiceContext 服务上下文
type ServiceContext struct {
	Config          config.Config
	ChatRpc         chatservice.ChatService
	MessagingClient *messaging.Client
	JwtAuth         *JwtAuth
	RedisClient     *redis.Client
	SaveQueue       *queue.SaveQueue // 新增：消息保存队列
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	// 创建 RPC 客户端
	chatRpc := chatservice.NewChatService(zrpc.MustNewClient(c.ChatRpc))

	// 创建 Redis 客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr:     c.Redis.Host,
		Password: c.Redis.Pass,
		DB:       c.Redis.DB,
	})

	// 创建消息中间件客户端
	messagingConfig := messaging.Config{
		Redis: messaging.RedisConfig{
			Addr:     c.Redis.Host,
			Password: c.Redis.Pass,
			DB:       c.Redis.DB,
		},
		ServiceName:   "websocket-service",
		EnableMetrics: true,
		EnableGoZero:  true,
		RetryConfig: messaging.RetryConfig{
			MaxRetries:      3,
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     10 * time.Second,
			Multiplier:      2.0,
		},
	}

	messagingClient, err := messaging.NewClient(messagingConfig)
	if err != nil {
		panic(err)
	}

	// 创建消息保存队列（10 个工作协程）
	saveQueue := queue.NewSaveQueue(chatRpc, 10)

	return &ServiceContext{
		Config:          c,
		ChatRpc:         chatRpc,
		MessagingClient: messagingClient,
		JwtAuth:         NewJwtAuth(c.Auth.AccessSecret),
		RedisClient:     redisClient,
		SaveQueue:       saveQueue, // 新增
	}
}
