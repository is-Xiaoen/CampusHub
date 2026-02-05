# WebSocket 服务目录结构

## 完整目录树

```
app/chat/ws/
├── websocket.go                    # 主入口文件
├── etc/
│   └── websocket.yaml              # 配置文件
├── hub/
│   ├── hub.go                      # Hub 连接管理中心
│   ├── client.go                   # 客户端连接管理
│   └── manager.go                  # 连接管理器（可选）
└── internal/
    ├── config/
    │   └── config.go               # 配置结构定义
    ├── handler/
    │   └── ws_handler.go           # WebSocket HTTP 处理器
    ├── logic/
    │   ├── message_logic.go        # 消息处理逻辑
    │   └── connection_logic.go     # 连接管理逻辑（可选）
    ├── svc/
    │   └── service_context.go      # 服务上下文
    ├── types/
    │   └── types.go                # 消息类型定义
    └── middleware/
        └── auth.go                 # JWT 认证中间件（可选）
```

## 文件说明

### 根目录
- **websocket.go**: 服务主入口，负责启动 HTTP 服务器和 Hub

### etc/
- **websocket.yaml**: 服务配置文件（端口、Redis、RPC 等）

### hub/
- **hub.go**: 连接管理中心，负责客户端注册/注销、消息广播
- **client.go**: 单个客户端连接的封装，处理读写操作
- **manager.go**: 连接管理器（可选，用于更复杂的连接管理）

### internal/config/
- **config.go**: 配置结构体定义

### internal/handler/
- **ws_handler.go**: HTTP 处理器，负责升级 HTTP 连接为 WebSocket

### internal/logic/
- **message_logic.go**: 消息处理业务逻辑（认证、发送消息、加入群聊等）
- **connection_logic.go**: 连接管理业务逻辑（可选）

### internal/svc/
- **service_context.go**: 服务上下文，管理 RPC 客户端、消息中间件等依赖

### internal/types/
- **types.go**: WebSocket 消息协议定义

### internal/middleware/
- **auth.go**: JWT 认证中间件（可选）

## 核心文件依赖关系

```
websocket.go
    ├── config.Config
    ├── svc.ServiceContext
    │   ├── config.Config
    │   ├── chatservice.ChatService (RPC)
    │   └── messaging.Client
    ├── hub.Hub
    │   ├── hub.Client
    │   └── logic.MessageLogic
    └── handler.WebSocketHandler
        └── hub.Hub
```

## 导入路径规范

```go
// 内部包导入
import (
    "activity-platform/app/chat/ws/hub"
    "activity-platform/app/chat/ws/internal/config"
    "activity-platform/app/chat/ws/internal/handler"
    "activity-platform/app/chat/ws/internal/logic"
    "activity-platform/app/chat/ws/internal/svc"
    "activity-platform/app/chat/ws/internal/types"
)

// 项目其他包导入
import (
    "activity-platform/app/chat/rpc/chatservice"
    "activity-platform/common/messaging"
)

// 第三方包导入
import (
    "github.com/gorilla/websocket"
    "github.com/zeromicro/go-zero/core/logx"
    "github.com/zeromicro/go-zero/zrpc"
)
```

## 启动命令

```bash
# 开发环境
cd app/chat/ws
go run websocket.go -f etc/websocket.yaml

# 生产环境
cd app/chat/ws
go build -o websocket websocket.go
./websocket -f etc/websocket.yaml
```

## Docker 构建

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .

# 构建 WebSocket 服务
RUN cd app/chat/ws && \
    go build -o websocket websocket.go

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/app/chat/ws/websocket .
COPY --from=builder /app/app/chat/ws/etc ./etc

EXPOSE 8889

CMD ["./websocket", "-f", "etc/websocket.yaml"]
```

## 与现有服务的关系

```
app/chat/
├── api/            # HTTP API 服务（REST 接口）
├── rpc/            # gRPC 服务（业务逻辑）
├── ws/             # WebSocket 服务（实时推送）← 新增
└── model/          # 数据模型（共享）
```

三个服务的职责：
- **api**: 提供 HTTP REST API，处理同步请求
- **rpc**: 提供 gRPC 接口，处理业务逻辑和数据持久化
- **ws**: 提供 WebSocket 连接，处理实时消息推送

## 服务间通信

```
┌─────────┐         HTTP          ┌─────────┐
│ 前端     │ ◄──────────────────► │ API     │
│         │                       │ Service │
└─────────┘                       └─────────┘
     │                                  │
     │ WebSocket                        │ gRPC
     │                                  │
     ▼                                  ▼
┌─────────┐                       ┌─────────┐
│   WS    │ ◄────── gRPC ────────►│  RPC    │
│ Service │                       │ Service │
└─────────┘                       └─────────┘
     │                                  │
     │                                  │
     └──────────► Redis Stream ◄────────┘
                (消息中间件)
```
