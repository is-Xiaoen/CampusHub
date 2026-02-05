# CampusHub WebSocket 连接与消息转发完整流程讲解

## 目录
1. [架构概览](#架构概览)
2. [WebSocket 连接建立全过程](#websocket-连接建立全过程)
3. [消息转发全过程](#消息转发全过程)
4. [关键组件详解](#关键组件详解)
5. [消息类型说明](#消息类型说明)

---

## 架构概览

### 核心组件关系图

```
客户端 (Browser/App)
    ↓ HTTP Upgrade
WebSocket Handler (ws_handler.go)
    ↓ 创建 Client
Hub (hub.go) ← 管理所有连接
    ↓ 消息处理
MessageLogic (message_logic.go)
    ↓ 持久化
RPC Service + 消息队列
    ↓ 发布
消息中间件 (Watermill)
    ↓ 订阅
Hub → 广播到群聊成员
    ↓
Client → WritePump → 客户端
```

### 技术栈
- **WebSocket 库**: gorilla/websocket
- **消息中间件**: Watermill (支持 Redis Streams)
- **RPC 框架**: gRPC (go-zero)
- **状态存储**: Redis
- **并发模型**: Goroutine + Channel

---

## WebSocket 连接建立全过程

### 第 1 步：服务启动 - 初始化核心组件

**文件：** `app/chat/ws/websocket.go:24-48`

```go
func main() {
    flag.Parse()

    // 1. 加载配置文件
    var c config.Config
    conf.MustLoad(*configFile, &c)

    // 2. 创建服务上下文（包含 RPC 客户端、Redis、消息队列等）
    svcCtx := svc.NewServiceContext(c)

    // 3. 创建消息处理器（负责业务逻辑）
    messageHandler := logic.NewMessageLogic(context.Background(), svcCtx)

    // 4. 创建 Hub（连接管理中心）
    h := hub.NewHub(messageHandler, svcCtx.MessagingClient, svcCtx.RedisClient)

    // 5. 启动 Hub（在后台运行，监听注册/注销事件）
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    go h.Run(ctx)

    // 6. 创建 HTTP 服务器，注册 WebSocket 路由
    mux := http.NewServeMux()
    mux.HandleFunc("/ws", handler.WebSocketHandler(svcCtx, h))

    // 健康检查端点
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })

    // 在线用户数查询
    mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        fmt.Fprintf(w, `{"online_users":%d}`, h.GetOnlineUserCount())
    })

    // 7. 启动服务器
    server := &http.Server{
        Addr:    fmt.Sprintf("%s:%d", c.Host, c.Port),
        Handler: mux,
    }
    go server.ListenAndServe()
}
```

**Hub 的核心数据结构：** `app/chat/ws/hub/hub.go:24-48`

```go
type Hub struct {
    // 已注册的客户端映射（userID -> Client）
    clients map[string]*Client

    // 群聊订阅映射（groupID -> 该群的所有在线客户端）
    groups map[string]map[*Client]bool

    // 注册通道（新连接通过此通道注册）
    register chan *Client

    // 注销通道（断开连接通过此通道注销）
    unregister chan *Client

    // 消息处理器（处理各种消息类型）
    messageHandler MessageHandler

    // 消息中间件客户端（用于跨服务通信）
    messagingClient *messaging.Client

    // Redis 客户端（存储用户在线状态）
    redisClient *redis.Client

    mu sync.RWMutex
}
```

---

### 第 2 步：HTTP 升级为 WebSocket 连接

当客户端发起 WebSocket 连接请求时（例如：`ws://localhost:8080/ws`），服务器会将 HTTP 连接升级为 WebSocket 连接。

**文件：** `app/chat/ws/internal/handler/ws_handler.go:22-44`

```go
func WebSocketHandler(svcCtx *svc.ServiceContext, h *hub.Hub) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. 使用 gorilla/websocket 升级 HTTP 连接为 WebSocket
        conn, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
            logx.Errorf("升级连接失败: %v", err)
            return
        }

        // 2. 创建客户端对象（封装 WebSocket 连接）
        client := hub.NewClient(h, conn)

        // 3. 注册客户端到 Hub（通过 channel 异步通信）
        h.Register() <- client

        // 4. 启动读写协程（并发处理收发消息）
        go client.WritePump()  // 负责发送消息给客户端
        go client.ReadPump()   // 负责接收客户端消息

        logx.Info("新的 WebSocket 连接已建立")
    }
}
```

**Upgrader 配置：** `app/chat/ws/internal/handler/ws_handler.go:13-20`

```go
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    // 允许跨域（生产环境应该配置具体的域名）
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}
```

---

### 第 3 步：Hub 注册客户端

Hub 在后台运行一个事件循环，监听注册和注销事件。

**文件：** `app/chat/ws/hub/hub.go:72-90`

```go
// Hub.Run() 在后台运行，监听注册/注销事件
func (h *Hub) Run(ctx context.Context) {
    // 订阅消息中间件的消息（用于接收其他服务发来的消息）
    go h.subscribeMessages(ctx)

    for {
        select {
        case client := <-h.register:
            h.registerClient(client)  // 处理注册

        case client := <-h.unregister:
            h.unregisterClient(client)  // 处理注销

        case <-ctx.Done():
            logx.Info("Hub 正在关闭")
            return
        }
    }
}
```

**注册逻辑：** `app/chat/ws/hub/hub.go:97-114`

```go
func (h *Hub) registerClient(client *Client) {
    h.mu.Lock()
    defer h.mu.Unlock()

    if client.userID != "" {
        // 如果用户已有连接，关闭旧连接（实现单点登录）
        if oldClient, exists := h.clients[client.userID]; exists {
            close(oldClient.send)
        }
        h.clients[client.userID] = client

        // 更新用户在线状态到 Redis
        h.updateUserStatus(client.userID, true)

        logx.Infof("用户 %s 已连接", client.userID)
    }
}
```

**更新用户状态到 Redis：** `app/chat/ws/hub/hub.go:305-328`

```go
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
```

---

### 第 4 步：启动读写协程

每个客户端连接都有两个独立的 Goroutine：
- **ReadPump**: 从 WebSocket 读取消息
- **WritePump**: 向 WebSocket 写入消息

**Client 数据结构：** `app/chat/ws/hub/client.go:25-34`

```go
type Client struct {
    hub      *Hub                // 所属的 Hub
    conn     *websocket.Conn     // WebSocket 连接
    send     chan []byte         // 发送消息的缓冲通道（256 容量）
    userID   string              // 用户ID（认证后设置）
    groups   map[string]bool     // 用户加入的群聊列表
    mu       sync.RWMutex        // 读写锁
    isAuthed bool                // 是否已认证
}
```

**ReadPump（读取协程）：** `app/chat/ws/hub/client.go:46-72`

```go
func (c *Client) ReadPump() {
    defer func() {
        // 连接断开时，注销客户端
        c.hub.unregister <- c
        c.conn.Close()
    }()

    // 设置读取限制和超时
    c.conn.SetReadLimit(maxMessageSize)  // 最大 512KB
    c.conn.SetReadDeadline(time.Now().Add(pongWait))  // 60 秒超时

    // 设置 Pong 处理器（收到 Pong 时重置超时）
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })

    // 循环读取消息
    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                logx.Errorf("WebSocket 错误: %v", err)
            }
            break
        }

        // 处理接收到的消息
        c.handleMessage(message)
    }
}
```

**WritePump（写入协程）：** `app/chat/ws/hub/client.go:74-115`

```go
func (c *Client) WritePump() {
    ticker := time.NewTicker(pingPeriod)  // 54 秒发送一次 Ping
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))  // 10 秒写入超时
            if !ok {
                // 通道关闭，发送关闭消息
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            w, err := c.conn.NextWriter(websocket.TextMessage)
            if err != nil {
                return
            }
            w.Write(message)

            // 批量写入队列中的消息（性能优化）
            n := len(c.send)
            for i := 0; i < n; i++ {
                w.Write([]byte{'\n'})
                w.Write(<-c.send)
            }

            if err := w.Close(); err != nil {
                return
            }

        case <-ticker.C:
            // 定时发送 Ping 保持连接活跃
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}
```

**心跳机制常量：** `app/chat/ws/hub/client.go:14-23`

```go
const (
    // 写入超时时间
    writeWait = 10 * time.Second
    // 心跳超时时间（60 秒没收到 Pong 就断开）
    pongWait = 60 * time.Second
    // Ping 间隔（必须小于 pongWait）
    pingPeriod = (pongWait * 9) / 10  // 54 秒
    // 最大消息大小
    maxMessageSize = 512 * 1024  // 512KB
)
```

---

### 第 5 步：客户端认证

连接建立后，客户端必须先发送认证消息才能进行其他操作。

**客户端发送认证消息：**

```json
{
    "type": "auth",
    "message_id": "uuid-xxx",
    "timestamp": 1234567890,
    "data": {
        "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    }
}
```

**服务端处理认证：** `app/chat/ws/internal/logic/message_logic.go:35-70`

```go
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
            Data:      json.RawMessage(`{"message":"Token 无效或已过期"}`),
        })
        return err
    }

    // 设置客户端用户ID
    client.SetUserID(userID)
    client.SetAuthed(true)

    // 发送认证成功消息
    successData, _ := json.Marshal(map[string]string{"user_id": userID})
    client.SendMessage(&types.WSMessage{
        Type:      types.TypeAuthSuccess,
        Timestamp: time.Now().Unix(),
        Data:      successData,
    })

    // 自动加入用户的所有群聊
    go l.autoJoinUserGroups(client, userID)

    logx.Infof("用户 %s 认证成功", userID)
    return nil
}
```

**自动加入用户群聊：** `app/chat/ws/internal/logic/message_logic.go:249-268`

```go
func (l *MessageLogic) autoJoinUserGroups(client *hub.Client, userID string) {
    // 调用 RPC 获取用户的所有群聊
    resp, err := l.svcCtx.ChatRpc.GetUserGroups(l.ctx, &chat.GetUserGroupsReq{
        UserId:   userID,
        Page:     1,
        PageSize: 100,
    })
    if err != nil {
        logx.Errorf("获取用户群聊列表失败: %v", err)
        return
    }

    // 将用户添加到所有群聊
    for _, group := range resp.Groups {
        client.GetHub().AddClientToGroup(client, group.GroupId)
    }

    logx.Infof("用户 %s 自动加入了 %d 个群聊", userID, len(resp.Groups))
}
```

---

## 消息转发全过程

### 完整流程图

```
客户端 A 发送消息
    ↓
ReadPump 读取消息
    ↓
handleMessage 解析消息
    ↓
Hub.handleClientMessage 路由消息
    ↓
MessageLogic.HandleSendMessage 处理消息
    ↓
├─ 1. 立即发送 ACK（1-2ms）
├─ 2. 发布到消息中间件（实时推送）
└─ 3. 异步保存到数据库（推送到队列）
    ↓
消息中间件 (Watermill)
    ↓
Hub.subscribeMessages 订阅消息
    ↓
Hub.BroadcastToGroup 广播到群聊
    ↓
Client.SendMessage 发送到各客户端
    ↓
WritePump 写入 WebSocket
    ↓
客户端 B、C、D 收到消息
```

---

### 步骤 1：客户端发送消息

**客户端发送消息格式：**

```json
{
    "type": "send_message",
    "message_id": "client-msg-uuid",
    "timestamp": 1234567890,
    "data": {
        "group_id": "group-123",
        "msg_type": 1,
        "content": "Hello, World!"
    }
}
```

---

### 步骤 2：ReadPump 读取并路由消息

**文件：** `app/chat/ws/hub/client.go:133-149`

```go
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
```

**Hub 路由消息：** `app/chat/ws/hub/hub.go:144-181`

```go
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
```

---

### 步骤 3：处理发送消息（异步版本）

这是项目的核心逻辑，采用**异步保存 + 实时推送**的架构，实现低延迟和高吞吐。

**文件：** `app/chat/ws/internal/logic/message_logic.go:72-134`

```go
func (l *MessageLogic) HandleSendMessage(client *hub.Client, msg *types.WSMessage) error {
    var sendData types.SendMessageData
    if err := json.Unmarshal(msg.Data, &sendData); err != nil {
        return err
    }

    // 生成消息ID
    messageID := uuid.New().String()
    now := time.Now().Unix()

    // 构造新消息数据
    newMsgData := types.NewMessageData{
        MessageID:  messageID,
        GroupID:    sendData.GroupID,
        SenderID:   client.GetUserID(),
        SenderName: "User",  // TODO: 从用户服务获取用户名
        MsgType:    sendData.MsgType,
        Content:    sendData.Content,
        ImageURL:   sendData.ImageURL,
        CreatedAt:  now,
    }

    // 1. 立即发送 ACK（快速响应，延迟 1-2ms）✅
    ackData := types.AckData{
        MessageID: messageID,
        Success:   true,
    }
    ackPayload, _ := json.Marshal(ackData)
    client.SendMessage(&types.WSMessage{
        Type:      types.TypeAck,
        MessageID: msg.MessageID,
        Timestamp: now,
        Data:      ackPayload,
    })

    // 2. 发布到消息中间件（实时推送）
    payload, _ := json.Marshal(newMsgData)
    if err := l.messagingClient.Publish(l.ctx, "chat.message.new", payload); err != nil {
        logx.Errorf("发布消息到中间件失败: %v", err)
        // 不影响后续流程，消息已保存到数据库
    }

    // 3. 异步保存到数据库（推送到队列）✅
    task := &queue.SaveMessageTask{
        MessageID: messageID,
        GroupID:   sendData.GroupID,
        SenderID:  client.GetUserID(),
        MsgType:   sendData.MsgType,
        Content:   sendData.Content,
        ImageURL:  sendData.ImageURL,
        Retry:     0,
    }

    if err := l.svcCtx.SaveQueue.Push(task); err != nil {
        logx.Errorf("推送消息到保存队列失败: %v", err)
        // 队列满了，记录告警
        // TODO: 发送告警通知
    }

    logx.Infof("消息处理完成（异步）: message_id=%s, group_id=%s", messageID, sendData.GroupID)
    return nil
}
```

**关键设计点：**
1. **立即 ACK**：客户端 1-2ms 就能收到确认，用户体验好
2. **实时推送**：通过消息中间件立即推送给在线用户
3. **异步持久化**：数据库写入不阻塞消息发送，提高吞吐量

---

### 步骤 4：消息中间件订阅与广播

Hub 启动时会订阅消息中间件的消息。

**文件：** `app/chat/ws/hub/hub.go:213-232`

```go
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

    // 启动消息订阅
    if err := h.messagingClient.Run(ctx); err != nil {
        logx.Errorf("消息中间件客户端停止: %v", err)
    }
}
```

**广播到群聊：** `app/chat/ws/hub/hub.go:183-198`

```go
func (h *Hub) BroadcastToGroup(groupID string, msg *types.WSMessage) {
    h.mu.RLock()
    clients, ok := h.groups[groupID]
    h.mu.RUnlock()

    if !ok {
        return
    }

    // 遍历群聊中的所有在线客户端
    for client := range clients {
        if client.IsInGroup(groupID) {
            client.SendMessage(msg)
        }
    }
}
```

---

### 步骤 5：发送消息到客户端

**文件：** `app/chat/ws/hub/client.go:117-131`

```go
func (c *Client) SendMessage(msg *types.WSMessage) error {
    data, err := json.Marshal(msg)
    if err != nil {
        return err
    }

    select {
    case c.send <- data:
        return nil
    default:
        // 发送缓冲区已满（256 容量）
        logx.Errorf("用户 %s 的发送缓冲区已满", c.userID)
        return ErrSendBufferFull
    }
}
```

消息被推送到 `c.send` 通道后，`WritePump` 协程会自动从通道读取并发送给客户端。

---

## 关键组件详解

### 1. Hub - 连接管理中心

**职责：**
- 管理所有在线客户端
- 管理群聊订阅关系
- 路由消息到对应的处理器
- 订阅消息中间件并广播消息
- 维护用户在线状态（Redis）

**核心方法：**
- `Run()`: 事件循环，处理注册/注销
- `registerClient()`: 注册新客户端
- `unregisterClient()`: 注销客户端
- `BroadcastToGroup()`: 广播消息到群聊
- `SendToUser()`: 发送消息给指定用户
- `subscribeMessages()`: 订阅消息中间件

---

### 2. Client - WebSocket 客户端

**职责：**
- 封装 WebSocket 连接
- 管理读写协程
- 维护用户的群聊列表
- 处理心跳机制

**核心方法：**
- `ReadPump()`: 读取客户端消息
- `WritePump()`: 发送消息给客户端
- `SendMessage()`: 推送消息到发送队列
- `handleMessage()`: 解析并路由消息

---

### 3. MessageLogic - 消息处理逻辑

**职责：**
- 处理各种消息类型（认证、发送消息、加入群聊等）
- 调用 RPC 服务进行数据持久化
- 发布消息到消息中间件
- 管理异步保存队列

**核心方法：**
- `HandleAuth()`: 处理认证
- `HandleSendMessage()`: 处理发送消息（异步）
- `HandleSendMessageSync()`: 处理发送消息（同步）
- `HandleJoinGroup()`: 处理加入群聊
- `HandleLeaveGroup()`: 处理离开群聊
- `HandleMarkRead()`: 处理标记已读

---

### 4. 消息中间件 (Watermill)

**作用：**
- 解耦消息发送和接收
- 支持多实例部署（水平扩展）
- 提供消息持久化和重试机制

**主题（Topic）：**
- `chat.message.new`: 新消息
- `chat.notification.new`: 系统通知

---

## 消息类型说明

### 客户端 → 服务端

| 类型 | 说明 | 数据结构 |
|------|------|----------|
| `ping` | 心跳 | 无 |
| `auth` | 认证 | `{"token": "jwt-token"}` |
| `send_message` | 发送消息 | `{"group_id": "xxx", "msg_type": 1, "content": "xxx"}` |
| `join_group` | 加入群聊 | `{"group_id": "xxx"}` |
| `leave_group` | 离开群聊 | `{"group_id": "xxx"}` |
| `mark_read` | 标记已读 | `{"group_id": "xxx", "message_id": "xxx"}` |

### 服务端 → 客户端

| 类型 | 说明 | 数据结构 |
|------|------|----------|
| `pong` | 心跳响应 | 无 |
| `auth_success` | 认证成功 | `{"user_id": "xxx"}` |
| `auth_failed` | 认证失败 | `{"message": "Token 无效"}` |
| `new_message` | 新消息 | `NewMessageData` |
| `notification` | 系统通知 | `NotificationData` |
| `error` | 错误消息 | `{"code": 400, "message": "xxx"}` |
| `ack` | 消息确认 | `{"message_id": "xxx", "success": true}` |

---

## 性能优化点

### 1. 异步保存架构
- 消息先发送 ACK，再异步保存到数据库
- 延迟从 50-100ms 降低到 1-2ms

### 2. 批量写入优化
- WritePump 会批量写入队列中的消息
- 减少系统调用次数

### 3. 消息中间件解耦
- 支持多实例部署
- 消息可靠传递

### 4. Redis 存储用户状态
- 快速查询用户在线状态
- 支持分布式部署

### 5. Channel 通信
- 使用 Go Channel 实现高效的并发通信
- 避免锁竞争

---

## 总结

CampusHub 的 WebSocket 实现采用了以下关键技术：

1. **Hub-Client 架构**：集中管理所有连接
2. **读写分离**：每个连接两个独立协程
3. **异步持久化**：消息先推送再保存
4. **消息中间件**：解耦发送和接收，支持水平扩展
5. **心跳机制**：保持连接活跃，及时检测断线
6. **认证机制**：JWT Token 认证，保证安全性

这种架构实现了**低延迟、高吞吐、高可用**的实时通信系统。
