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
| 认证 | JWT (双Token) |
| 链路追踪 | Jaeger |
| 消息队列 | Kafka (可选) |

## 微服务架构特性

本项目是真正的微服务架构，具备以下核心特性：

| 特性        | 说明 |
|-----------|------|
| 服务注册/发现   | Etcd，服务启动自动注册，Gateway 动态发现 |
| 负载均衡      | P2C 算法，自适应流量分发 |
| 熔断降级      | go-zero 内置，故障快速失败 |
| 链路追踪      | Jaeger，跨服务调用链可视化 |
| 统一API网关   | Gateway 层，协议转换 |
| **服务间通信** | RPC 服务之间通过 gRPC + Etcd 相互调用 |

详细说明见 `docs/design/microservice-architecture.md`

### 服务间通信架构

```
┌──────────┐     ┌──────────┐     ┌──────────┐
│ User RPC │◄────│Activity  │     │ Chat RPC │
│  :9001   │     │   RPC    │◄────│  :9003   │
└──────────┘     │  :9002   │     └──────────┘
     ▲           └──────────┘          │
     │                                 │
     └─────────────────────────────────┘
                 gRPC (服务间通信)
```

- Activity → User：获取组织者信息、验证用户资格
- Chat → User：获取用户信息（推送通知时）
- Chat → Activity：获取活动信息（活动提醒通知）


## 项目结构

```
activity-platform/
├── app/                          # 应用服务
│   ├── demo/rpc/                 # ⭐ 示例服务（开发规范参考）
│   ├── gateway/api/              # API网关 (BFF层)
│   │   └── internal/
│   │       ├── middleware/       # 中间件 ✓
│   │       └── ...
│   ├── user/rpc/                 # 用户服务 - B同学
│   ├── activity/rpc/             # 活动服务 - C/D同学
│   ├── chat/rpc/                 # 聊天服务 - E同学
│
├── common/                       # 公共组件 ✓
│   ├── errorx/                   # 错误码 ✓
│   ├── response/                 # 响应封装 ✓
│   ├── ctxdata/                  # 上下文 ✓
│   ├── constants/                # 常量 ✓
│   └── utils/
│       ├── jwt/                  # JWT工具 (B同学实现)
│       ├── encrypt/              # 加密脱敏 ✓
│       └── validate/             # 验证器 ✓
├── deploy/
│   ├── docker/                   # Docker配置 ✓
│   └── sql/                      # 数据库脚本 ✓
└── scripts/                      # 脚本工具
```

## 快速开始

### 1. 环境要求

- Go 1.21+
- Docker & Docker Compose
- protoc (Protocol Buffers 编译器)

### 2. 初始化项目

```bash
# 下载依赖
go mod tidy

# 启动基础设施（MySQL + Redis + Etcd + Jaeger）
cd deploy/docker
docker-compose up -d

# 验证 Etcd 启动
docker exec activity-etcd etcdctl endpoint health

# 初始化数据库
make db-init
```

### 3. 运行服务

```bash
# 终端1: 用户服务（自动注册到 Etcd）
cd app/user/rpc
go run user.go

# 终端2: 网关服务（从 Etcd 发现服务）
cd app/gateway/api
go run gateway.go
```

### 4. 验证微服务

```bash
# 查看注册的服务
docker exec activity-etcd etcdctl get --prefix /services/

# 访问 Jaeger UI 查看链路追踪
# http://localhost:16686
```

## 开发指南

### ⭐ 重要：先看 Demo 服务

**开发自己的服务前，请先阅读 `app/demo/` 目录**，包含：
- 完整的目录结构
- 代码规范和注释
- Proto 文件定义示例
- Model/Logic/Server 层示例
- 详细的 README 文档

### 开发流程

1. **阅读 Demo** - 理解项目规范
2. **定义 Proto** - 在 `app/{服务}/rpc/` 下创建 `.proto` 文件
3. **生成代码** - `protoc --go_out=. --go-grpc_out=. xxx.proto`
4. **实现 Model** - 数据库操作层
5. **实现 Logic** - 业务逻辑层
6. **实现 Server** - gRPC 服务入口
7. **测试接口** - 使用 grpcurl 测试

### Git 分支规范

```
main                    # 主分支 (保护)
├── develop             # 开发分支
│   ├── feature/user-login       # B: 登录注册
│   ├── feature/activity-crud    # C: 活动CRUD
│   ├── feature/registration     # D: 报名
│   ├── feature/websocket        # E: WebSocket
│   └── ...
```

### 代码规范

```bash
# 提交前执行
go fmt ./...     # 格式化
go vet ./...     # 静态检查
go test ./...    # 运行测试
```

## 人员分配

| 成员 | 模块 | 核心任务 |
|------|------|----------|
| A | 学生认证+信用分 | OCR认证、规则引擎 |
| B | 注册登录+验证码 | JWT双Token、限流 |
| C | 活动CRUD+缓存+搜索 | Cache-Aside、ES |
| D | 报名(高并发)+签到 | Redis预扣、MQ |
| E | WebSocket+通知 | 心跳保活、ACK |

## 服务端口

| 服务 | 端口 | 说明 |
|------|------|------|
| Gateway API | 8080 | HTTP 入口 |
| User RPC | 9001 | 用户服务 |
| Activity RPC | 9002 | 活动服务（含报名签到） |
| Chat RPC | 9003 | 消息通知服务 |
| Demo RPC | 9100 | 示例服务 |
| MySQL | 3306 | 数据库 |
| Redis | 6379 | 缓存 |
| Etcd | 2379 | 服务注册 |
| Jaeger UI | 16686 | 链路追踪 |


## License
MIT
