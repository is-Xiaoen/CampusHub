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
	// 心跳超时时间
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
				logx.Errorf("WebSocket 错误: %v", err)
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
		logx.Errorf("用户 %s 的发送缓冲区已满", c.userID)
		return ErrSendBufferFull
	}
}

// handleMessage 处理接收到的消息
func (c *Client) handleMessage(message []byte) {
	var msg types.WSMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		c.SendError(400, "消息格式错误")
		return
	}

	// 未认证只能发送认证消息和心跳
	if !c.isAuthed && msg.Type != types.TypeAuth && msg.Type != types.TypePing {
		c.SendError(401, "未授权，请先认证")
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

// SetUserID 设置用户ID
func (c *Client) SetUserID(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.userID = userID
}

// GetUserID 获取用户ID
func (c *Client) GetUserID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.userID
}

// SetAuthed 设置认证状态
func (c *Client) SetAuthed(authed bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isAuthed = authed
}

// IsAuthed 是否已认证
func (c *Client) IsAuthed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isAuthed
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

// GetHub 获取 Hub
func (c *Client) GetHub() *Hub {
	return c.hub
}
