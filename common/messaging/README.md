# Messaging SDK 使用指南

## 概述

`common/messaging` 是基于 [Watermill](https://watermill.io/) 和 Redis Stream 构建的消息中间件 SDK，为 CampusHub 项目提供统一的消息发布/订阅能力。

### 核心特性

- ✅ **发布/订阅模式**：基于 Redis Stream 的可靠消息传递
- ✅ **Go-Zero 集成**：自动传播 `trace_id`、`span_id`，支持分布式链路追踪
- ✅ **Prometheus 监控**：内置消息处理指标（延迟、成功率、错误率）
- ✅ **自动重试**：支持指数退避的重试机制
- ✅ **死信队列（DLQ）**：自动处理失败消息
- ✅ **错误分类**：区分可重试和不可重试错误

---

## 快速开始

### 1. 安装依赖

```bash
go get github.com/ThreeDotsLabs/watermill
go get github.com/ThreeDotsLabs/watermill-redisstream
go get github.com/redis/go-redis/v9
```

### 2. 创建客户端

```go
package main

import (
    "context"
    "log"

    "activity-platform/common/messaging"
)

func main() {
    // 使用默认配置
    config := messaging.DefaultConfig()
    config.Redis.Addr = "localhost:6379"
    config.ServiceName = "my-service"

    // 创建客户端
    client, err := messaging.NewClient(config)
    if err != nil {
        log.Fatalf("创建客户端失败: %v", err)
    }
    defer client.Close()

    // 客户端已就绪，可以发布和订阅消息
}
```

### 3. 发布消息

```go
func publishExample(client *messaging.Client) {
    ctx := context.Background()

    // 发布消息到指定 topic
    payload := []byte(`{"user_id": 123, "action": "login"}`)
    err := client.Publish(ctx, "user.events", payload)
    if err != nil {
        log.Printf("发布消息失败: %v", err)
    }
}
```

### 4. 订阅消息

```go
func subscribeExample(client *messaging.Client) {
    // 订阅消息
    client.Subscribe("user.events", "user-event-handler", func(msg *message.Message) error {
        log.Printf("收到消息: %s", string(msg.Payload))

        // 处理消息逻辑
        // ...

        // 返回 nil 表示处理成功，消息会被 ACK
        return nil
    })

    // 启动 Router（阻塞）
    ctx := context.Background()
    if err := client.Run(ctx); err != nil {
        log.Printf("Router 停止: %v", err)
    }
}
```

---

## 配置详解

### 完整配置示例

```go
config := messaging.Config{
    // Redis 配置
    Redis: messaging.RedisConfig{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    },

    // 服务名称（用于消费者组和链路追踪）
    ServiceName: "user-service",

    // 启用 Prometheus 指标
    EnableMetrics: true,

    // 启用 Go-Zero trace_id 传播
    EnableGoZero: true,

    // 重试配置
    RetryConfig: messaging.RetryConfig{
        MaxRetries:      3,                      // 最大重试次数
        InitialInterval: 100 * time.Millisecond, // 初始重试间隔
        MaxInterval:     10 * time.Second,       // 最大重试间隔
        Multiplier:      2.0,                    // 退避倍数
    },

    // 死信队列配置
    DLQConfig: messaging.DLQConfig{
        Enabled:     true,
        TopicSuffix: ".dlq", // 死信队列 topic 后缀
    },
}
```

### 配置说明

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `Redis.Addr` | string | `localhost:6379` | Redis 服务器地址 |
| `Redis.Password` | string | `""` | Redis 密码 |
| `Redis.DB` | int | `0` | Redis 数据库编号 |
| `ServiceName` | string | `default-service` | 服务名称，用于消费者组 |
| `EnableMetrics` | bool | `true` | 是否启用 Prometheus 指标 |
| `EnableGoZero` | bool | `true` | 是否启用 Go-Zero 链路追踪 |
| `RetryConfig.MaxRetries` | int | `3` | 最大重试次数 |
| `RetryConfig.InitialInterval` | Duration | `100ms` | 初始重试间隔 |
| `RetryConfig.MaxInterval` | Duration | `10s` | 最大重试间隔 |
| `RetryConfig.Multiplier` | float64 | `2.0` | 退避倍数 |
| `DLQConfig.Enabled` | bool | `true` | 是否启用死信队列 |
| `DLQConfig.TopicSuffix` | string | `.dlq` | 死信队列 topic 后缀 |

---

## 核心功能

### 1. 发布消息

#### 基本发布

```go
ctx := context.Background()
payload := []byte(`{"event": "user_created", "user_id": 123}`)

err := client.Publish(ctx, "user.events", payload)
if err != nil {
    log.Printf("发布失败: %v", err)
}
```

#### 带 trace_id 的发布（Go-Zero 集成）

```go
import "activity-platform/common/messaging/gozero"

// 从 Go-Zero 的 context 中自动提取 trace_id
ctx := r.Context() // 来自 HTTP 请求的 context

// 发布时会自动注入 trace_id 到消息 Metadata
err := client.Publish(ctx, "user.events", payload)
```

### 2. 订阅消息

#### 基本订阅

```go
client.Subscribe("user.events", "user-handler", func(msg *message.Message) error {
    // 处理消息
    log.Printf("收到消息: %s", string(msg.Payload))

    // 返回 nil 表示成功
    return nil
})

// 启动 Router
ctx := context.Background()
client.Run(ctx)
```

#### 带错误处理的订阅

```go
client.Subscribe("user.events", "user-handler", func(msg *message.Message) error {
    // 解析消息
    var event UserEvent
    if err := json.Unmarshal(msg.Payload, &event); err != nil {
        // 返回不可重试错误（消息会进入 DLQ）
        return messaging.NewNonRetryableError(err)
    }

    // 处理业务逻辑
    if err := processEvent(event); err != nil {
        // 返回可重试错误（会自动重试）
        return messaging.NewRetryableError(err)
    }

    return nil
})
```

#### 在 Goroutine 中运行 Router

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
    if err := client.Run(ctx); err != nil {
        log.Printf("Router 停止: %v", err)
    }
}()

// 等待 Router 启动
<-client.Running()

// 继续其他操作...
```

### 3. Go-Zero 链路追踪集成

#### 在消息处理器中获取 trace_id

```go
import "activity-platform/common/messaging/gozero"

client.Subscribe("user.events", "user-handler", func(msg *message.Message) error {
    ctx := msg.Context()

    // 获取 trace_id
    traceID := gozero.GetTraceID(ctx)
    spanID := gozero.GetSpanID(ctx)
    serviceName := gozero.GetServiceName(ctx)

    log.Printf("处理消息 [trace_id=%s, span_id=%s, service=%s]",
        traceID, spanID, serviceName)

    // 使用 ctx 调用其他服务，trace_id 会自动传播
    return processWithContext(ctx, msg.Payload)
})
```

#### 手动注入 trace_id

```go
import "activity-platform/common/messaging/gozero"

ctx := context.Background()
ctx = gozero.WithTraceID(ctx, "custom-trace-id")
ctx = gozero.WithSpanID(ctx, "custom-span-id")

// 发布时会自动注入到消息
client.Publish(ctx, "user.events", payload)
```

### 4. 错误处理与重试

#### 可重试错误

```go
client.Subscribe("user.events", "user-handler", func(msg *message.Message) error {
    // 网络错误、临时故障等应该重试
    if err := callExternalAPI(); err != nil {
        return messaging.NewRetryableError(err)
    }
    return nil
})
```

#### 不可重试错误

```go
client.Subscribe("user.events", "user-handler", func(msg *message.Message) error {
    // 数据格式错误、业务逻辑错误等不应该重试
    if err := validateMessage(msg); err != nil {
        return messaging.NewNonRetryableError(err)
    }
    return nil
})
```

#### 自定义错误判断

```go
func handleMessage(msg *message.Message) error {
    err := processMessage(msg)
    if err != nil {
        // 判断是否可重试
        if messaging.IsRetryable(err) {
            log.Printf("可重试错误: %v", err)
        } else {
            log.Printf("不可重试错误: %v", err)
        }
        return err
    }
    return nil
}
```

### 5. Prometheus 监控

启用 `EnableMetrics: true` 后，SDK 会自动暴露以下指标：

| 指标名称 | 类型 | 说明 |
|---------|------|------|
| `messaging_handler_duration_seconds` | Histogram | 消息处理耗时 |
| `messaging_handler_total` | Counter | 消息处理总数 |
| `messaging_handler_errors_total` | Counter | 消息处理错误总数 |

标签（Labels）：
- `service`: 服务名称
- `topic`: 消息主题
- `handler`: 处理器名称
- `status`: 处理状态（`success` / `error`）

---

## 最佳实践

### 1. 消息格式

建议使用 JSON 格式，并定义清晰的消息结构：

```go
type UserEvent struct {
    EventType string    `json:"event_type"` // 事件类型
    UserID    int64     `json:"user_id"`
    Timestamp time.Time `json:"timestamp"`
    Data      any       `json:"data"`
}

// 发布
event := UserEvent{
    EventType: "user_created",
    UserID:    123,
    Timestamp: time.Now(),
    Data:      map[string]any{"email": "user@example.com"},
}
payload, _ := json.Marshal(event)
client.Publish(ctx, "user.events", payload)

// 订阅
client.Subscribe("user.events", "handler", func(msg *message.Message) error {
    var event UserEvent
    if err := json.Unmarshal(msg.Payload, &event); err != nil {
        return messaging.NewNonRetryableError(err)
    }
    // 处理事件
    return nil
})
```

### 2. Topic 命名规范

建议使用分层命名：

```
<domain>.<entity>.<action>

示例：
- user.account.created
- user.account.updated
- order.payment.completed
- notification.email.sent
```

### 3. Handler 命名规范

建议使用描述性名称：

```go
client.Subscribe("user.events", "user-event-processor", handler)
client.Subscribe("order.events", "order-notification-sender", handler)
```

### 4. 优雅关闭

```go
func main() {
    config := messaging.DefaultConfig()
    client, err := messaging.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 注册订阅
    client.Subscribe("user.events", "handler", messageHandler)

    // 启动 Router
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // 监听系统信号
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    go func() {
        <-sigChan
        log.Println("收到关闭信号，正在优雅关闭...")
        cancel() // 取消 context，停止 Router
    }()

    // 阻塞运行
    if err := client.Run(ctx); err != nil {
        log.Printf("Router 停止: %v", err)
    }
}
```

### 5. 错误处理策略

```go
func messageHandler(msg *message.Message) error {
    // 1. 验证消息格式（不可重试）
    if err := validateMessage(msg); err != nil {
        log.Printf("消息格式错误: %v", err)
        return messaging.NewNonRetryableError(err)
    }

    // 2. 业务逻辑处理（可重试）
    if err := processBusinessLogic(msg); err != nil {
        log.Printf("业务处理失败: %v", err)
        return messaging.NewRetryableError(err)
    }

    // 3. 外部调用（可重试）
    if err := callExternalService(msg); err != nil {
        log.Printf("外部服务调用失败: %v", err)
        return messaging.NewRetryableError(err)
    }

    return nil
}
```

---

## 完整示例

### 示例 1：用户服务发布事件

```go
package main

import (
    "context"
    "encoding/json"
    "log"

    "activity-platform/common/messaging"
)

type UserCreatedEvent struct {
    UserID int64  `json:"user_id"`
    Email  string `json:"email"`
}

func main() {
    // 创建客户端
    config := messaging.DefaultConfig()
    config.ServiceName = "user-service"
    config.Redis.Addr = "localhost:6379"

    client, err := messaging.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 发布用户创建事件
    event := UserCreatedEvent{
        UserID: 123,
        Email:  "user@example.com",
    }

    payload, _ := json.Marshal(event)
    ctx := context.Background()

    if err := client.Publish(ctx, "user.account.created", payload); err != nil {
        log.Printf("发布事件失败: %v", err)
    } else {
        log.Println("事件发布成功")
    }
}
```

### 示例 2：通知服务订阅事件

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "os/signal"
    "syscall"

    "activity-platform/common/messaging"
    "github.com/ThreeDotsLabs/watermill/message"
)

type UserCreatedEvent struct {
    UserID int64  `json:"user_id"`
    Email  string `json:"email"`
}

func main() {
    // 创建客户端
    config := messaging.DefaultConfig()
    config.ServiceName = "notification-service"
    config.Redis.Addr = "localhost:6379"

    client, err := messaging.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 订阅用户创建事件
    client.Subscribe("user.account.created", "send-welcome-email", func(msg *message.Message) error {
        var event UserCreatedEvent
        if err := json.Unmarshal(msg.Payload, &event); err != nil {
            return messaging.NewNonRetryableError(err)
        }

        log.Printf("发送欢迎邮件给用户 %d (%s)", event.UserID, event.Email)

        // 发送邮件逻辑
        if err := sendWelcomeEmail(event.Email); err != nil {
            return messaging.NewRetryableError(err)
        }

        return nil
    })

    // 优雅关闭
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    go func() {
        <-sigChan
        log.Println("正在关闭...")
        cancel()
    }()

    // 启动 Router
    log.Println("通知服务已启动")
    if err := client.Run(ctx); err != nil {
        log.Printf("服务停止: %v", err)
    }
}

func sendWelcomeEmail(email string) error {
    // 实现邮件发送逻辑
    return nil
}
```

---

## 测试

### 运行集成测试

```bash
# 确保 Redis 正在运行
docker run -d -p 6379:6379 redis:latest

# 运行测试
cd common/messaging/tests/integration
go test -v
```

### 测试示例

参考 `tests/integration/basic_test.go` 和 `tests/integration/gozero_test.go`。

---

## 故障排查

### 1. 连接 Redis 失败

**错误信息**：`连接 Redis 失败: dial tcp: connection refused`

**解决方案**：
- 检查 Redis 是否正在运行
- 检查 `Redis.Addr` 配置是否正确
- 检查防火墙设置

### 2. 消息未被消费

**可能原因**：
- Router 未启动（忘记调用 `client.Run()`）
- Handler 返回错误导致消息被重试
- 消费者组名称冲突

**解决方案**：
- 确保调用了 `client.Run(ctx)`
- 检查 Handler 的错误处理逻辑
- 为不同服务使用不同的 `ServiceName`

### 3. trace_id 未传播

**可能原因**：
- `EnableGoZero` 未启用
- Context 中没有 trace_id

**解决方案**：
- 设置 `config.EnableGoZero = true`
- 确保发布消息时的 context 包含 trace_id

---

## 依赖项

- [Watermill](https://watermill.io/) - 消息流处理库
- [Redis](https://redis.io/) - 消息存储
- [Prometheus](https://prometheus.io/) - 监控指标（可选）
- [Go-Zero](https://go-zero.dev/) - 链路追踪（可选）

---

## 许可证

本项目遵循 MIT 许可证。
