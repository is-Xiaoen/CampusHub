# Activity API 服务

## 负责人

Xiaoen（C/D同学）

## 服务说明

activity-api 是活动服务的 HTTP 接口层，负责：
- 活动 CRUD（创建、查询、更新、删除）
- 报名、签到
- 我的活动列表

## 目录结构

```
activity/api/
├── desc/
│   └── activity.api      # API 接口定义文件
├── etc/
│   └── activity-api.yaml # 配置文件
├── internal/
│   ├── config/           # 配置结构体（goctl 生成）
│   ├── handler/          # HTTP 处理器（goctl 生成）
│   ├── logic/            # 业务逻辑（需要你实现）
│   ├── svc/              # 服务上下文（goctl 生成）
│   └── types/            # 请求响应类型（goctl 生成）
├── activity.go           # 入口文件
└── README.md
```

## 开发步骤

### 1. 生成代码

```bash
cd app/activity/api
goctl api go -api desc/activity.api -dir . -style go_zero
```

### 2. 修改配置

编辑 `etc/activity-api.yaml`：
- **重要**：`Auth.AccessSecret` 必须与 user-api 一致
- 配置 `ActivityRpc` 的 Etcd 地址

### 3. 实现业务逻辑

在 `internal/logic/` 目录下实现各接口的业务逻辑。

### 4. 启动服务

```bash
go run activity.go -f etc/activity-api.yaml
```

## 接口列表

| 方法 | 路径 | 说明 | 需要登录 |
|------|------|------|----------|
| GET | /api/v1/activity/list | 活动列表 | 否 |
| GET | /api/v1/activity/:id | 活动详情 | 否 |
| POST | /api/v1/activity | 创建活动 | 是 |
| PUT | /api/v1/activity/:id | 更新活动 | 是 |
| DELETE | /api/v1/activity/:id | 删除活动 | 是 |
| POST | /api/v1/activity/:id/join | 报名活动 | 是 |
| POST | /api/v1/activity/:id/checkin | 签到 | 是 |
| GET | /api/v1/activity/my/joined | 我报名的活动 | 是 |
| GET | /api/v1/activity/my/created | 我创建的活动 | 是 |

## 端口

- HTTP: 8002
- 对应 RPC: 9002
