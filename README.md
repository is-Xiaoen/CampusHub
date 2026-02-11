# CampusHub - 校园活动平台

[![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![go-zero](https://img.shields.io/badge/go--zero-v1.7.3-blue)](https://go-zero.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

基于 **go-zero** 微服务框架构建的校园活动发布、报名、签到平台。采用 Clean Architecture 设计，支持分布式事务、全文搜索、实时通讯。

---

## 功能特性

**活动管理**
- 活动 CRUD（创建 / 更新 / 删除 / 详情查询）
- 表驱动状态机（草稿 → 待审核 → 已发布 → 进行中 → 已结束）
- 多条件筛选 + 深分页优化（延迟关联，覆盖索引）
- 热门活动排行榜（Top10）
- 浏览量统计（防刷 + 异步批量更新）

**报名签到**
- 活动报名 / 取消报名
- 电子票券（TOTP 动态码核销）
- 幂等签到保障

**用户体系**
- 注册登录 / JWT 鉴权
- 信用分体系
- 学生认证（OCR 识别）

**即时通讯**
- WebSocket 实时消息推送
- 群聊 / 单聊
- 消息通知

**基础设施**
- DTM 分布式事务（SAGA 模式，子事务屏障保障幂等）
- Elasticsearch 全文搜索（IK 中文分词）
- Redis 缓存三防一体（穿透 / 击穿 / 雪崩）
- Watermill 事件驱动消息队列
- Jaeger 全链路追踪
- GitHub Actions CI/CD + K3s 自动部署

---

## 架构设计

```
                          ┌──────────────────────┐
                          │      客户端            │
                          │  Web / App / 小程序    │
                          └──────────┬───────────┘
                                     │ HTTPS
                                     ▼
                          ┌──────────────────────┐
                          │   Nginx 网关 (:8888)   │
                          │    路由分发 + 负载      │
                          └──────────┬───────────┘
                                     │
              ┌──────────────────────┼──────────────────────┐
              │                      │                      │
              ▼                      ▼                      ▼
     ┌────────────────┐    ┌────────────────┐    ┌────────────────┐
     │   user-api     │    │ activity-api   │    │   chat-api     │
     │    :8001       │    │    :8002       │    │    :8003       │
     │   JWT Auth     │    │   JWT Auth     │    │   JWT Auth     │
     └───────┬────────┘    └───────┬────────┘    └───────┬────────┘
             │ gRPC                │ gRPC                │ gRPC
             ▼                     ▼                     ▼
     ┌────────────────┐    ┌────────────────┐    ┌────────────────┐
     │   user-rpc     │◄──►│ activity-rpc   │◄──►│   chat-rpc     │
     │    :9001       │    │    :9002       │    │    :9003       │
     └────────────────┘    └───────┬────────┘    └────────────────┘
                                   │
                    ┌──────────────┼──────────────┐
                    ▼              ▼              ▼
              ┌──────────┐  ┌──────────┐  ┌──────────┐
              │  MySQL   │  │  Redis   │  │   Etcd   │
              │   8.0    │  │    7     │  │  v3.5    │
              └──────────┘  └──────────┘  └──────────┘
                    ▲              ▲
                    │              │
              ┌──────────┐  ┌──────────┐
              │   DTM    │  │  Jaeger  │
              │  Server  │  │ Tracing  │
              └──────────┘  └──────────┘
```

### 调用规则

| 调用方向 | 说明 |
|---------|------|
| API → RPC | 通过 gRPC 调用，API 层**禁止直接访问数据库** |
| RPC → Model | 直接导入同服务的 Model 层 |
| RPC → 其他 RPC | 跨服务**必须走 gRPC**，禁止导入其他服务的 Model |

---

## 技术栈

| 类别 | 技术 | 版本 | 说明 |
|------|------|------|------|
| **语言** | Go | 1.24 | |
| **微服务框架** | go-zero | v1.7.3 | 内置服务治理、熔断、限流 |
| **RPC** | gRPC + Protobuf | gRPC v1.67 | 高性能序列化 |
| **ORM** | GORM | v1.25.5 | 数据持久层 |
| **数据库** | MySQL | 8.0 | 主存储 |
| **缓存** | Redis | 7 | 缓存 + 消息队列 + 分布式锁 |
| **搜索引擎** | Elasticsearch | 7.17 LTS | IK 中文分词，olivere/elastic v7 |
| **分布式事务** | DTM | v1.18.7 | SAGA 模式 + 子事务屏障 |
| **消息队列** | Watermill | v1.5.0 | Redis Stream 驱动 |
| **服务发现** | Etcd | v3.5 | 服务注册与发现 |
| **认证** | JWT | golang-jwt v4 | 跨服务 Token 互通 |
| **WebSocket** | Gorilla WebSocket | v1.5.0 | 实时通讯 |
| **链路追踪** | Jaeger | OpenTelemetry | 全链路可视化 |
| **监控** | Prometheus | client v1.23 | 指标采集 |
| **网关** | Nginx | latest | 路由分发 |
| **容器化** | Docker | 多阶段构建 | 镜像锁定 digest |
| **CI/CD** | GitHub Actions | - | PR 质检 + 自动构建部署 |
| **容器编排** | K3s | - | 轻量 Kubernetes |
| **对象存储** | 七牛云 | go-sdk v7 | 图片存储 |

---

## 项目结构

```
CampusHub/
├── app/                            # 微服务目录
│   ├── user/                       # 用户服务
│   │   ├── api/                    #   HTTP 接口层
│   │   │   ├── desc/user.api       #     API 定义
│   │   │   ├── etc/                #     配置文件
│   │   │   └── internal/           #     handler / logic / svc
│   │   ├── rpc/                    #   gRPC 服务层
│   │   │   ├── user.proto          #     Proto 定义
│   │   │   └── internal/           #     logic / svc / cron / ocr
│   │   └── model/                  #   数据模型（与 api/rpc 同级）
│   │
│   ├── activity/                   # 活动服务
│   │   ├── api/                    #   HTTP 接口层
│   │   │   └── desc/               #     activity.api + ticket.api + types.api
│   │   ├── rpc/                    #   gRPC 服务层
│   │   │   ├── activity.proto      #     Proto 定义（550+ 行）
│   │   │   └── internal/
│   │   │       ├── logic/          #       核心业务逻辑（18+ Logic）
│   │   │       ├── cache/          #       缓存模块（Activity / Category / Hot）
│   │   │       ├── search/         #       ES 搜索模块
│   │   │       ├── cron/           #       定时任务（状态自动流转）
│   │   │       ├── dtm/            #       DTM 分布式事务客户端
│   │   │       └── mq/             #       消息队列消费者
│   │   └── model/                  #   数据模型
│   │
│   └── chat/                       # 聊天服务
│       ├── api/                    #   HTTP 接口层
│       ├── rpc/                    #   gRPC 服务层
│       ├── ws/                     #   WebSocket 服务
│       └── model/                  #   数据模型
│
├── common/                         # 公共组件
│   ├── errorx/                     #   统一错误码 + 错误封装
│   ├── response/                   #   HTTP 响应格式化
│   ├── constants/                  #   全局常量（状态、Redis Key）
│   ├── ctxdata/                    #   上下文数据（JWT 用户提取）
│   ├── middleware/                 #   HTTP 中间件
│   ├── interceptor/                #   gRPC 拦截器
│   ├── messaging/                  #   Watermill 事件驱动
│   └── utils/                      #   工具库（JWT / 加密 / 验证）
│
├── deploy/                         # 部署配置
│   ├── docker/                     #   Docker Compose + Dockerfile
│   ├── nginx/                      #   Nginx 网关配置
│   ├── sql/                        #   数据库初始化脚本
│   └── k8s/                        #   Kubernetes 部署（预留）
│
├── docs/                           # 设计文档
│   ├── plan/                       #   模块设计方案
│   ├── api/                        #   接口文档
│   └── guide/                      #   开发指南
│
├── .github/workflows/ci-cd.yml    # CI/CD 流水线
├── Makefile                        # 构建脚本
├── go.mod
└── go.sum
```

> **注意**：Model 层与 `api/`、`rpc/` 同级，**不在** `internal/` 目录下，方便跨层引用。

---

## 快速开始

### 环境要求

| 工具 | 版本 | 说明 |
|------|------|------|
| Go | 1.24+ | 编程语言 |
| Docker | 20.10+ | 容器运行时 |
| Docker Compose | v2+ | 编排工具 |
| goctl | latest | go-zero 代码生成 |
| protoc | 3.x | Protobuf 编译器 |

### 1. 克隆项目

```bash
git clone https://github.com/APXT-CR/CampusHub.git
cd CampusHub
```

### 2. 安装 goctl

```bash
go install github.com/zeromicro/go-zero/tools/goctl@latest
```

### 3. 启动基础设施

```bash
# 启动 MySQL、Redis、Etcd、DTM、Jaeger
docker-compose -f deploy/docker/docker-compose.yaml up -d

# 初始化数据库
make db-init
```

### 4. 下载依赖

```bash
go mod download
```

### 5. 启动服务

```bash
# 终端 1：User RPC
cd app/user/rpc && go run user.go -f etc/user.yaml

# 终端 2：Activity RPC
cd app/activity/rpc && go run activity.go -f etc/activity.yaml

# 终端 3：User API
cd app/user/api && go run user.go -f etc/user.yaml

# 终端 4：Activity API
cd app/activity/api && go run activity.go -f etc/activity.yaml
```

### 6. 验证服务

```bash
# 活动列表
curl http://localhost:8002/api/v1/activity/lists?page=1&page_size=10

# 分类列表
curl http://localhost:8002/api/v1/activity/categories

# 搜索活动
curl http://localhost:8002/api/v1/activity/search?keyword=篮球
```

---

## API 概览

### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/activity/lists` | 活动列表（分页 + 筛选） |
| GET | `/api/v1/activity/:id` | 活动详情 |
| GET | `/api/v1/activity/search` | 搜索活动 |
| GET | `/api/v1/activity/hot` | 热门活动 Top10 |
| GET | `/api/v1/activity/categories` | 分类列表 |
| GET | `/api/v1/activity/tags` | 标签列表 |
| POST | `/api/v1/activity/:id/view` | 增加浏览量 |

### 需要登录（JWT）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/activity` | 创建活动 |
| PUT | `/api/v1/activity/:id` | 更新活动 |
| DELETE | `/api/v1/activity/:id` | 删除活动 |
| POST | `/api/v1/activity/:id/submit` | 提交审核 |
| POST | `/api/v1/activity/:id/cancel` | 取消活动 |
| GET | `/api/v1/activity/my/created` | 我创建的活动 |
| POST | `/api/v1/activity/:id/register` | 报名活动 |

### 管理员接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/admin/activity/:id/approve` | 审核通过 |
| POST | `/api/v1/admin/activity/:id/reject` | 审核拒绝 |

> 完整接口文档见 [`docs/api/`](docs/api/)

---

## 核心技术实现

### 表驱动状态机

活动有 7 种状态，使用表驱动模式管理状态流转，配合乐观锁 + 指数退避重试处理并发冲突：

```
Draft(草稿) ──► Pending(待审核) ──► Published(已发布) ──► Ongoing(进行中) ──► Finished(已结束)
    │                                     │                    │
    └──────► Published(直接发布)           └─► Cancelled(取消)  └─► Cancelled(取消)
                                Rejected(拒绝) ◄── Pending
```

### 深分页优化

```
第 1~20 页：常规 LIMIT/OFFSET
第 20~100 页：延迟关联（先查 ID 再回表，利用覆盖索引）
超过 100 页：直接拒绝
```

### 缓存三防一体

| 问题 | 方案 | 实现 |
|------|------|------|
| 缓存穿透 | 空值缓存 | go-zero cache.Take 自动处理 |
| 缓存击穿 | singleflight | go-zero 内置 |
| 缓存雪崩 | 随机 TTL | 5min +/- 10% 随机过期 |

### DTM 分布式事务

跨服务操作使用 SAGA 模式，子事务屏障保障幂等性、防空补偿、防悬挂：

```
创建活动 SAGA:
  Step 1: CreateActivityAction     → Compensate: CreateActivityCompensate
  Step 2: IncrTagUsageCount        → Compensate: DecrTagUsageCount
```

---

## 开发指南

### 代码生成

```bash
# 生成 API 代码
cd app/activity/api
goctl api go -api desc/activity.api -dir . --style go_zero

# 生成 RPC 代码（单服务）
cd app/activity/rpc
goctl rpc protoc activity.proto --go_out=. --go-grpc_out=. --zrpc_out=. --style go_zero

# 生成 RPC 代码（多服务，需要 -m）
goctl rpc protoc activity.proto --go_out=. --go-grpc_out=. --zrpc_out=. --style go_zero -m
```

> `--style go_zero` 参数**必须加上**，否则生成的文件名与项目不一致会导致编译错误。

### 常用 Make 命令

```bash
make build            # 编译所有服务
make fmt              # 代码格式化
make vet              # 静态检查
make lint             # golangci-lint 检查
make test             # 运行测试
make test-coverage    # 测试覆盖率报告
make test-race        # 并发竞态检测
make docker-up        # 启动基础设施
make docker-down      # 停止基础设施
make db-init          # 初始化数据库
make clean            # 清理构建产物
```

---

## 部署

### Docker Compose 部署

```bash
# 1. 启动基础设施（MySQL / Redis / Etcd / DTM / Jaeger）
docker-compose -f deploy/docker/docker-compose.yaml up -d

# 2. 启动所有应用服务
docker-compose -f deploy/docker/docker-compose-app.yaml up -d
```

### 单服务 Docker 构建

```bash
# 构建镜像（多阶段构建，基础镜像锁定 digest）
docker build \
  --build-arg SERVICE=activity \
  --build-arg TYPE=rpc \
  -t campushub/activity-rpc:latest \
  -f deploy/docker/Dockerfile .
```

### CI/CD 流程

```
PR 提交 → lint-and-test（格式检查 / go vet / 编译 / 单测 + 竞态检测）
合并到 main → detect-changes → build-and-deploy
  ├── Docker 多阶段构建
  ├── 本地导入 K3s containerd（秒级就绪）
  ├── 异步推送 GHCR
  └── 更新部署仓库 → ArgoCD 自动同步
```

### 服务端口规划

| 服务 | HTTP | RPC | 说明 |
|------|------|-----|------|
| Nginx | 8888 | - | 统一入口 |
| User | 8001 | 9001 | 用户服务 |
| Activity | 8002 | 9002 | 活动服务 |
| Chat | 8003 | 9003 | 聊天服务 |

---

## 数据库设计

| 表名 | 说明 |
|------|------|
| `users` | 用户主表 |
| `activities` | 活动主表（含 version 乐观锁字段） |
| `categories` | 活动分类 |
| `activity_tags` | 活动-标签关联 |
| `activity_status_logs` | 状态变更审计日志 |
| `activity_tickets` | 报名票券（TOTP 动态码） |
| `check_in_records` | 签到记录（幂等键） |

> 完整建表语句见 [`deploy/sql/`](deploy/sql/)

---

## 文档

| 文档 | 说明 |
|------|------|
| [架构规范](docs/guide/架构规范说明.md) | 目录结构和调用规则 |
| [开发指南](docs/guide/开发指南.md) | 开发流程和规范 |
| [部署指南](deploy/README.md) | 部署操作手册 |

---

## Contributing

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交改动 (`git commit -m '添加新功能'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

---

## License

[MIT](LICENSE)
