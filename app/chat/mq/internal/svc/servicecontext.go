package svc

import (
	"activity-platform/app/chat/mq/internal/config"
	"activity-platform/app/chat/rpc/chat"
	"activity-platform/common/messaging"
	"fmt"
	"log"

	"github.com/zeromicro/go-zero/zrpc"
)

// ServiceContext 消费者服务上下文
type ServiceContext struct {
	Config config.Config

	// Chat RPC 客户端
	ChatRpc chat.ChatServiceClient

	// 消息中间件客户端
	MsgClient *messaging.Client
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化 Chat RPC 客户端
	chatRpc := chat.NewChatServiceClient(zrpc.MustNewClient(c.ChatRpc).Conn())

	// 初始化消息中间件客户端
	msgClient, err := initMessaging(c)
	if err != nil {
		log.Fatalf("消息中间件初始化失败: %v", err)
	}

	return &ServiceContext{
		Config:    c,
		ChatRpc:   chatRpc,
		MsgClient: msgClient,
	}
}

// initMessaging 初始化消息中间件
func initMessaging(c config.Config) (*messaging.Client, error) {
	// 构建 messaging 配置
	msgConfig := messaging.Config{
		Redis: messaging.RedisConfig{
			Addr:     c.Redis.Host,
			Password: c.Redis.Pass,
			DB:       c.Redis.DB,
		},
		ServiceName:   c.Messaging.ServiceName,
		EnableMetrics: c.Messaging.EnableMetrics,
		EnableGoZero:  c.Messaging.EnableGoZero,
		RetryConfig: messaging.RetryConfig{
			MaxRetries:      c.Messaging.Retry.MaxRetries,
			InitialInterval: c.Messaging.Retry.InitialInterval,
			MaxInterval:     c.Messaging.Retry.MaxInterval,
			Multiplier:      c.Messaging.Retry.Multiplier,
		},
		DLQConfig: messaging.DLQConfig{
			Enabled:     true,
			TopicSuffix: ".dlq",
		},
	}

	// 创建消息客户端
	client, err := messaging.NewClient(msgConfig)
	if err != nil {
		return nil, fmt.Errorf("创建消息客户端失败: %w", err)
	}

	return client, nil
}
