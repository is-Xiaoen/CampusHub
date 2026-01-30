# 校园活动平台 (Campus Activity Platform)

基于 go-zero 微服务框架构建的校园活动管理平台。

## 技术栈

| 类别 | 技术 |
|------|------|
| 框架 | go-zero (微服务) |
| 数据库 | MySQL 8.0 |
| 缓存 | Redis 7 |
| 服务注册 | Etcd |
| 通信 | gRPC + RESTful |
| 认证 | JWT |
| 网关 | Nginx |
| 链路追踪 | Jaeger |

## 微服务架构

### 架构图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              客户端                                      │
│                     (Web / App / 小程序)                                 │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │ HTTPS
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      Nginx 网关 (:8888)                                  │
│                    （路由分发，不做业务逻辑）                               │
└─────────────────────────────────────────────────────────────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        │                       │                       │
        ▼                       ▼                       ▼
┌───────────────┐       ┌───────────────┐       ┌───────────────┐
│   user-api    │       │ activity-api  │       │   chat-api    │
│    :8001      │       │    :8002      │       │    :8003      │
│   (jwt:Auth)  │       │   (jwt:Auth)  │       │   (jwt:Auth)  │
└───────┬───────┘       └───────┬───────┘       └───────┬───────┘
        │ gRPC                  │ gRPC                  │ gRPC
        ▼                       ▼                       ▼
┌───────────────┐       ┌───────────────┐       ┌───────────────┐
│   user-rpc    │◄─────►│ activity-rpc  │◄─────►│   chat-rpc    │
│    :9001      │       │    :9002      │       │    :9003      │
└───────────────┘       └───────────────┘       └───────────────┘
        │                       │                       │
        └───────────────────────┼───────────────────────┘
                                ▼
                    ┌───────────────────────┐
                    │   MySQL / Redis / Etcd │
                    └───────────────────────┘
```

### 架构特点

| 特性 | 说明 |
|------|------|
| Nginx 网关 | 统一入口，只做路由分发 |
| 各服务独立 API | 每个服务有自己的 HTTP 接口层 |
| go-zero jwt:Auth | 各 API 服务使用 go-zero 内置 JWT 鉴权 |
| 共享 AccessSecret | 所有 API 使用相同密钥，Token 跨服务有效 |
| RPC 内网通信 | gRPC 服务仅在内网，通过 Etcd 发现 |

## 项目结构

```
activity-platform/
├── app/
│   ├── user/
│   │   ├── api/                  # 用户 HTTP 接口（杨春路）
│   │   │   ├── desc/user.api     # API 定义文件
│   │   │   ├── etc/              # 配置
│   │   │   └── internal/         # handler/logic/svc
│   │   └── rpc/                  # 用户 gRPC 服务
│   │
│   ├── activity/
│   │   ├── api/                  # 活动 HTTP 接口（马肖阳）
│   │   └── rpc/                  # 活动 gRPC 服务
│   │
│   ├── chat/
│   │   ├── api/                  # 聊天 HTTP 接口（马华恩）
│   │   ├── rpc/                  # 聊天 gRPC 服务
│   │   └── ws/                   # WebSocket 服务
│   │
│   ├── demo/rpc/                 # 示例服务（开发规范参考）
│   │
│   └── gateway/                  # [已废弃] 见 DEPRECATED.md
│
├── common/                       # 公共组件
│   ├── errorx/                   # 错误码
│   ├── response/                 # 响应封装
│   ├── ctxdata/                  # 上下文
│   ├── constants/                # 常量
│   └── utils/
│       ├── jwt/                  # JWT 工具（杨春路实现）
│       ├── encrypt/              # 加密脱敏
│       └── validate/             # 验证器
│
├── deploy/
│   ├── nginx/                    # Nginx 网关配置
│   │   └── gateway.conf
│   ├── docker/                   # Docker 配置
│   └── sql/                      # 数据库脚本
│
└── scripts/                      # 脚本工具
```

## 快速开始

### 1. 环境要求

- Go 1.21+
- Docker & Docker Compose
- goctl (go-zero 代码生成工具)

### 2. 安装 goctl

```bash
go install github.com/zeromicro/go-zero/tools/goctl@latest
```

### 3. 启动基础设施

```bash
cd deploy/docker
docker-compose up -d
```

### 4. 生成 API 代码

```bash
# 以 user-api 为例
cd app/user/api
goctl api go -api desc/user.api -dir . -style go_zero
```

### 5. 启动服务

```bash
# 终端1: User RPC
cd app/user/rpc && go run user.go

# 终端2: User API
cd app/user/api && go run user.go

# 终端3: Nginx（或直接访问各 API）
# 开发时可直接访问 localhost:8001
```

## 开发指南

### API 服务开发流程

1. **定义 .api 文件** - 在 `desc/xxx.api` 定义接口
2. **生成代码** - `goctl api go -api desc/xxx.api -dir . -style go_zero`
3. **实现 logic** - 在 `internal/logic/` 实现业务逻辑
4. **配置 RPC 客户端** - 在 `internal/svc/servicecontext.go` 添加
5. **测试接口** - 使用 Postman 或 curl

### RPC 服务开发流程

1. **定义 .proto 文件** - 在 `xxx.proto` 定义接口
2. **生成代码** - `goctl rpc protoc xxx.proto --go_out=. --go-grpc_out=. --zrpc_out=.`
3. **实现 logic** - 在 `internal/logic/` 实现业务逻辑
4. **实现 model** - 在 `internal/model/` 实现数据库操作
5. **测试接口** - 使用 grpcurl

### JWT 鉴权说明

- 在 `.api` 文件中使用 `jwt: Auth` 声明需要鉴权的路由
- 所有 API 服务的 `Auth.AccessSecret` 必须一致
- user-api 签发的 Token 在其他 API 服务中也能验证

## 服务端口

| 服务 | HTTP 端口 | RPC 端口 | 说明 |
|------|-----------|----------|------|
| Nginx 网关 | 8888 | - | 统一入口 |
| user | 8001 | 9001 | 用户服务 |
| activity | 8002 | 9002 | 活动服务 |
| chat | 8003 | 9003 | 聊天服务 |
| demo | - | 9100 | 示例服务 |

## 人员分配

| 成员 | 服务 | 核心任务 |
|------|------|----------|
| 杨春路 | user-api/rpc | 注册登录、JWT 签发、用户信息 |
| 马肖阳 | activity-api/rpc | 活动 CRUD、报名、签到 |
| 马华恩 | chat-api/rpc/ws | 消息、WebSocket、通知 |

## License

MIT
