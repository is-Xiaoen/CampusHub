# Chat API 服务

## 负责人

马华恩（E同学）

## 服务说明

chat-api 是聊天服务的 HTTP 接口层，负责：
- 聊天室列表
- 消息历史查询
- 消息发送（HTTP 方式）

**注意**：WebSocket 实时通信在 `chat/ws` 目录单独实现。

## 目录结构

```
chat/api/
├── desc/
│   └── chat.api          # API 接口定义文件
├── etc/
│   └── chat-api.yaml     # 配置文件
├── internal/
│   ├── config/           # 配置结构体（goctl 生成）
│   ├── handler/          # HTTP 处理器（goctl 生成）
│   ├── logic/            # 业务逻辑（需要你实现）
│   ├── svc/              # 服务上下文（goctl 生成）
│   └── types/            # 请求响应类型（goctl 生成）
├── chat.go               # 入口文件
└── README.md
```

## 开发步骤

### 1. 生成代码

```bash
cd app/chat/api
goctl api go -api desc/chat.api -dir . -style go_zero
```

### 2. 修改配置

编辑 `etc/chat-api.yaml`：
- **重要**：`Auth.AccessSecret` 必须与 user-api 一致
- 配置 `ChatRpc` 的 Etcd 地址

### 3. 实现业务逻辑

在 `internal/logic/` 目录下实现各接口的业务逻辑。

### 4. 启动服务

```bash
go run chat.go -f etc/chat-api.yaml
```

## 接口列表

| 方法 | 路径 | 说明 | 需要登录 |
|------|------|------|----------|
| GET | /api/v1/chat/rooms | 聊天室列表 | 是 |
| GET | /api/v1/chat/rooms/:id/messages | 消息历史 | 是 |
| POST | /api/v1/chat/messages | 发送消息 | 是 |

## 端口

- HTTP: 8003
- 对应 RPC: 9003
- WebSocket: 单独配置（在 ws 目录）
