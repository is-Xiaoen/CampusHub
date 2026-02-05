# 项目名称：基于 Go-Zero 的分布式社交活动 IM 平台

## 1. 项目背景 (Project Background)

本项目是一个集社交、活动发布与即时通讯（IM）于一体的高并发微服务平台。旨在解决用户"发起活动难、沟通成本高"的痛点。用户可以发布线下社交活动，系统会自动为报名参与的用户创建专属群聊，实现从"报名"到"沟通"的无缝闭环。

## 2. 技术栈 (Tech Stack)

- **开发语言**: Golang (1.24)
- **微服务框架**: Go-Zero (gRPC + HTTP)
- **消息中间件**: Redis (后续可能会换RedPanda) + Watermill (Go)
- **存储层**: MySQL (持久化), Redis (缓存/会话), GORM/Sqlx
- **通信协议**: WebSocket (实时消息), Protobuf (RPC)
- **基础设施**: Docker, Docker Compose

## 3. 系统架构设计 (Architecture)

采用 **微服务架构** 与 **事件驱动架构 (EDA)** 相结合的设计：

- **服务拆分**: 将系统拆分为 **3 个核心服务**：
  - **用户服务 (User Service)**: 用户注册、登录、个人信息管理
  - **活动服务 (Activity Service)**: 活动发布、报名、审核管理
  - **IM 服务 (IM Service)**: 即时通讯，包含群聊管理和系统通知两大模块
- **通信模式**: 服务间同步调用使用 gRPC，异步解耦使用 Pub/Sub 模型。
- **消息流转**: 实现了 API Gateway -> Kafka (Redpanda) -> Consumer Group -> DB 的削峰填谷链路。

## 4. 我负责的模块 (My Responsibilities)

作为后端核心开发者及消息中间件架构负责人，主要负责以下模块：

### A. 消息中间件基础设施 (Infrastructure)

- **技术选型**: 针对团队开发环境与生产环境的差异，主导选型 Redpanda 代替传统 Kafka，降低了 80% 的运维成本与内存占用，同时保持生产环境的 Kafka 兼容性。
- **SDK 封装**: 基于 Watermill 框架封装了统一的 MQ SDK (common/mq)，屏蔽了底层驱动细节，为团队提供了开箱即用的 Publish/Subscribe 接口，实现了业务逻辑与具体 MQ 的完全解耦。
- **可靠性设计**: 实现了 At-least-once 投递机制，配置了 Retry (自动重试) 和 Poison Queue (死信队列) 中间件，防止单条异常消息阻塞整个消费链路。

### B. IM 即时通讯服务 (IM Service)

- **连接管理**: 设计并实现了基于 Gorilla/WebSocket 的 ConnManager，利用 RWMutex 和 Map 管理数千个并发长连接，支持用户上线、下线及消息路由。
- **心跳保活**: 实现了 WebSocket 的 Ping/Pong 心跳机制 与 ReadDeadline 断线检测，有效解决了公网环境下（如 NAT 超时）导致的连接假死问题。
- **消息分发**: 设计了 WebSocket -> Kafka -> Storage 的异步存库流程，确保高并发下的消息不丢失，并实现了基于 GroupId 的 Partition 分区策略，保证了群聊消息的严格时序性。

### C. IM 服务 - 群组与通知模块 (Group & Notification Modules)

- **业务闭环**: 实现了"活动发布自动建群"的业务逻辑，通过 gRPC 串联活动服务与 IM 服务的群聊模块。
- **系统通知**: 基于事件总线（Event Bus）实现了系统通知模块。当用户加入活动或活动变更时，异步触发系统通知，并通过 WebSocket 实时推送到客户端。

---

## 🎓 业务场景梳理

### 核心流程

```
发布活动 → 自动创建群聊
    ↓
其他用户报名
    ↓
报名成功 → 自动加群 + 发送通知
```

这是典型的事件驱动架构，需要通过消息中间件解耦服务。

---

## 🏗️ 服务架构设计

整体架构是：

```
┌──────────────────┐      ┌──────────────────┐      ┌─────────────────────────┐
│  User 服务       │      │  Activity 服务   │      │      IM 服务            │
│  (用户管理)      │      │  (活动管理)      │      │   (即时通讯)            │
│                  │      │                  │      │                         │
│  - 用户注册      │      │  - 发布活动      │      │  【群聊模块】           │
│  - 用户登录      │      │  - 报名活动      │      │  - 创建群聊             │
│  - 个人信息      │      │  - 报名审核      │      │  - 成员管理             │
│                  │      │                  │      │  - WebSocket 消息收发   │
│                  │      │                  │      │                         │
│                  │      │                  │      │  【系统通知模块】       │
│                  │      │                  │      │  - 通知生成             │
│                  │      │                  │      │  - 通知推送             │
│                  │      │                  │      │  - 消息持久化           │
└────────┬─────────┘      └────────┬─────────┘      └────────┬────────────────┘
         │                         │                         │
         │                         │                         │
         └─────────────────────────┴─────────────────────────┘
                                   │
                          ┌────────▼────────┐
                          │   Redpanda      │
                          │  (消息中间件)    │
                          └─────────────────┘
                                   │
          ┌────────────────────────┴────────────────────────┐
          │                        │                        │
     ┌────▼─────┐          ┌──────▼──────┐         ┌──────▼──────┐
     │  Topic:  │          │   Topic:    │         │   Topic:    │
     │ activity │          │   group     │         │ notification│
     │ .created │          │  .member    │         │             │
     └──────────┘          │  .added     │         └─────────────┘
                           └─────────────┘
```

### 你负责的部分

作为 IM 服务的核心开发者，主要负责以下模块：

#### 1. IM 服务 - 群聊模块 (Group Chat Module)

- 监听 `activity.created` 事件 → 自动创建群聊
- 监听 `activity.member.joined` 事件 → 自动加人入群
- 提供 RPC 接口：解散群、踢人、查询群信息
- 发布事件：`group.created`、`group.member.added`、`group.member.removed`
- WebSocket 长连接管理
- 群聊消息收发（文字）
- 消息持久化到 MySQL

#### 2. IM 服务 - 系统通知模块 (System Notification Module)

- 监听各种事件 → 生成通知
- 通知类型：注册成功、报名成功、被踢出群等
- 通知持久化
- 应用内通知展示
- 离线消息推送

#### 3. 消息中间件 SDK（common/mq）

- 封装 Watermill + Redis
- 统一的 Pub/Sub 接口
- 消息重试、死信队列
- 监控和日志

---

## 📋 关键事件定义

### 订阅的事件（来自其他服务）

```go
// 活动创建事件（Activity 服务发布）
type ActivityCreatedEvent struct {
    ActivityID   string
    CreatorID    string
    Title        string
    CreatedAt    time.Time
}

// 用户报名成功事件（Activity 服务发布）
type ActivityMemberJoinedEvent struct {
    ActivityID   string
    UserID       string
    JoinedAt     time.Time
}

```