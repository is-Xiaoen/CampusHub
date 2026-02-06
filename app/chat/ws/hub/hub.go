package hub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"

	"activity-platform/app/chat/ws/internal/types"
	"activity-platform/common/messaging"
)

var (
	ErrSendBufferFull = errors.New("发送缓冲区已满")
	ErrUserNotOnline  = errors.New("用户不在线")
)

// Hub 连接管理中心
type Hub struct {
	// 已注册的客户端
	clients map[string]*Client // userID -> Client

	// 群聊订阅 (groupID -> clients)
	groups map[string]map[*Client]bool

	// 注册请求
	register chan *Client

	// 注销请求
	unregister chan *Client

	// 消息处理器
	messageHandler MessageHandler

	// 消息中间件客户端
	messagingClient *messaging.Client

	// Redis 客户端（用于存储用户状态）
	redisClient *redis.Client

	mu sync.RWMutex
}

// MessageHandler 消息处理器接口
type MessageHandler interface {
	HandleAuth(client *Client, msg *types.WSMessage) error
	HandleSendMessage(client *Client, msg *types.WSMessage) error
	HandleJoinGroup(client *Client, msg *types.WSMessage) error
	HandleLeaveGroup(client *Client, msg *types.WSMessage) error
	HandleMarkRead(client *Client, msg *types.WSMessage) error
}

// NewHub 创建新的 Hub
func NewHub(handler MessageHandler, messagingClient *messaging.Client, redisClient *redis.Client) *Hub {
	return &Hub{
		clients:         make(map[string]*Client),
		groups:          make(map[string]map[*Client]bool),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		messageHandler:  handler,
		messagingClient: messagingClient,
		redisClient:     redisClient,
	}
}

// Run 运行 Hub
func (h *Hub) Run(ctx context.Context) {
	// 订阅消息中间件的消息
	go h.subscribeMessages(ctx)

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case <-ctx.Done():
			logx.Info("Hub 正在关闭")
			return
		}
	}
}

// Register 获取注册通道
func (h *Hub) Register() chan<- *Client {
	return h.register
}

// registerClient 注册客户端
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.userID != "" {
		// 如果用户已有连接，关闭旧连接
		if oldClient, exists := h.clients[client.userID]; exists {
			close(oldClient.send)
		}
		h.clients[client.userID] = client

		// 更新用户在线状态到 Redis
		h.updateUserStatus(client.userID, true)

		logx.Infof("用户 %s 已连接", client.userID)
	}
}

// unregisterClient 注销客户端
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.userID != "" {
		if _, exists := h.clients[client.userID]; exists {
			delete(h.clients, client.userID)
			close(client.send)

			// 从所有群聊中移除
			for groupID := range client.groups {
				if clients, ok := h.groups[groupID]; ok {
					delete(clients, client)
					if len(clients) == 0 {
						delete(h.groups, groupID)
					}
				}
			}

			// 更新用户离线状态到 Redis
			h.updateUserStatus(client.userID, false)

			logx.Infof("用户 %s 已断开连接", client.userID)
		}
	}
}

// handleClientMessage 处理客户端消息
func (h *Hub) handleClientMessage(client *Client, msg *types.WSMessage) {
	var err error

	switch msg.Type {
	case types.TypePing:
		// 心跳响应
		client.SendMessage(&types.WSMessage{
			Type:      types.TypePong,
			Timestamp: time.Now().Unix(),
		})
		return

	case types.TypeAuth:
		err = h.messageHandler.HandleAuth(client, msg)

	case types.TypeSendMessage:
		err = h.messageHandler.HandleSendMessage(client, msg)

	case types.TypeJoinGroup:
		err = h.messageHandler.HandleJoinGroup(client, msg)

	case types.TypeLeaveGroup:
		err = h.messageHandler.HandleLeaveGroup(client, msg)

	case types.TypeMarkRead:
		err = h.messageHandler.HandleMarkRead(client, msg)

	default:
		client.SendError(400, "未知的消息类型")
		return
	}

	if err != nil {
		logx.Errorf("处理消息错误: %v", err)
		client.SendError(500, err.Error())
	}
}

// BroadcastToGroup 向群聊广播消息
func (h *Hub) BroadcastToGroup(groupID string, msg *types.WSMessage) {
	h.mu.RLock()
	clients, ok := h.groups[groupID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	for client := range clients {
		if client.IsInGroup(groupID) {
			client.SendMessage(msg)
		}
	}
}

// SendToUser 发送消息给指定用户
func (h *Hub) SendToUser(userID string, msg *types.WSMessage) error {
	h.mu.RLock()
	client, ok := h.clients[userID]
	h.mu.RUnlock()

	if !ok {
		return ErrUserNotOnline
	}

	return client.SendMessage(msg)
}

// subscribeMessages 订阅消息中间件的消息
func (h *Hub) subscribeMessages(ctx context.Context) {
	// 订阅群聊消息
	h.messagingClient.Subscribe("chat.message.new", "ws-message-handler", func(msg *message.Message) error {
		var msgData types.NewMessageData
		if err := json.Unmarshal(msg.Payload, &msgData); err != nil {
			return messaging.NewNonRetryableError(err)
		}

		// 广播到群聊
		wsMsg := &types.WSMessage{
			Type:      types.TypeNewMessage,
			MessageID: msgData.MessageID,
			Timestamp: msgData.CreatedAt,
			Data:      json.RawMessage(msg.Payload),
		}
		h.BroadcastToGroup(msgData.GroupID, wsMsg)

		return nil
	})

	// 订阅系统通知
	h.messagingClient.Subscribe("chat.notification.new", "ws-notification-handler", func(msg *message.Message) error {
		var notifData types.NotificationData
		if err := json.Unmarshal(msg.Payload, &notifData); err != nil {
			return messaging.NewNonRetryableError(err)
		}

		// 从消息元数据中获取目标用户ID
		userID := msg.Metadata.Get("user_id")
		if userID == "" {
			return messaging.NewNonRetryableError(errors.New("消息元数据中缺少 user_id"))
		}

		// 发送给指定用户
		wsMsg := &types.WSMessage{
			Type:      types.TypeNotification,
			MessageID: notifData.NotificationID,
			Timestamp: notifData.CreatedAt,
			Data:      json.RawMessage(msg.Payload),
		}
		if err := h.SendToUser(userID, wsMsg); err != nil {
			// 用户不在线，不算错误
			if err == ErrUserNotOnline {
				logx.Infof("用户 %s 不在线，跳过通知", userID)
				return nil
			}
			return err
		}

		return nil
	})

	// 启动消息订阅
	if err := h.messagingClient.Run(ctx); err != nil {
		logx.Errorf("消息中间件客户端停止: %v", err)
	}
}

// AddClientToGroup 将客户端添加到群聊
func (h *Hub) AddClientToGroup(client *Client, groupID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.groups[groupID]; !ok {
		h.groups[groupID] = make(map[*Client]bool)
	}
	h.groups[groupID][client] = true
	client.JoinGroup(groupID)
}

// RemoveClientFromGroup 将客户端从群聊移除
func (h *Hub) RemoveClientFromGroup(client *Client, groupID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.groups[groupID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.groups, groupID)
		}
	}
	client.LeaveGroup(groupID)
}

// GetOnlineUserCount 获取在线用户数
func (h *Hub) GetOnlineUserCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// updateUserStatus 更新用户在线状态到 Redis
func (h *Hub) updateUserStatus(userID string, isOnline bool) {
	ctx := context.Background()
	key := fmt.Sprintf("user:status:%s", userID)
	now := time.Now().Unix()

	data := map[string]interface{}{
		"is_online": isOnline,
		"last_seen": now,
	}

	if isOnline {
		data["last_online_at"] = now
	} else {
		data["last_offline_at"] = now
	}

	// 存储到 Redis，设置 30 天过期
	if err := h.redisClient.HMSet(ctx, key, data).Err(); err != nil {
		logx.Errorf("更新用户状态失败: %v", err)
		return
	}
	h.redisClient.Expire(ctx, key, 30*24*time.Hour)
}

// GetUserStatus 获取用户状态
func (h *Hub) GetUserStatus(userID string) (map[string]interface{}, error) {
	ctx := context.Background()
	key := fmt.Sprintf("user:status:%s", userID)

	result, err := h.redisClient.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, errors.New("用户状态不存在")
	}

	return map[string]interface{}{
		"is_online":       result["is_online"] == "true",
		"last_seen":       result["last_seen"],
		"last_online_at":  result["last_online_at"],
		"last_offline_at": result["last_offline_at"],
	}, nil
}
