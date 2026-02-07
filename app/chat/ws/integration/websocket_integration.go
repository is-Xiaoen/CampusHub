package integration

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"

	"activity-platform/app/chat/rpc/chatservice"
	"activity-platform/app/chat/ws/hub"
	"activity-platform/app/chat/ws/internal/handler"
	"activity-platform/app/chat/ws/internal/logic"
	"activity-platform/app/chat/ws/internal/queue"
	"activity-platform/app/chat/ws/internal/svc"
	"activity-platform/common/messaging"
)

// WebSocketService WebSocket 服务封装
type WebSocketService struct {
	Hub             *hub.Hub
	SaveQueue       *queue.SaveQueue
	MessagingClient *messaging.Client
	RedisClient     *redis.Client
	serviceContext  *svc.ServiceContext
}

// WebSocketConfig WebSocket 配置
type WebSocketConfig struct {
	RedisHost         string
	RedisPass         string
	RedisDB           int
	ChatRpcConfig     zrpc.RpcClientConf
	JwtSecret         string
	MaxConnections    int
	ReadTimeout       int
	WriteTimeout      int
	HeartbeatInterval int
}

// NewWebSocketService 创建 WebSocket 服务
func NewWebSocketService(config WebSocketConfig) (*WebSocketService, error) {
	logx.Info("正在初始化 WebSocket 服务...")

	// 创建 Redis 客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisHost,
		Password: config.RedisPass,
		DB:       config.RedisDB,
	})

	// 创建 RPC 客户端
	chatRpc := chatservice.NewChatService(zrpc.MustNewClient(config.ChatRpcConfig))

	// 创建消息中间件客户端
	messagingConfig := messaging.Config{
		Redis: messaging.RedisConfig{
			Addr:     config.RedisHost,
			Password: config.RedisPass,
			DB:       config.RedisDB,
		},
		ServiceName:   "chat-api-websocket",
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
		logx.Errorf("创建消息中间件客户端失败: %v", err)
		return nil, err
	}

	// 创建消息保存队列
	saveQueue := queue.NewSaveQueue(chatRpc, 10)

	// 创建服务上下文
	serviceContext := &svc.ServiceContext{
		ChatRpc:         chatRpc,
		MessagingClient: messagingClient,
		JwtAuth:         svc.NewJwtAuth(config.JwtSecret),
		RedisClient:     redisClient,
		SaveQueue:       saveQueue,
	}

	// 创建消息处理器
	messageHandler := logic.NewMessageLogic(context.Background(), serviceContext)

	// 创建 Hub
	wsHub := hub.NewHub(messageHandler, messagingClient, redisClient)

	logx.Info("WebSocket 服务初始化完成")

	return &WebSocketService{
		Hub:             wsHub,
		SaveQueue:       saveQueue,
		MessagingClient: messagingClient,
		RedisClient:     redisClient,
		serviceContext:  serviceContext,
	}, nil
}

// Start 启动 WebSocket 服务
func (s *WebSocketService) Start(ctx context.Context) {
	go s.Hub.Run(ctx)
}

// GetWebSocketHandler 获取 WebSocket HTTP 处理器
func (s *WebSocketService) GetWebSocketHandler() http.HandlerFunc {
	return handler.WebSocketHandler(s.serviceContext, s.Hub)
}

// GetHealthHandler 获取健康检查处理器
func (s *WebSocketService) GetHealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("WebSocket OK"))
	}
}

// GetStatsHandler 获取统计信息处理器
func (s *WebSocketService) GetStatsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		count := s.Hub.GetOnlineUserCount()
		fmt.Fprintf(w, `{"online_users":%d}`, count)
	}
}

// Close 关闭 WebSocket 服务
func (s *WebSocketService) Close() error {
	logx.Info("正在关闭 WebSocket 服务...")

	if s.SaveQueue != nil {
		logx.Info("等待消息保存队列处理完成...")
		s.SaveQueue.Stop()
	}

	if s.MessagingClient != nil {
		if err := s.MessagingClient.Close(); err != nil {
			logx.Errorf("关闭消息中间件客户端失败: %v", err)
			return err
		}
	}

	if s.RedisClient != nil {
		if err := s.RedisClient.Close(); err != nil {
			logx.Errorf("关闭 Redis 客户端失败: %v", err)
			return err
		}
	}

	logx.Info("WebSocket 服务已关闭")
	return nil
}
