# User API 服务

## 负责人

杨春路（B同学）

## 服务说明

user-api 是用户服务的 HTTP 接口层，负责：
- 用户注册、登录（签发 JWT Token）
- 用户信息查询和更新

## 目录结构

```
user/api/
├── desc/
│   └── user.api          # API 接口定义文件
├── etc/
│   └── user-api.yaml     # 配置文件
├── internal/
│   ├── config/           # 配置结构体（goctl 生成）
│   ├── handler/          # HTTP 处理器（goctl 生成）
│   ├── logic/            # 业务逻辑（需要你实现）
│   ├── svc/              # 服务上下文（goctl 生成）
│   └── types/            # 请求响应类型（goctl 生成）
├── user.go               # 入口文件
└── README.md
```

## 开发步骤

### 1. 生成代码

```bash
cd app/user/api
goctl api go -api desc/user.api -dir . -style go_zero
```

### 2. 修改配置

编辑 `etc/user-api.yaml`：
- 确保 `Auth.AccessSecret` 与其他 API 服务一致
- 配置 `UserRpc` 的 Etcd 地址

### 3. 实现业务逻辑

在 `internal/logic/` 目录下实现各接口的业务逻辑：

- `public/register_logic.go` - 实现注册逻辑
- `public/login_logic.go` - 实现登录逻辑，签发 JWT Token
- `user/get_profile_logic.go` - 获取用户信息
- `user/update_profile_logic.go` - 更新用户信息

### 4. 启动服务

```bash
go run user.go -f etc/user-api.yaml
```

## JWT 说明

- 登录成功后，在 `login_logic.go` 中签发 JWT Token
- Token 中必须包含 `userId` 字段（go-zero jwt 中间件约定）
- 所有 API 服务的 `Auth.AccessSecret` 必须一致

## 接口列表

| 方法 | 路径 | 说明 | 需要登录 |
|------|------|------|----------|
| POST | /api/v1/user/register | 用户注册 | 否 |
| POST | /api/v1/user/login | 用户登录 | 否 |
| GET | /api/v1/user/profile | 获取个人信息 | 是 |
| PUT | /api/v1/user/profile | 更新个人信息 | 是 |

## 端口

- HTTP: 8001
- 对应 RPC: 9001
