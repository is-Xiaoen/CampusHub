# WebSocket 服务 - 实时消息推送实现方案

## 1. 概述

### 1.1 目标
为 CampusHub 项目实现基于 WebSocket 的实时消息推送服务，支持：
- 群聊实时消息推送
- 系统通知实时推送
- 在线状态管理
- 离线消息处理

### 1.2 技术选型
- **WebSocket 框架**: `gorilla/websocket` - Go 生态最成熟的 WebSocket 库
- **消息中间件**: 复用现有的 `common/messaging` (Watermill + Redis Stream)
- **连接管理**: Redis 存储在线状态和连接映射
- **协议格式**: JSON (与现有 API 保持一致)
- **认证方式**: JWT Token (与现有 API 保持一致)

### 1.3 架构设计

```
┌─────────────┐         WebSocket          ┌──────────────────┐
│   客户端     │ ◄─────────────────────────► │  WebSocket 服务  │
│  (前端)     │                             │   (ws-server)    │
└─────────────┘                             └──────────────────┘
                                                      │
                                                      │ 订阅消息
                                                      ▼
                                            ┌──────────────────┐
                                            │  消息中间件       │
                                            │  (Redis Stream)  │
                                            └──────────────────┘
                                                      ▲
                                                      │ 发布消息
                                            ┌──────────────────┐
                                            │   Chat RPC       │
                                            │   (业务服务)     │
                                            └──────────────────┘
```

---

## 2. 目录结构

```
app/chat/
├── ws/                         # WebSocket 服务 (新增)
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go       # WebSocket 配置
│   │   ├── handler/
│   │   │   └── ws_handler.go   # WebSocket 处理器
│   │   ├── logic/
│   │   │   ├── connection_logic.go  # 连接管理逻辑
│   │   │   └── message_logic.go     # 消息处理逻辑
│   │   ├── svc/
│   │   │   └── service_context.go   # 服务上下文
│   │   ├── types/
│   │   │   └── types.go        # 消息类型定义
│   │   └── middleware/
│   │       └── auth.go         # 认证中间件
│   ├── hub/
│   │   ├── hub.go              # 连接管理中心
│   │   ├── client.go           # 客户端连接
│   │   └── manager.go          # 连接管理器
│   ├── etc/
│   │   └── websocket.yaml      # 配置文件
│   └── websocket.go            # 主入口
├── api/
│   └── ...
├── rpc/
│   └── ...
└── model/
    └── ...
```

---

## 3. 核心组件设计

### 3.1 消息协议定义

```go
// app/chat/ws/internal/types/types.go

package types

// MessageType 消息类型
type MessageType string

const (
	// 客户端 -> 服务端
	TypePing         MessageType = "ping"          // 心跳
	TypeAuth         MessageType = "auth"          // 认证
	TypeSendMessage  MessageType = "send_message"  // 发送消息
	TypeJoinGroup    MessageType = "join_group"    // 加入群聊
	TypeLeaveGroup   MessageType = "leave_group"   // 离开群聊
	TypeMarkRead     MessageType = "mark_read"     // 标记已读

	// 服务端 -> 客户端
	TypePong         MessageType = "pong"          // 心跳响应
	TypeAuthSuccess  MessageType = "auth_success"  // 认证成功
	TypeAuthFailed   MessageType = "auth_failed"   // 认证失败
	TypeNewMessage   MessageType = "new_message"   // 新消息
	TypeNotification MessageType = "notification"  // 系统通知
	TypeError        MessageType = "error"         // 错误消息
	TypeAck          MessageType = "ack"           // 消息确认
)

// WSMessage WebSocket 消息结构
type WSMessage struct {
	Type      MessageType     `json:"type"`                // 消息类型
	MessageID string          `json:"message_id"`          // 消息ID (用于去重和确认)
	Timestamp int64           `json:"timestamp"`           // 时间戳
	Data      json.RawMessage `json:"data,omitempty"`      // 消息数据
}

// AuthData 认证数据
type AuthData struct {
	Token string `json:"token"` // JWT Token
}

// SendMessageData 发送消息数据
type SendMessageData struct {
	GroupID   string `json:"group_id"`            // 群聊ID
	MsgType   int32  `json:"msg_type"`            // 消息类型: 1-文字 2-图片
	Content   string `json:"content,omitempty"`   // 文本内容
	ImageURL  string `json:"image_url,omitempty"` // 图片URL
}

// NewMessageData 新消息数据
type NewMessageData struct {
	MessageID  string `json:"message_id"`  // 消息ID
	GroupID    string `json:"group_id"`    // 群聊ID
	SenderID   string `json:"sender_id"`   // 发送者ID
	SenderName string `json:"sender_name"` // 发送者名称
	MsgType    int32  `json:"msg_type"`    // 消息类型
	Content    string `json:"content"`     // 内容
	ImageURL   string `json:"image_url"`   // 图片URL
	CreatedAt  int64  `json:"created_at"`  // 创建时间
}

// NotificationData 通知数据
type NotificationData struct {
	NotificationID string `json:"notification_id"` // 通知ID
	Type           string `json:"type"`            // 通知类型
	Title          string `json:"title"`           // 标题
	Content        string `json:"content"`         // 内容
	CreatedAt      int64  `json:"created_at"`      // 创建时间
}

// ErrorData 错误数据
type ErrorData struct {
	Code    int    `json:"code"`    // 错误码
	Message string `json:"message"` // 错误信息
}

// AckData 确认数据
type AckData struct {
	MessageID string `json:"message_id"` // 确认的消息ID
	Success   bool   `json:"success"`    // 是否成功
}
```

### 3.2 客户端连接管理

```go
// app/chat/ws/hub/client.go

package hub

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"

	"activity-platform/app/chat/ws/internal/types"
)

const (
	// 写入超时时间
	writeWait = 10 * time.Second
	// 心跳间隔
	pongWait = 60 * time.Second
	// Ping 间隔 (必须小于 pongWait)
	pingPeriod = (pongWait * 9) / 10
	// 最大消息大小
	maxMessageSize = 512 * 1024 // 512KB
)

// Client WebSocket 客户端
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	userID   string
	groups   map[string]bool // 用户加入的群聊
	mu       sync.RWMutex
	isAuthed bool
}

// NewClient 创建新客户端
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		groups: make(map[string]bool),
	}
}

// ReadPump 读取消息
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logx.Errorf("websocket error: %v", err)
			}
			break
		}

		// 处理消息
		c.handleMessage(message)
	}
}

// WritePump 写入消息
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 批量写入队列中的消息
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendMessage 发送消息给客户端
func (c *Client) SendMessage(msg *types.WSMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case c.send <- data:
		return nil
	default:
		return ErrSendBufferFull
	}
}

// handleMessage 处理接收到的消息
func (c *Client) handleMessage(message []byte) {
	var msg types.WSMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		c.SendError(400, "invalid message format")
		return
	}

	// 未认证只能发送认证消息
	if !c.isAuthed && msg.Type != types.TypeAuth && msg.Type != types.TypePing {
		c.SendError(401, "unauthorized")
		return
	}

	// 路由到对应的处理器
	c.hub.handleClientMessage(c, &msg)
}

// SendError 发送错误消息
func (c *Client) SendError(code int, message string) {
	errData := types.ErrorData{
		Code:    code,
		Message: message,
	}
	data, _ := json.Marshal(errData)

	msg := &types.WSMessage{
		Type:      types.TypeError,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
	c.SendMessage(msg)
}

// JoinGroup 加入群聊
func (c *Client) JoinGroup(groupID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.groups[groupID] = true
}

// LeaveGroup 离开群聊
func (c *Client) LeaveGroup(groupID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.groups, groupID)
}

// IsInGroup 是否在群聊中
func (c *Client) IsInGroup(groupID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.groups[groupID]
}
```

### 3.3 Hub 连接管理中心

```go
// app/chat/ws/hub/hub.go

package hub

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/zeromicro/go-zero/core/logx"

	"activity-platform/app/chat/ws/internal/types"
	"activity-platform/common/messaging"
)

var (
	ErrSendBufferFull = errors.New("send buffer is full")
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
func NewHub(handler MessageHandler, messagingClient *messaging.Client) *Hub {
	return &Hub{
		clients:         make(map[string]*Client),
		groups:          make(map[string]map[*Client]bool),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		messageHandler:  handler,
		messagingClient: messagingClient,
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
			logx.Info("Hub shutting down")
			return
		}
	}
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
		logx.Infof("User %s connected", client.userID)
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

			logx.Infof("User %s disconnected", client.userID)
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
			Timestamp: msg.Timestamp,
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
		client.SendError(400, "unknown message type")
		return
	}

	if err != nil {
		logx.Errorf("Handle message error: %v", err)
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
		return errors.New("user not connected")
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
			Data:      msg.Payload,
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
			return messaging.NewNonRetryableError(errors.New("missing user_id in metadata"))
		}

		// 发送给指定用户
		wsMsg := &types.WSMessage{
			Type:      types.TypeNotification,
			MessageID: notifData.NotificationID,
			Timestamp: notifData.CreatedAt,
			Data:      msg.Payload,
		}
		h.SendToUser(userID, wsMsg)

		return nil
	})

	// 启动消息订阅
	if err := h.messagingClient.Run(ctx); err != nil {
		logx.Errorf("Messaging client stopped: %v", err)
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
```

### 3.4 消息处理逻辑

```go
// app/chat/ws/internal/logic/message_logic.go

package logic

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"activity-platform/app/chat/ws/hub"
	"activity-platform/app/chat/ws/internal/svc"
	"activity-platform/app/chat/ws/internal/types"
	"activity-platform/app/chat/rpc/chatservice"
	"activity-platform/common/messaging"
)

// MessageLogic 消息处理逻辑
type MessageLogic struct {
	ctx             context.Context
	svcCtx          *svc.ServiceContext
	messagingClient *messaging.Client
}

// NewMessageLogic 创建消息处理逻辑
func NewMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MessageLogic {
	return &MessageLogic{
		ctx:             ctx,
		svcCtx:          svcCtx,
		messagingClient: svcCtx.MessagingClient,
	}
}

// HandleAuth 处理认证
func (l *MessageLogic) HandleAuth(client *hub.Client, msg *types.WSMessage) error {
	var authData types.AuthData
	if err := json.Unmarshal(msg.Data, &authData); err != nil {
		return err
	}

	// 验证 JWT Token
	userID, err := l.svcCtx.JwtAuth.ParseToken(authData.Token)
	if err != nil {
		client.SendMessage(&types.WSMessage{
			Type:      types.TypeAuthFailed,
			Timestamp: time.Now().Unix(),
			Data:      json.RawMessage(`{"message":"invalid token"}`),
		})
		return err
	}

	// 设置客户端用户ID
	client.SetUserID(userID)
	client.SetAuthed(true)

	// 发送认证成功消息
	client.SendMessage(&types.WSMessage{
		Type:      types.TypeAuthSuccess,
		Timestamp: time.Now().Unix(),
		Data:      json.RawMessage(`{"user_id":"` + userID + `"}`),
	})

	// 自动加入用户的所有群聊
	go l.autoJoinUserGroups(client, userID)

	logx.Infof("User %s authenticated", userID)
	return nil
}

// HandleSendMessage 处理发送消息
func (l *MessageLogic) HandleSendMessage(client *hub.Client, msg *types.WSMessage) error {
	var sendData types.SendMessageData
	if err := json.Unmarshal(msg.Data, &sendData); err != nil {
		return err
	}

	// 生成消息ID
	messageID := uuid.New().String()

	// 调用 RPC 保存消息
	// 注意：这里简化处理，实际应该通过 RPC 调用 chat 服务
	// 保存成功后，chat 服务会发布消息到消息中间件
	// 消息中间件会通知所有订阅的 WebSocket 服务
	// WebSocket 服务再推送给客户端

	// 构造新消息数据
	newMsgData := types.NewMessageData{
		MessageID:  messageID,
		GroupID:    sendData.GroupID,
		SenderID:   client.GetUserID(),
		SenderName: "User", // 实际应该从用户服务获取
		MsgType:    sendData.MsgType,
		Content:    sendData.Content,
		ImageURL:   sendData.ImageURL,
		CreatedAt:  time.Now().Unix(),
	}

	// 发布到消息中间件
	payload, _ := json.Marshal(newMsgData)
	if err := l.messagingClient.Publish(l.ctx, "chat.message.new", payload); err != nil {
		return err
	}

	// 发送 ACK
	ackData := types.AckData{
		MessageID: messageID,
		Success:   true,
	}
	ackPayload, _ := json.Marshal(ackData)

	client.SendMessage(&types.WSMessage{
		Type:      types.TypeAck,
		MessageID: msg.MessageID,
		Timestamp: time.Now().Unix(),
		Data:      ackPayload,
	})

	return nil
}

// HandleJoinGroup 处理加入群聊
func (l *MessageLogic) HandleJoinGroup(client *hub.Client, msg *types.WSMessage) error {
	var data struct {
		GroupID string `json:"group_id"`
	}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return err
	}

	// 验证用户是否是群成员（调用 RPC）
	// 这里简化处理，实际应该调用 chat RPC 验证

	// 将客户端添加到群聊
	client.GetHub().AddClientToGroup(client, data.GroupID)

	logx.Infof("User %s joined group %s", client.GetUserID(), data.GroupID)
	return nil
}

// HandleLeaveGroup 处理离开群聊
func (l *MessageLogic) HandleLeaveGroup(client *hub.Client, msg *types.WSMessage) error {
	var data struct {
		GroupID string `json:"group_id"`
	}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return err
	}

	// 将客户端从群聊移除
	client.GetHub().RemoveClientFromGroup(client, data.GroupID)

	logx.Infof("User %s left group %s", client.GetUserID(), data.GroupID)
	return nil
}

// HandleMarkRead 处理标记已读
func (l *MessageLogic) HandleMarkRead(client *hub.Client, msg *types.WSMessage) error {
	var data struct {
		GroupID   string `json:"group_id"`
		MessageID string `json:"message_id"`
	}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return err
	}

	// 调用 RPC 标记已读
	// 这里简化处理，实际应该调用 chat RPC

	logx.Infof("User %s marked message %s as read in group %s",
		client.GetUserID(), data.MessageID, data.GroupID)
	return nil
}

// autoJoinUserGroups 自动加入用户的所有群聊
func (l *MessageLogic) autoJoinUserGroups(client *hub.Client, userID string) {
	// 调用 RPC 获取用户的所有群聊
	resp, err := l.svcCtx.ChatRpc.GetUserGroups(l.ctx, &chatservice.GetUserGroupsReq{
		UserId:   userID,
		Page:     1,
		PageSize: 100,
	})
	if err != nil {
		logx.Errorf("Failed to get user groups: %v", err)
		return
	}

	// 将用户添加到所有群聊
	for _, group := range resp.Groups {
		client.GetHub().AddClientToGroup(client, group.GroupId)
	}

	logx.Infof("User %s auto-joined %d groups", userID, len(resp.Groups))
}
```

### 3.5 WebSocket 处理器

```go
// app/chat/ws/internal/handler/ws_handler.go

package handler

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"

	"activity-platform/app/chat/ws/hub"
	"activity-platform/app/chat/ws/internal/svc"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许跨域
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocketHandler WebSocket 连接处理器
func WebSocketHandler(svcCtx *svc.ServiceContext, h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 升级 HTTP 连接为 WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logx.Errorf("Failed to upgrade connection: %v", err)
			return
		}

		// 创建客户端
		client := hub.NewClient(h, conn)

		// 注册客户端
		h.Register() <- client

		// 启动读写协程
		go client.WritePump()
		go client.ReadPump()

		logx.Info("New WebSocket connection established")
	}
}
```

### 3.6 服务上下文

```go
// app/chat/ws/internal/svc/service_context.go

package svc

import (
	"activity-platform/app/chat/ws/internal/config"
	"activity-platform/app/chat/rpc/chatservice"
	"activity-platform/common/messaging"

	"github.com/zeromicro/go-zero/zrpc"
)

// ServiceContext 服务上下文
type ServiceContext struct {
	Config          config.Config
	ChatRpc         chatservice.ChatService
	MessagingClient *messaging.Client
	JwtAuth         *JwtAuth
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	// 创建 RPC 客户端
	chatRpc := chatservice.NewChatService(zrpc.MustNewClient(c.ChatRpc))

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

	return &ServiceContext{
		Config:          c,
		ChatRpc:         chatRpc,
		MessagingClient: messagingClient,
		JwtAuth:         NewJwtAuth(c.Auth.AccessSecret),
	}
}
```

### 3.7 配置文件

```go
// app/chat/ws/internal/config/config.go

package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

// Config WebSocket 服务配置
type Config struct {
	rest.RestConf

	// Chat RPC 配置
	ChatRpc zrpc.RpcClientConf

	// Redis 配置
	Redis RedisConf

	// JWT 认证配置
	Auth AuthConf

	// WebSocket 配置
	WebSocket WebSocketConf
}

// RedisConf Redis 配置
type RedisConf struct {
	Host string
	Pass string
	DB   int
}

// AuthConf 认证配置
type AuthConf struct {
	AccessSecret string
	AccessExpire int64
}

// WebSocketConf WebSocket 配置
type WebSocketConf struct {
	// 最大连接数
	MaxConnections int `json:",default=10000"`
	// 读取超时（秒）
	ReadTimeout int `json:",default=60"`
	// 写入超时（秒）
	WriteTimeout int `json:",default=10"`
	// 心跳间隔（秒）
	HeartbeatInterval int `json:",default=30"`
}
```

```yaml
# app/chat/ws/etc/websocket.yaml

Name: websocket-service
Host: 0.0.0.0
Port: 8889

# Chat RPC 配置
ChatRpc:
  Etcd:
    Hosts:
      - 127.0.0.1:2379
    Key: chat.rpc

# Redis 配置
Redis:
  Host: 127.0.0.1:6379
  Pass: ""
  DB: 0

# JWT 认证配置
Auth:
  AccessSecret: "your-secret-key-here"
  AccessExpire: 86400

# WebSocket 配置
WebSocket:
  MaxConnections: 10000
  ReadTimeout: 60
  WriteTimeout: 10
  HeartbeatInterval: 30
```

### 3.8 主入口

```go
// app/chat/ws/websocket.go

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"

	"activity-platform/app/chat/ws/hub"
	"activity-platform/app/chat/ws/internal/config"
	"activity-platform/app/chat/ws/internal/handler"
	"activity-platform/app/chat/ws/internal/logic"
	"activity-platform/app/chat/ws/internal/svc"
)

var configFile = flag.String("f", "etc/websocket.yaml", "the config file")

func main() {
	flag.Parse()

	// 加载配置
	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 创建服务上下文
	svcCtx := svc.NewServiceContext(c)

	// 创建消息处理器
	messageHandler := logic.NewMessageLogic(context.Background(), svcCtx)

	// 创建 Hub
	h := hub.NewHub(messageHandler, svcCtx.MessagingClient)

	// 启动 Hub
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go h.Run(ctx)

	// 创建 HTTP 服务器
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.WebSocketHandler(svcCtx, h))

	// 健康检查
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", c.Host, c.Port),
		Handler: mux,
	}

	// 启动服务器
	go func() {
		logx.Infof("Starting WebSocket server at %s:%d", c.Host, c.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logx.Errorf("Server error: %v", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logx.Info("Shutting down server...")
	cancel()

	if err := server.Shutdown(context.Background()); err != nil {
		logx.Errorf("Server shutdown error: %v", err)
	}

	logx.Info("Server stopped")
}
```

---

## 4. 数据库设计

### 4.1 在线状态表（可选，使用 Redis 代替）

```sql
-- 使用 Redis 存储在线状态，无需数据库表
-- Key: online:user:{user_id}
-- Value: {
--   "user_id": "xxx",
--   "connected_at": 1234567890,
--   "last_heartbeat": 1234567890
-- }
-- TTL: 120 秒（心跳间隔的 2 倍）
```

### 4.2 Redis 数据结构

```
# 在线用户集合
Key: online:users
Type: Set
Value: user_id 列表

# 用户在线状态
Key: online:user:{user_id}
Type: Hash
Fields:
  - connected_at: 连接时间戳
  - last_heartbeat: 最后心跳时间戳
TTL: 120 秒

# 群聊在线用户
Key: online:group:{group_id}
Type: Set
Value: user_id 列表
TTL: 无（根据用户在线状态动态维护）
```

---

## 5. 消息流程

### 5.1 用户发送消息流程

```
1. 客户端通过 WebSocket 发送消息
   ↓
2. WebSocket 服务接收消息
   ↓
3. 验证用户权限（是否是群成员）
   ↓
4. 调用 Chat RPC 保存消息到数据库
   ↓
5. Chat RPC 发布消息到消息中间件 (chat.message.new)
   ↓
6. 所有 WebSocket 服务订阅到消息
   ↓
7. WebSocket 服务推送消息给群内在线用户
   ↓
8. 客户端接收到新消息
```

### 5.2 系统通知流程

```
1. 业务服务（如 Activity 服务）调用 Chat RPC 创建通知
   ↓
2. Chat RPC 保存通知到数据库
   ↓
3. Chat RPC 发布通知到消息中间件 (chat.notification.new)
   ↓
4. WebSocket 服务订阅到通知
   ↓
5. WebSocket 服务推送通知给目标用户
   ↓
6. 客户端接收到通知
```

### 5.3 离线消息处理

```
1. 用户上线后，WebSocket 连接建立
   ↓
2. 客户端发送认证消息
   ↓
3. WebSocket 服务验证 Token
   ↓
4. 自动加入用户的所有群聊
   ↓
5. 客户端调用 HTTP API 获取离线消息
   ↓
6. 客户端调用 HTTP API 获取未读通知
   ↓
7. 客户端展示离线消息和通知
```

---

## 6. 客户端接入示例

### 6.1 JavaScript 客户端

```javascript
class WebSocketClient {
  constructor(url, token) {
    this.url = url;
    this.token = token;
    this.ws = null;
    this.messageHandlers = new Map();
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
  }

  connect() {
    this.ws = new WebSocket(this.url);

    this.ws.onopen = () => {
      console.log('WebSocket connected');
      this.reconnectAttempts = 0;

      // 发送认证消息
      this.send({
        type: 'auth',
        message_id: this.generateMessageId(),
        timestamp: Date.now(),
        data: {
          token: this.token
        }
      });

      // 启动心跳
      this.startHeartbeat();
    };

    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      this.handleMessage(message);
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    this.ws.onclose = () => {
      console.log('WebSocket closed');
      this.stopHeartbeat();
      this.reconnect();
    };
  }

  send(message) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  sendMessage(groupId, msgType, content, imageUrl = '') {
    this.send({
      type: 'send_message',
      message_id: this.generateMessageId(),
      timestamp: Date.now(),
      data: {
        group_id: groupId,
        msg_type: msgType,
        content: content,
        image_url: imageUrl
      }
    });
  }

  joinGroup(groupId) {
    this.send({
      type: 'join_group',
      message_id: this.generateMessageId(),
      timestamp: Date.now(),
      data: {
        group_id: groupId
      }
    });
  }

  leaveGroup(groupId) {
    this.send({
      type: 'leave_group',
      message_id: this.generateMessageId(),
      timestamp: Date.now(),
      data: {
        group_id: groupId
      }
    });
  }

  on(type, handler) {
    this.messageHandlers.set(type, handler);
  }

  handleMessage(message) {
    const handler = this.messageHandlers.get(message.type);
    if (handler) {
      handler(message);
    }
  }

  startHeartbeat() {
    this.heartbeatTimer = setInterval(() => {
      this.send({
        type: 'ping',
        timestamp: Date.now()
      });
    }, 30000); // 30 秒
  }

  stopHeartbeat() {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
    }
  }

  reconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
      console.log(`Reconnecting in ${delay}ms...`);
      setTimeout(() => this.connect(), delay);
    }
  }

  generateMessageId() {
    return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  close() {
    this.stopHeartbeat();
    if (this.ws) {
      this.ws.close();
    }
  }
}

// 使用示例
const client = new WebSocketClient('ws://localhost:8889/ws', 'your-jwt-token');

// 监听认证成功
client.on('auth_success', (message) => {
  console.log('Authenticated:', message.data);
});

// 监听新消息
client.on('new_message', (message) => {
  const data = JSON.parse(message.data);
  console.log('New message:', data);
  // 更新 UI
});

// 监听通知
client.on('notification', (message) => {
  const data = JSON.parse(message.data);
  console.log('Notification:', data);
  // 显示通知
});

// 监听错误
client.on('error', (message) => {
  const data = JSON.parse(message.data);
  console.error('Error:', data);
});

// 连接
client.connect();

// 发送消息
client.sendMessage('group-123', 1, 'Hello, World!');

// 加入群聊
client.joinGroup('group-456');
```

---

## 7. 部署方案

### 7.1 单机部署

```yaml
# docker-compose.yml

version: '3.8'

services:
  websocket:
    build: ./app/chat/ws
    ports:
      - "8889:8889"
    environment:
      - REDIS_HOST=redis:6379
      - CHAT_RPC_HOST=chat-rpc:8080
    depends_on:
      - redis
      - chat-rpc
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    restart: unless-stopped

volumes:
  redis-data:
```

### 7.2 集群部署

```
                    ┌─────────────┐
                    │   Nginx     │
                    │  (负载均衡)  │
                    └─────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
   ┌────▼────┐       ┌────▼────┐       ┌────▼────┐
   │  WS-1   │       │  WS-2   │       │  WS-3   │
   │ :8889   │       │ :8889   │       │ :8889   │
   └────┬────┘       └────┬────┘       └────┬────┘
        │                  │                  │
        └──────────────────┼──────────────────┘
                           │
                    ┌─────▼─────┐
                    │   Redis   │
                    │  Stream   │
                    └───────────┘
```

**Nginx 配置**：

```nginx
upstream websocket_backend {
    # IP Hash 确保同一用户连接到同一服务器
    ip_hash;

    server ws-server-1:8889;
    server ws-server-2:8889;
    server ws-server-3:8889;
}

server {
    listen 80;
    server_name ws.example.com;

    location /ws {
        proxy_pass http://websocket_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

        # WebSocket 超时设置
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
}
```

### 7.3 Kubernetes 部署

```yaml
# websocket-deployment.yaml

apiVersion: apps/v1
kind: Deployment
metadata:
  name: websocket-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: websocket-service
  template:
    metadata:
      labels:
        app: websocket-service
    spec:
      containers:
      - name: websocket
        image: campushub/websocket:latest
        ports:
        - containerPort: 8889
        env:
        - name: REDIS_HOST
          value: "redis-service:6379"
        - name: CHAT_RPC_HOST
          value: "chat-rpc-service:8080"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8889
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8889
          initialDelaySeconds: 5
          periodSeconds: 5

---
apiVersion: v1
kind: Service
metadata:
  name: websocket-service
spec:
  type: LoadBalancer
  selector:
    app: websocket-service
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8889
  sessionAffinity: ClientIP
```

---

## 8. 监控与运维

### 8.1 监控指标

**Prometheus 指标**：

```go
// 连接数指标
websocket_connections_total{service="websocket"}

// 消息处理指标
websocket_messages_total{type="send|receive",service="websocket"}
websocket_message_duration_seconds{type="send|receive",service="websocket"}

// 错误指标
websocket_errors_total{type="auth|send|receive",service="websocket"}

// 在线用户数
websocket_online_users{service="websocket"}
```

### 8.2 日志规范

```go
// 连接日志
logx.Infof("[WS] User %s connected from %s", userID, remoteAddr)

// 消息日志
logx.Infof("[WS] User %s sent message to group %s", userID, groupID)

// 错误日志
logx.Errorf("[WS] Failed to send message: %v", err)

// 性能日志
logx.Slowf("[WS] Message processing took %dms", duration)
```

### 8.3 告警规则

```yaml
# Prometheus 告警规则

groups:
- name: websocket
  rules:
  # 连接数过高
  - alert: WebSocketHighConnections
    expr: websocket_connections_total > 8000
    for: 5m
    annotations:
      summary: "WebSocket 连接数过高"

  # 错误率过高
  - alert: WebSocketHighErrorRate
    expr: rate(websocket_errors_total[5m]) > 10
    for: 5m
    annotations:
      summary: "WebSocket 错误率过高"

  # 消息延迟过高
  - alert: WebSocketHighLatency
    expr: histogram_quantile(0.95, websocket_message_duration_seconds) > 1
    for: 5m
    annotations:
      summary: "WebSocket 消息延迟过高"
```

---

## 9. 性能优化

### 9.1 连接池优化

- 使用 `sync.Pool` 复用 buffer
- 限制最大连接数
- 实现连接限流

### 9.2 消息批处理

- 批量发送消息减少系统调用
- 使用 buffer 缓存消息

### 9.3 内存优化

- 及时释放断开连接的资源
- 使用对象池减少 GC 压力
- 限制消息大小

### 9.4 网络优化

- 启用 TCP_NODELAY
- 调整 buffer 大小
- 使用消息压缩（可选）

---

## 10. 安全考虑

### 10.1 认证与授权

- JWT Token 认证
- Token 过期自动断开连接
- 验证用户群聊权限

### 10.2 防护措施

- 限制消息大小（512KB）
- 限制消息频率（防止刷屏）
- IP 限流
- 连接数限制

### 10.3 数据安全

- 使用 WSS (WebSocket over TLS)
- 敏感数据加密
- 防止 XSS 攻击

---

## 11. 测试方案

### 11.1 单元测试

```go
// hub_test.go
func TestHub_BroadcastToGroup(t *testing.T) {
    // 测试群聊广播
}

// client_test.go
func TestClient_SendMessage(t *testing.T) {
    // 测试客户端发送消息
}
```

### 11.2 集成测试

```go
// integration_test.go
func TestWebSocket_EndToEnd(t *testing.T) {
    // 测试完整的消息流程
}
```

### 11.3 压力测试

```bash
# 使用 websocket-bench 进行压力测试
websocket-bench -c 1000 -s 10 ws://localhost:8889/ws
```

---

## 12. 实施计划

### 12.1 第一阶段：基础功能（1-2 周）

- [ ] 实现 WebSocket 连接管理
- [ ] 实现消息协议
- [ ] 实现认证机制
- [ ] 实现群聊消息推送

### 12.2 第二阶段：完善功能（1 周）

- [ ] 实现系统通知推送
- [ ] 实现在线状态管理
- [ ] 实现离线消息处理
- [ ] 集成消息中间件

### 12.3 第三阶段：优化与测试（1 周）

- [ ] 性能优化
- [ ] 单元测试
- [ ] 集成测试
- [ ] 压力测试

### 12.4 第四阶段：部署上线（3-5 天）

- [ ] 编写部署文档
- [ ] 配置监控告警
- [ ] 灰度发布
- [ ] 全量上线

---

## 13. 总结

本方案基于 CampusHub 现有架构，设计了一套完整的 WebSocket 实时消息推送服务。主要特点：

1. **架构清晰**：分层设计，职责明确
2. **可扩展**：支持水平扩展，可部署多实例
3. **高性能**：使用 goroutine 并发处理，支持万级并发连接
4. **可靠性**：集成消息中间件，保证消息不丢失
5. **易维护**：完善的监控、日志和告警机制
6. **安全性**：JWT 认证、权限控制、防护措施

该方案遵循 Go-Zero 框架规范，与现有代码风格保持一致，可以无缝集成到 CampusHub 项目中。

