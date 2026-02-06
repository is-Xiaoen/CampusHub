# WebSocket 实时消息推送服务

## 概述

基于 Go-Zero 和 Gorilla WebSocket 实现的实时消息推送服务，支持群聊消息和系统通知的实时推送。

## 功能特性

- ✅ WebSocket 长连接管理
- ✅ JWT Token 认证
- ✅ 群聊消息实时推送
- ✅ 系统通知实时推送
- ✅ 心跳保活机制
- ✅ 自动重连支持
- ✅ 消息确认机制
- ✅ 集成消息中间件（Redis Stream）

## 目录结构

```
app/chat/ws/
├── websocket.go              # 主入口
├── websocket                 # 编译后的可执行文件
├── etc/
│   └── websocket.yaml        # 配置文件
├── hub/
│   ├── hub.go               # Hub 管理中心
│   └── client.go            # 客户端连接
└── internal/
    ├── config/              # 配置
    ├── handler/             # HTTP 处理器
    ├── logic/               # 业务逻辑
    ├── svc/                 # 服务上下文
    └── types/               # 类型定义
```

## 快速开始

### 1. 配置文件

编辑 `etc/websocket.yaml`：

```yaml
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
  AccessSecret: "k9#8G7&6F5%4D3$2S1@0P9*8O7!6N5^4M3+2L1=0"
  AccessExpire: 86400
```

### 2. 启动服务

```bash
# 开发环境
cd app/chat/ws
go run websocket.go -f etc/websocket.yaml

# 生产环境
./websocket -f etc/websocket.yaml
```

### 3. 健康检查

```bash
curl http://localhost:8889/health
# 返回: OK

curl http://localhost:8889/stats
# 返回: {"online_users":0}
```

## WebSocket 协议

### 连接地址

```
ws://localhost:8889/ws
```

### 消息格式

所有消息使用 JSON 格式：

```json
{
  "type": "消息类型",
  "message_id": "消息ID",
  "timestamp": 1234567890,
  "data": {}
}
```

### 消息类型

#### 客户端 -> 服务端

| 类型 | 说明 | Data 结构 |
|------|------|-----------|
| `ping` | 心跳 | 无 |
| `auth` | 认证 | `{"token": "jwt_token"}` |
| `send_message` | 发送消息 | `{"group_id": "xxx", "msg_type": 1, "content": "hello"}` |
| `join_group` | 加入群聊 | `{"group_id": "xxx"}` |
| `leave_group` | 离开群聊 | `{"group_id": "xxx"}` |
| `mark_read` | 标记已读 | `{"group_id": "xxx", "message_id": "xxx"}` |

#### 服务端 -> 客户端

| 类型 | 说明 | Data 结构 |
|------|------|-----------|
| `pong` | 心跳响应 | 无 |
| `auth_success` | 认证成功 | `{"user_id": "123"}` |
| `auth_failed` | 认证失败 | `{"message": "invalid token"}` |
| `new_message` | 新消息 | 消息详情 |
| `notification` | 系统通知 | 通知详情 |
| `error` | 错误 | `{"code": 400, "message": "错误信息"}` |
| `ack` | 消息确认 | `{"message_id": "xxx", "success": true}` |

## 使用示例

### JavaScript 客户端

```javascript
// 创建 WebSocket 连接
const ws = new WebSocket('ws://localhost:8889/ws');

ws.onopen = () => {
  console.log('Connected');

  // 发送认证消息
  ws.send(JSON.stringify({
    type: 'auth',
    message_id: Date.now().toString(),
    timestamp: Date.now(),
    data: {
      token: 'your_jwt_token_here'
    }
  }));
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message);

  switch (message.type) {
    case 'auth_success':
      console.log('Authenticated as user:', message.data.user_id);
      // 认证成功后，可以发送消息
      break;

    case 'new_message':
      console.log('New message:', message.data);
      // 处理新消息
      break;

    case 'notification':
      console.log('Notification:', message.data);
      // 处理通知
      break;
  }
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('Disconnected');
};

// 发送消息
function sendMessage(groupId, content) {
  ws.send(JSON.stringify({
    type: 'send_message',
    message_id: Date.now().toString(),
    timestamp: Date.now(),
    data: {
      group_id: groupId,
      msg_type: 1,
      content: content
    }
  }));
}

// 心跳
setInterval(() => {
  ws.send(JSON.stringify({
    type: 'ping',
    timestamp: Date.now()
  }));
}, 30000);
```

### Go 客户端

```go
package main

import (
    "encoding/json"
    "log"
    "time"

    "github.com/gorilla/websocket"
)

func main() {
    // 连接 WebSocket
    conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8889/ws", nil)
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    // 发送认证消息
    authMsg := map[string]interface{}{
        "type": "auth",
        "message_id": time.Now().Unix(),
        "timestamp": time.Now().Unix(),
        "data": map[string]string{
            "token": "your_jwt_token_here",
        },
    }
    conn.WriteJSON(authMsg)

    // 读取消息
    for {
        var msg map[string]interface{}
        err := conn.ReadJSON(&msg)
        if err != nil {
            log.Println("read error:", err)
            break
        }
        log.Printf("Received: %+v\n", msg)
    }
}
```

## 架构说明

### 消息流程

1. **用户发送消息**
   - 客户端通过 WebSocket 发送消息
   - WebSocket 服务验证权限
   - 发布消息到 Redis Stream
   - 所有 WebSocket 实例订阅到消息
   - 推送给群内在线用户

2. **系统通知**
   - 业务服务创建通知
   - 发布通知到 Redis Stream
   - WebSocket 服务订阅到通知
   - 推送给目标用户

### 扩展性

- 支持水平扩展，可部署多个 WebSocket 实例
- 通过 Redis Stream 实现消息广播
- 使用 Nginx 进行负载均衡（IP Hash）

## 监控

### 健康检查

```bash
GET /health
```

### 在线用户统计

```bash
GET /stats
```

返回：
```json
{
  "online_users": 100
}
```

## 注意事项

1. **JWT Token**: 必须与其他 API 服务使用相同的 AccessSecret
2. **跨域**: 当前配置允许所有来源，生产环境需要限制
3. **心跳**: 客户端需要定期发送 ping 消息保持连接
4. **重连**: 客户端需要实现断线重连机制
5. **消息去重**: 使用 message_id 进行消息去重

## 依赖服务

- **Chat RPC**: 群聊和消息管理
- **Redis**: 消息中间件（Redis Stream）
- **Etcd**: 服务发现

## 开发计划

- [ ] 添加消息限流
- [ ] 添加 IP 黑名单
- [ ] 添加 Prometheus 监控指标
- [ ] 添加消息压缩
- [ ] 支持 WSS (WebSocket over TLS)
- [ ] 添加更多的单元测试

## 故障排查

### 连接失败

1. 检查服务是否启动：`curl http://localhost:8889/health`
2. 检查防火墙设置
3. 检查配置文件中的端口号

### 认证失败

1. 检查 JWT Token 是否有效
2. 检查 AccessSecret 是否与其他服务一致
3. 检查 Token 是否过期

### 消息未推送

1. 检查 Redis 连接
2. 检查消息中间件是否正常运行
3. 检查用户是否已加入群聊

## 许可证

MIT License
