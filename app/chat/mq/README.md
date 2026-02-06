# Chat 消息队列消费者服务

## 概述

Chat 消息队列消费者服务负责监听来自 Activity 服务的事件，实现事件驱动的自动化业务逻辑。

## 功能

### 已实现的消费者

1. **活动创建事件消费者** (`activity.created`)
   - 监听活动创建事件
   - 自动创建对应的群聊
   - 将活动创建者设为群主
   - 发送系统通知

2. **用户报名成功事件消费者** (`activity.member.joined`)
   - 监听用户报名成功事件
   - 自动将用户添加到活动群聊
   - 发送加入群聊通知

3. **用户取消报名事件消费者** (`activity.member.left`)
   - 监听用户取消报名事件
   - 自动将用户从活动群聊移除
   - 发送退出群聊通知

## 目录结构

```
app/chat/mq/
├── consumer/                           # 消费者实现
│   ├── events.go                      # 事件定义
│   ├── activity_created_consumer.go   # 活动创建事件消费者
│   ├── activity_member_joined_consumer.go  # 用户报名事件消费者
│   └── activity_member_left_consumer.go    # 用户取消报名事件消费者
├── internal/
│   ├── config/
│   │   └── config.go                  # 配置结构
│   └── svc/
│       └── servicecontext.go          # 服务上下文
├── etc/
│   └── consumer.yaml                  # 配置文件
└── consumer.go                        # 主入口
```

## 配置

配置文件位于 `etc/consumer.yaml`：

```yaml
Name: chat.consumer
Mode: dev

# Chat RPC 服务配置
ChatRpc:
  Etcd:
    Hosts:
      - 127.0.0.1:2379
    Key: chat.rpc

# Redis 配置
Redis:
  Host: 192.168.10.4:6379
  Pass: "123456"
  DB: 0

# 消息中间件配置
Messaging:
  ServiceName: chat.consumer
  EnableMetrics: true
  EnableGoZero: true
  Retry:
    MaxRetries: 3
    InitialInterval: 100ms
    MaxInterval: 10s
    Multiplier: 2.0
```

## 启动服务

### 开发环境

```bash
cd app/chat/mq
go run consumer.go -f etc/consumer.yaml
```

### 生产环境

```bash
# 编译
go build -o chat-consumer consumer.go

# 运行
./chat-consumer -f etc/consumer.yaml
```

## 事件格式

### 1. 活动创建事件 (activity.created)

```json
{
  "activity_id": "act_123456",
  "creator_id": "user_789",
  "title": "周末爬山活动",
  "created_at": "2024-01-01T10:00:00Z"
}
```

### 2. 用户报名成功事件 (activity.member.joined)

```json
{
  "activity_id": "act_123456",
  "user_id": "user_456",
  "joined_at": "2024-01-01T11:00:00Z"
}
```

### 3. 用户取消报名事件 (activity.member.left)

```json
{
  "activity_id": "act_123456",
  "user_id": "user_456",
  "left_at": "2024-01-01T12:00:00Z"
}
```

## 业务流程

### 活动创建流程

```
Activity 服务发布 activity.created 事件
    ↓
Chat 消费者接收事件
    ↓
调用 Chat RPC CreateGroup 创建群聊
    ↓
发送系统通知给创建者
```

### 用户报名流程

```
Activity 服务发布 activity.member.joined 事件
    ↓
Chat 消费者接收事件
    ↓
查询活动对应的群聊
    ↓
调用 Chat RPC AddGroupMember 添加成员
    ↓
发送系统通知给用户
```

## ⚠️ 已知问题

### 问题 1: 通过 activity_id 查询群聊

**现状**:
- `activity_member_joined_consumer.go` 和 `activity_member_left_consumer.go` 中使用 `GetGroupInfo(activity_id)` 查询群聊
- 但 `GetGroupInfo` RPC 方法接收的是 `group_id`，不是 `activity_id`

**影响**:
- 用户报名和取消报名的自动加群/退群功能无法正常工作

**解决方案**:

**方案 1: 添加新的 RPC 方法（推荐）**

在 `chat.proto` 中添加：

```protobuf
// GetGroupByActivityIdReq 通过活动ID获取群聊请求
message GetGroupByActivityIdReq {
  string activity_id = 1;  // 活动ID
}

// GetGroupByActivityIdResp 通过活动ID获取群聊响应
message GetGroupByActivityIdResp {
  GroupInfo group = 1;     // 群聊信息
}

service ChatService {
  // ... 其他方法

  // GetGroupByActivityId 通过活动ID获取群聊
  rpc GetGroupByActivityId(GetGroupByActivityIdReq) returns (GetGroupByActivityIdResp);
}
```

然后在 logic 中实现：

```go
func (l *GetGroupByActivityIdLogic) GetGroupByActivityId(in *chat.GetGroupByActivityIdReq) (*chat.GetGroupByActivityIdResp, error) {
    group, err := l.svcCtx.GroupModel.FindByActivityID(l.ctx, in.ActivityId)
    if err != nil {
        return nil, err
    }
    // ... 返回结果
}
```

**方案 2: 在事件中包含 group_id**

修改 Activity 服务，在创建活动时调用 Chat RPC 创建群聊，并将返回的 `group_id` 存储在活动表中。后续事件中包含 `group_id`：

```json
{
  "activity_id": "act_123456",
  "group_id": "grp_789",  // 新增字段
  "user_id": "user_456",
  "joined_at": "2024-01-01T11:00:00Z"
}
```

## 监控

### 日志

服务使用 Go-Zero 的日志系统，日志级别可在配置文件中设置。

关键日志：
- 消费者启动日志
- 事件接收日志
- RPC 调用成功/失败日志
- 错误和重试日志

### Prometheus 指标

启用 `EnableMetrics: true` 后，可通过 Prometheus 监控：

- `messaging_handler_duration_seconds`: 消息处理耗时
- `messaging_handler_total`: 消息处理总数
- `messaging_handler_errors_total`: 消息处理错误总数

## 错误处理

### 可重试错误

以下错误会自动重试（最多 3 次）：
- RPC 调用失败
- 网络错误
- 临时性故障

### 不可重试错误

以下错误不会重试，直接进入死信队列：
- 消息格式错误
- JSON 解析失败
- 数据验证失败

### 死信队列

失败的消息会进入死信队列（topic 后缀 `.dlq`）：
- `activity.created.dlq`
- `activity.member.joined.dlq`
- `activity.member.left.dlq`

可以手动处理死信队列中的消息。

## 依赖

- Chat RPC 服务（必须先启动）
- Redis（消息中间件）
- Etcd（服务发现）

## 故障排查

### 消费者无法启动

1. 检查 Redis 连接
2. 检查 Chat RPC 服务是否启动
3. 检查 Etcd 服务发现配置

### 消息未被消费

1. 检查 Activity 服务是否正确发布事件
2. 检查 topic 名称是否匹配
3. 查看日志中的错误信息
4. 检查死信队列

### RPC 调用失败

1. 检查 Chat RPC 服务状态
2. 检查网络连接
3. 查看 Chat RPC 服务日志

## 开发指南

### 添加新的消费者

1. 在 `consumer/events.go` 中定义事件结构
2. 创建新的消费者文件（如 `xxx_consumer.go`）
3. 实现消费者逻辑
4. 在 `consumer.go` 的 `registerConsumers` 函数中注册

示例：

```go
// consumer/new_event_consumer.go
type NewEventConsumer struct {
    chatRpc chat.ChatServiceClient
    logger  logx.Logger
}

func NewNewEventConsumer(chatRpc chat.ChatServiceClient) *NewEventConsumer {
    return &NewEventConsumer{
        chatRpc: chatRpc,
        logger:  logx.WithContext(context.Background()),
    }
}

func (c *NewEventConsumer) Subscribe(msgClient *messaging.Client) {
    msgClient.Subscribe("new.event", "handler-name", c.handleNewEvent)
}

func (c *NewEventConsumer) handleNewEvent(msg *message.Message) error {
    // 实现处理逻辑
    return nil
}
```

## 测试

### 单元测试

```bash
go test ./consumer/...
```

### 集成测试

1. 启动依赖服务（Redis, Etcd, Chat RPC）
2. 启动消费者服务
3. 使用 Activity 服务发布测试事件
4. 验证群聊是否正确创建/更新

## 许可证

本项目遵循 MIT 许可证。
